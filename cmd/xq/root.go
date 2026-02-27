package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/beer/xq/internal/email"
	"github.com/beer/xq/internal/xueqiu"
	"github.com/spf13/cobra"
)

var (
	cubesFile       string
	page            int
	count           int
	cookiesFile     string
	emailTo         string
	weightThreshold float64
)

var rootCmd = &cobra.Command{
	Use:   "xq",
	Short: "雪球组合调仓历史与详情查询",
	Long:  `从 cookies.txt 与组合列表拉取雪球组合的调仓历史，可选拉取单次调仓详情。`,
	RunE:  runRoot,
}

func init() {
	rootCmd.Flags().StringVarP(&cubesFile, "cubes-file", "f", "cubes.txt", "组合列表文件路径，每行一个 symbol，支持 # 注释")
	rootCmd.Flags().IntVarP(&page, "page", "p", 1, "调仓历史页码")
	rootCmd.Flags().IntVarP(&count, "count", "c", 20, "每页条数(1-50)")
	rootCmd.Flags().StringVarP(&cookiesFile, "cookies-file", "", "cookies.txt", "Get cookies.txt LOCALLY 导出的 cookies.txt 路径")
	rootCmd.Flags().StringVarP(&emailTo, "to", "t", "", "收件人邮箱（多个用逗号分隔）")
	rootCmd.Flags().Float64VarP(&weightThreshold, "weight-threshold", "w", 5, "比例变化阈值(%)，超过此值才发邮件提醒")
}

func runRoot(cmd *cobra.Command, args []string) error {
	cookie, _ := xueqiu.LoadCookieFromTxt(cookiesFile)
	if cookie == "" {
		log.Fatalf("请提供 Cookie：在项目目录放 cookies.txt（或 -cookies-file 指定路径）")
	}
	client := xueqiu.NewClient(cookie)

	cubeSymbols, _ := xueqiu.LoadCubeSymbolsFromFile(cubesFile)
	if len(cubeSymbols) == 0 {
		cubeSymbols = []string{"ZH3347671"}
	}

	var emailLines []string
	var failLines []string
	for _, cubeSym := range cubeSymbols {
		if cubeSym == "" {
			continue
		}
		body, code, err := client.GetHistory(cubeSym, page, count)
		if err != nil {
			msg := fmt.Sprintf("[%s] 获取调仓历史失败: %v", cubeSym, err)
			log.Printf("%s", msg)
			failLines = append(failLines, msg)
			continue
		}
		if code != http.StatusOK {
			msg := fmt.Sprintf("[%s] HTTP %d: %s", cubeSym, code, string(body))
			log.Printf("%s", msg)
			failLines = append(failLines, msg)
			continue
		}

		//log.Printf("%v", string(body))

		hist, err := xueqiu.ParseHistory(body)
		if err != nil {
			msg := fmt.Sprintf("[%s] 解析调仓历史失败: %v", cubeSym, err)
			log.Printf("%s", msg)
			failLines = append(failLines, msg)
			continue
		}

		now := time.Now()
		todayY, todayM, todayD := now.Date()
		var todayList []xueqiu.RebalanceItem
		for _, item := range hist.List {
			ts := time.UnixMilli(item.UpdatedAt)
			y, m, d := ts.Date()
			if y == todayY && m == todayM && d == todayD {
				todayList = append(todayList, item)
			}
		}

		if len(todayList) == 0 {
			fmt.Printf("\n组合 %s 当天无调仓记录\n", cubeSym)
			continue
		}
		fmt.Printf("\n组合 %s 当天调仓 (共 %d 条)\n", cubeSym, len(todayList))
		for _, item := range todayList {
			for _, s := range item.RebalancingHistories {
				fmt.Printf("%s %s 成交价:%.2f 比例:%.2f%% -> %.2f%%\n", s.StockSymbol, s.StockName, s.Price, s.PrevWeightAdjusted, s.PrevTargetWeight)
				if math.Abs(s.PrevWeightAdjusted-s.PrevTargetWeight) >= weightThreshold {
					line := fmt.Sprintf("[%s] %s %s 成交价:%.2f 比例:%.2f%% -> %.2f%%", cubeSym, s.StockSymbol, s.StockName, s.Price, s.PrevWeightAdjusted, s.PrevTargetWeight)
					emailLines = append(emailLines, line)
				}
			}
		}
	}

	toSend := emailTo
	sendMail := func(subject, body string) {
		cfg, err := email.LoadFromHome()
		if err != nil {
			log.Printf("读取邮箱配置 $HOME/.email 失败: %v", err)
			return
		}
		if cfg.SMTPHost == "" || cfg.From == "" || cfg.Password == "" {
			log.Printf("邮箱配置不完整，需设置 smtp_host、from、password")
			return
		}
		if toSend == "" && len(cfg.To) == 0 {
			log.Printf("未指定收件人：请使用 -t 或在校验 .email 中配置 to")
			return
		}
		if err := cfg.Send(toSend, subject, body); err != nil {
			log.Printf("发送邮件失败: %v", err)
			return
		}
		if toSend != "" {
			log.Printf("已发送邮件至 %s", toSend)
		} else {
			log.Printf("已发送邮件至 %s", strings.Join(cfg.To, ", "))
		}
	}
	if len(emailLines) > 0 {
		subject := fmt.Sprintf("雪球调仓提醒：%d 条比例变化≥%.0f%%", len(emailLines), weightThreshold)
		sendMail(subject, strings.Join(emailLines, "\n"))
	}
	if len(failLines) > 0 {
		subject := fmt.Sprintf("雪球获取失败：%d 个组合", len(failLines))
		sendMail(subject, strings.Join(failLines, "\n"))
	}
	return nil
}
