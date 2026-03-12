package server

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beer/xq/internal/feishu"
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

// isTradingTimeForSummary 判断当前是否为交易日 14:50（北京时间）
func isTradingTimeForSummary() bool {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	now := time.Now().In(loc)
	if now.Weekday() == time.Sunday || now.Weekday() == time.Saturday {
		return false
	}
	hour := now.Hour()
	minute := now.Minute()
	return hour == 14 && minute == 50
}

// runNotify 执行一次持仓对比与飞书消息提醒
func (s *Server) runNotify() {
	cfg := s.configStore.getNotify()
	if !cfg.Enabled {
		return
	}
	if !isTradingTime() {
		logger.Log.Printf("[notify] 非交易时段（仅交易日 9:00-15:00），跳过检查")
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

	var notifyLines []string
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
				logger.Log.Printf("[notify] 阈值触发: %s (diff=%.2f%%, threshold=%.1f%%)", line, diff, cfg.WeightThreshold)
				notifyLines = append(notifyLines, line)
			} else if lastMap[h.Symbol].Symbol != "" {
				// 只在有历史记录时记录未触发阈值的变化
				logger.Log.Printf("[notify] 变化但未达阈值: %s %s %.2f%%->%.2f%% (diff=%.2f%%, threshold=%.1f%%)", cubeSym, h.Symbol, prev, h.Weight, diff, cfg.WeightThreshold)
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
					logger.Log.Printf("[notify] 阈值触发(调出): %s (diff=-%.2f%%, threshold=%.1f%%)", line, h.Weight, cfg.WeightThreshold)
					notifyLines = append(notifyLines, line)
				} else {
					logger.Log.Printf("[notify] 调出但未达阈值: %s %s (原比例%.2f%%, threshold=%.1f%%)", cubeSym, h.Symbol, h.Weight, cfg.WeightThreshold)
				}
			}
		}

		if err := xueqiu.SaveSnapshot(xueqiu.SnapshotPath(dir, cubeSym), cur); err != nil {
			logger.Log.Printf("[notify] [%s] 保存快照失败: %v", cubeSym, err)
		}
	}

	sendMessage := func(subject, body string) {
		logger.Log.Printf("[notify] 准备发送飞书消息 subject=%q", subject)
		if cfg.FeishuAppID == "" || cfg.FeishuAppSecret == "" {
			logger.Log.Printf("[notify] 飞书 App ID 或 App Secret 未配置，跳过发送")
			return
		}
		fcfg := &feishu.Config{
			AppID:       cfg.FeishuAppID,
			AppSecret:   cfg.FeishuAppSecret,
			ReceiveID:   cfg.FeishuReceiveID,
			ReceiveType: cfg.FeishuReceiveType,
		}
		if err := fcfg.SendText(subject, body); err != nil {
			logger.Log.Printf("[notify] 发送飞书消息失败: %v", err)
			return
		}
		logger.Log.Printf("[notify] 飞书消息已发送: %s", subject)
	}

	logger.Log.Printf("[notify] 持仓变化 %d 条, 失败 %d 个组合, 阈值=%.1f%%", len(notifyLines), len(failLines), cfg.WeightThreshold)
	if len(notifyLines) > 0 {
		subject := fmt.Sprintf("雪球持仓变化提醒：%d 条比例变化≥%.0f%%", len(notifyLines), cfg.WeightThreshold)
		logger.Log.Printf("[notify] 发送飞书消息: 持仓变化提醒")
		sendMessage(subject, strings.Join(notifyLines, "\n"))
	}
	if len(failLines) > 0 {
		subject := fmt.Sprintf("雪球获取失败：%d 个组合", len(failLines))
		logger.Log.Printf("[notify] 发送飞书消息: 获取失败提醒")
		sendMessage(subject, strings.Join(failLines, "\n"))
	}
	if len(notifyLines) == 0 && len(failLines) == 0 {
		if cfg.WeightThreshold == 0 {
			logger.Log.Printf("[notify] 发送飞书消息: 无变化提醒 (阈值=0)")
			sendMessage("雪球持仓检查：无变化", "本次检查完成，持仓无变化。")
		} else {
			logger.Log.Printf("[notify] 无变化且阈值>0，不发消息")
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

// isYesterday 判断给定时间戳是否是昨天（北京时间）
func isYesterday(ts int64) bool {
	if ts == 0 {
		return false
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	t := time.Unix(ts, 0).In(loc)
	now := time.Now().In(loc)
	yesterday := now.AddDate(0, 0, -1)
	return t.Year() == yesterday.Year() && t.Month() == yesterday.Month() && t.Day() == yesterday.Day()
}

// runDailySummary 发送每日持仓汇总到飞书
func (s *Server) runDailySummary() {
	cfg := s.configStore.getNotify()
	if !cfg.Enabled {
		return
	}
	if !isTradingTimeForSummary() {
		return
	}
	logger.Log.Printf("[daily-summary] 开始生成每日持仓汇总…")
	if s.cfg.CookiesFile == "" {
		logger.Log.Printf("[daily-summary] 跳过：未配置 cookies 文件")
		return
	}
	if _, err := os.Stat(s.cfg.CookiesFile); err != nil {
		logger.Log.Printf("[daily-summary] Cookie 文件不存在: %s", s.cfg.CookiesFile)
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

	var summaryLines []string
	successCount := 0
	failCount := 0

	for _, cubeSym := range symbols {
		if cubeSym == "" {
			continue
		}
		cur, cubeName, err := xueqiu.FetchCubeViaAPI(cubeSym, s.cfg.CookiesFile)
		if err != nil {
			logger.Log.Printf("[daily-summary] [%s] 获取持仓失败: %v", cubeSym, err)
			failCount++
			continue
		}
		if len(cur.Holdings) == 0 {
			continue
		}
		successCount++

		displayName := names[cubeSym]
		if displayName == "" {
			displayName = cubeName
		}
		if displayName == "" {
			displayName = cubeSym
		}

		// 读取上次快照，是否是昨天的数据
		snapshotPath := xueqiu.SnapshotPath(dir, cubeSym)
		last, snapErr := xueqiu.LoadSnapshot(snapshotPath)

		// 构建昨天的持仓 map（只有当快照是昨天的才使用）
		var lastMap map[string]xueqiu.Holding
		hasYesterdayData := false
		if snapErr == nil && last != nil && isYesterday(last.UpdatedAt) {
			hasYesterdayData = true
			lastMap = make(map[string]xueqiu.Holding)
			for _, h := range last.Holdings {
				lastMap[h.Symbol] = h
			}
		}

		// 构建组合持仓明细
		cubeHeader := fmt.Sprintf("【%s (%s)】", displayName, cubeSym)
		summaryLines = append(summaryLines, cubeHeader)
		for _, h := range cur.Holdings {
			line := fmt.Sprintf("  %s %s: %.2f%%", h.Symbol, h.Name, h.Weight)
			if hasYesterdayData {
				if prev, ok := lastMap[h.Symbol]; ok {
					diff := h.Weight - prev.Weight
					if diff > 0 {
						line += fmt.Sprintf(" (昨日 %.2f%%, ↑+%.2f%%)", prev.Weight, diff)
					} else if diff < 0 {
						line += fmt.Sprintf(" (昨日 %.2f%%, ↓%.2f%%)", prev.Weight, -diff)
					} else {
						line += fmt.Sprintf(" (昨日 %.2f%%, 无变化)", prev.Weight)
					}
				} else {
					line += " (昨日未有，今日新增)"
				}
			}
			summaryLines = append(summaryLines, line)
		}
		// 检查是否昨日有但今日已调出的
		if hasYesterdayData {
			curMap := make(map[string]bool)
			for _, h := range cur.Holdings {
				curMap[h.Symbol] = true
			}
			removedAdded := false
			for sym, h := range lastMap {
				if !curMap[sym] {
					if !removedAdded {
						summaryLines = append(summaryLines, "  已调出:")
						removedAdded = true
					}
					summaryLines = append(summaryLines, fmt.Sprintf("    %s %s: 昨日 %.2f%%", sym, h.Name, h.Weight))
				}
			}
		}
		summaryLines = append(summaryLines, "") // 空行分隔
	}

	if failCount > 0 {
		summaryLines = append(summaryLines, fmt.Sprintf("注意: %d 个组合获取失败", failCount))
	}

	if len(summaryLines) > 0 {
		subject := fmt.Sprintf("雪球持仓日报 - %s", time.Now().Format("2006-01-02 14:50"))
		body := strings.Join(summaryLines, "\n")

		if cfg.FeishuAppID == "" || cfg.FeishuAppSecret == "" {
			logger.Log.Printf("[daily-summary] 飞书 App ID 或 App Secret 未配置，跳过发送")
			return
		}
		fcfg := &feishu.Config{
			AppID:       cfg.FeishuAppID,
			AppSecret:   cfg.FeishuAppSecret,
			ReceiveID:   cfg.FeishuReceiveID,
			ReceiveType: cfg.FeishuReceiveType,
		}
		if err := fcfg.SendText(subject, body); err != nil {
			logger.Log.Printf("[daily-summary] 发送飞书消息失败: %v", err)
			return
		}
		logger.Log.Printf("[daily-summary] 飞书消息已发送: %s (成功 %d 个组合, 失败 %d 个)", subject, successCount, failCount)
	} else {
		logger.Log.Printf("[daily-summary] 无持仓数据可发送")
	}
	logger.Log.Printf("[daily-summary] 汇总完成")
}

// startDailySummaryLoop 启动每日 15:50 持仓汇总循环
func (s *Server) startDailySummaryLoop() {
	go func() {
		for {
			s.runDailySummary()
			// 每分钟检查一次
			time.Sleep(time.Minute)
		}
	}()
	logger.Log.Printf("[daily-summary] 每日持仓汇总已启动（交易日 14:50）")
}
