package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/beer/xq/internal/email"
	"github.com/beer/xq/internal/xueqiu"
	"github.com/spf13/cobra"
)

var (
	cubesFile       string
	cookiesFile     string
	emailTo         string
	weightThreshold float64
	snapshotDir     string
)

var rootCmd = &cobra.Command{
	Use:   "xq",
	Short: "雪球组合持仓分布对比与提醒",
	Long:  `从 cookies.txt 与组合列表拉取雪球组合当前持仓分布，与上次快照对比；比例变化超过阈值时发邮件提醒。`,
	RunE:  runRoot,
}

func init() {
	rootCmd.Flags().StringVarP(&cubesFile, "cubes-file", "f", "cubes.txt", "组合列表文件路径，每行一个 symbol，支持 # 注释")
	rootCmd.Flags().StringVarP(&cookiesFile, "cookies-file", "", "cookies.txt", "Get cookies.txt LOCALLY 导出的 cookies.txt 路径")
	rootCmd.Flags().StringVarP(&emailTo, "to", "t", "", "收件人邮箱（多个用逗号分隔）")
	rootCmd.Flags().Float64VarP(&weightThreshold, "weight-threshold", "w", 5, "持仓比例变化阈值(%)，超过此值才发邮件提醒")
	rootCmd.Flags().StringVarP(&snapshotDir, "snapshot-dir", "s", "", "持仓快照目录，默认 $HOME/.xq_snapshots")
}

func runRoot(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(cookiesFile); err != nil {
		log.Fatalf("请提供 Cookie 文件：在项目目录放 cookies.txt（或 -cookies-file 指定路径）")
	}
	cubeSymbols, cubeNames, _ := xueqiu.LoadCubeSymbolsFromFile(cubesFile)
	if len(cubeSymbols) == 0 {
		cubeSymbols = []string{"ZH3347671"}
		cubeNames = make(map[string]string)
	}

	dir := snapshotDir
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".xq_snapshots")
	}

	var emailLines []string
	var failLines []string
	for _, cubeSym := range cubeSymbols {
		if cubeSym == "" {
			continue
		}
		cur, cubeName, err := xueqiu.FetchCubeViaAPI(cubeSym, cookiesFile)
		if err != nil {
			msg := fmt.Sprintf("[%s] 获取持仓失败: %v", cubeSym, err)
			log.Printf("%s", msg)
			failLines = append(failLines, msg)
			continue
		}
		if len(cur.Holdings) == 0 {
			fmt.Printf("\n组合 %s 无持仓数据\n", cubeSym)
			continue
		}
		displayName := cubeNames[cubeSym]
		if displayName == "" {
			displayName = cubeName
		}
		if displayName == "" {
			displayName = cubeSym
		}
		fmt.Printf("\n组合名: %s (%s)\n", displayName, cubeSym)
		fmt.Println("组合仓位详情:")

		last, err := xueqiu.LoadSnapshot(xueqiu.SnapshotPath(dir, cubeSym))
		if err != nil {
			log.Printf("[%s] 读取上次快照失败: %v", cubeSym, err)
		}

		lastMap := make(map[string]xueqiu.Holding)
		if last != nil {
			for _, h := range last.Holdings {
				lastMap[h.Symbol] = h
			}
		}

		curMap := make(map[string]xueqiu.Holding)
		for _, h := range cur.Holdings {
			curMap[h.Symbol] = h
			prev := lastMap[h.Symbol].Weight
			diff := h.Weight - prev
			if math.Abs(diff) >= weightThreshold {
				line := fmt.Sprintf("[%s] %s %s 比例: %.2f%% -> %.2f%% (变化 %+.2f%%)", cubeSym, h.Symbol, h.Name, prev, h.Weight, diff)
				emailLines = append(emailLines, line)
			}
			fmt.Printf("  %s %s %.2f%%", h.Symbol, h.Name, h.Weight)
			if last != nil {
				fmt.Printf(" (上次 %.2f%%, 变化 %+.2f%%)", prev, diff)
			}
			fmt.Println()
		}
		// 上次有、本次无的标的（视为比例变为 0）
		if last != nil {
			for _, h := range last.Holdings {
				if _, ok := curMap[h.Symbol]; ok {
					continue
				}
				diff := -h.Weight
				if math.Abs(diff) >= weightThreshold {
					line := fmt.Sprintf("[%s] %s %s 比例: %.2f%% -> 0%% (已调出)", cubeSym, h.Symbol, h.Name, h.Weight)
					emailLines = append(emailLines, line)
				}
			}
		}

		if err := xueqiu.SaveSnapshot(xueqiu.SnapshotPath(dir, cubeSym), cur); err != nil {
			log.Printf("[%s] 保存快照失败: %v", cubeSym, err)
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
		subject := fmt.Sprintf("雪球持仓变化提醒：%d 条比例变化≥%.0f%%", len(emailLines), weightThreshold)
		sendMail(subject, strings.Join(emailLines, "\n"))
	}
	if len(failLines) > 0 {
		subject := fmt.Sprintf("雪球获取失败：%d 个组合", len(failLines))
		sendMail(subject, strings.Join(failLines, "\n"))
	}
	return nil
}
