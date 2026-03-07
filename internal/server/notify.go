package server

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beer/xq/internal/email"
	"github.com/beer/xq/internal/logger"
	"github.com/beer/xq/internal/xueqiu"
)

// isTradingTime 判断当前是否为交易日 9:00-15:00（北京时间）
func isTradingTime() bool {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	now := time.Now().In(loc)
	if now.Weekday() == time.Sunday || now.Weekday() == time.Saturday {
		return false
	}
	hour := now.Hour()
	return hour >= 9 && hour < 15
}

// runNotify 执行一次持仓对比与邮件提醒
func (s *Server) runNotify() {
	cfg := s.configStore.getNotify()
	if !cfg.Enabled {
		return
	}
	if !isTradingTime() {
		logger.Log.Printf("[notify] 非交易日 9:00-15:00，跳过检查")
		return
	}
	logger.Log.Printf("[notify] 开始检查…")
	if s.cfg.CookiesFile == "" {
		logger.Log.Printf("[notify] 跳过：未配置 cookies 文件")
		return
	}
	if _, err := os.Stat(s.cfg.CookiesFile); err != nil {
		logger.Log.Printf("[notify] Cookie 文件不存在: %s", s.cfg.CookiesFile)
		return
	}

	symbols, names, _ := xueqiu.LoadCubeSymbolsFromFile(s.cfg.CubesFile)
	if len(symbols) == 0 {
		symbols = []string{"ZH3347671"}
		if names == nil {
			names = make(map[string]string)
		}
	}

	dir := ""
	if home, err := os.UserHomeDir(); err == nil {
		dir = filepath.Join(home, ".xq_snapshots")
	}
	if dir == "" {
		return
	}

	var emailLines []string
	var failLines []string
	for _, cubeSym := range symbols {
		if cubeSym == "" {
			continue
		}
		cur, cubeName, err := xueqiu.FetchCubeViaAPI(cubeSym, s.cfg.CookiesFile)
		if err != nil {
			msg := fmt.Sprintf("[%s] 获取持仓失败: %v", cubeSym, err)
			logger.Log.Printf("[notify] %s", msg)
			failLines = append(failLines, msg)
			continue
		}
		if len(cur.Holdings) == 0 {
			continue
		}
		displayName := names[cubeSym]
		if displayName == "" {
			displayName = cubeName
		}
		if displayName == "" {
			displayName = cubeSym
		}
		logger.Log.Printf("[notify] 组合 %s (%s)", displayName, cubeSym)

		last, err := xueqiu.LoadSnapshot(xueqiu.SnapshotPath(dir, cubeSym))
		if err != nil {
			logger.Log.Printf("[notify] [%s] 读取上次快照失败: %v", cubeSym, err)
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
			if math.Abs(diff) >= cfg.WeightThreshold {
				line := fmt.Sprintf("[%s] %s %s 比例: %.2f%% -> %.2f%% (变化 %+.2f%%)", cubeSym, h.Symbol, h.Name, prev, h.Weight, diff)
				emailLines = append(emailLines, line)
			}
		}
		if last != nil {
			for _, h := range last.Holdings {
				if _, ok := curMap[h.Symbol]; ok {
					continue
				}
				diff := -h.Weight
				if math.Abs(diff) >= cfg.WeightThreshold {
					line := fmt.Sprintf("[%s] %s %s 比例: %.2f%% -> 0%% (已调出)", cubeSym, h.Symbol, h.Name, h.Weight)
					emailLines = append(emailLines, line)
				}
			}
		}

		if err := xueqiu.SaveSnapshot(xueqiu.SnapshotPath(dir, cubeSym), cur); err != nil {
			logger.Log.Printf("[notify] [%s] 保存快照失败: %v", cubeSym, err)
		}
	}

	sendMail := func(subject, body string) {
		logger.Log.Printf("[notify] 准备发邮件 subject=%q", subject)
		ecfg, err := email.LoadFromHome()
		if err != nil {
			logger.Log.Printf("[notify] 读取邮箱配置 $HOME/.email 失败: %v", err)
			return
		}
		logger.Log.Printf("[notify] 邮箱配置: host=%s port=%d from=%s allow_plain=%v", ecfg.SMTPHost, ecfg.SMTPPort, ecfg.From, ecfg.AllowPlain)
		if ecfg.SMTPHost == "" || ecfg.From == "" || ecfg.Password == "" {
			logger.Log.Printf("[notify] 邮箱配置不完整，需设置 smtp_host、from、password")
			return
		}
		toSend := cfg.EmailTo
		if toSend == "" && len(ecfg.To) == 0 {
			logger.Log.Printf("[notify] 未指定收件人：请在页面配置 email_to 或在 $HOME/.email 中配置 to")
			return
		}
		if toSend == "" {
			toSend = strings.Join(ecfg.To, ",")
		}
		logger.Log.Printf("[notify] 收件人: %s", toSend)
		if err := ecfg.Send(toSend, subject, body); err != nil {
			logger.Log.Printf("[notify] 发送邮件失败: %v", err)
			return
		}
		logger.Log.Printf("[notify] 邮件已发送: %s", subject)
	}

	logger.Log.Printf("[notify] 持仓变化 %d 条, 失败 %d 个组合, 阈值=%.1f%%", len(emailLines), len(failLines), cfg.WeightThreshold)
	if len(emailLines) > 0 {
		subject := fmt.Sprintf("雪球持仓变化提醒：%d 条比例变化≥%.0f%%", len(emailLines), cfg.WeightThreshold)
		logger.Log.Printf("[notify] 发送邮件: 持仓变化提醒")
		sendMail(subject, strings.Join(emailLines, "\n"))
	}
	if len(failLines) > 0 {
		subject := fmt.Sprintf("雪球获取失败：%d 个组合", len(failLines))
		logger.Log.Printf("[notify] 发送邮件: 获取失败提醒")
		sendMail(subject, strings.Join(failLines, "\n"))
	}
	if len(emailLines) == 0 && len(failLines) == 0 {
		if cfg.WeightThreshold == 0 {
			logger.Log.Printf("[notify] 发送邮件: 无变化提醒 (阈值=0)")
			sendMail("雪球持仓检查：无变化", "本次检查完成，持仓无变化。")
		} else {
			logger.Log.Printf("[notify] 无变化且阈值>0，不发邮件")
		}
	}
	logger.Log.Printf("[notify] 检查完成")
}

// startNotifyLoop 启动定时提醒循环，间隔从配置读取
func (s *Server) startNotifyLoop() {
	go func() {
		for {
			s.runNotify()
			cfg := s.configStore.getNotify()
			interval := time.Duration(cfg.IntervalMinutes) * time.Minute
			if interval < time.Minute {
				interval = time.Minute
			}
			time.Sleep(interval)
		}
	}()
	cfg := s.configStore.getNotify()
	interval := time.Duration(cfg.IntervalMinutes) * time.Minute
	if interval < time.Minute {
		interval = time.Minute
	}
	logger.Log.Printf("[notify] 定时提醒已启动，间隔 %v", interval)
}
