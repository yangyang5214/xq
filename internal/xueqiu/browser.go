package xueqiu

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const cubePageTimeout = 25 * time.Second

// FetchCubeViaBrowser 用真实 Chrome 打开组合页，在页面内执行 JS 取出 SNB.cubeTreeData 与组合名，
// 返回解析后的持仓快照与组合名。需本机已安装 Chrome/Chromium。
// headless 为 true 时不显示窗口，为 false 时可视化运行。
func FetchCubeViaBrowser(cubeSymbol, cookiesTxtPath string, headless bool) (snapshot *HoldingsSnapshot, cubeName string, err error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, cubePageTimeout)
	defer cancel()

	url := "https://xueqiu.com/P/" + cubeSymbol
	var nameStr string

	tasks := chromedp.Tasks{network.Enable()}
	// 先注入 cookies，再访问组合页
	entries, _ := LoadCookieEntriesFromTxt(cookiesTxtPath)
	if len(entries) > 0 {
		tasks = append(tasks, chromedp.ActionFunc(func(ctx context.Context) error {
			exp := cdp.TimeSinceEpoch(time.Now().Add(365 * 24 * time.Hour))
			for _, e := range entries {
				domain := e.Domain
				if domain != "" && domain[0] != '.' && domain != "xueqiu.com" {
					domain = "." + domain
				}
				if domain == "" {
					domain = ".xueqiu.com"
				}
				_ = network.SetCookie(e.Name, e.Value).
					WithExpires(&exp).
					WithDomain(domain).
					WithPath(e.Path).
					Do(ctx)
			}
			return nil
		}))
	}
	tasks = append(tasks,
		chromedp.Navigate(url),
		chromedp.Sleep(1*time.Second),
	)
	if err := chromedp.Run(ctx, tasks); err != nil {
		return nil, "", fmt.Errorf("chromedp: %w", err)
	}

	var rawResult string
	// 轮询直到 SNB.cubeTreeData 或 view_rebalancing.holdings 就绪（最多约 12 秒）
	for wait := 0; wait < 24; wait++ {
		_ = chromedp.Run(ctx, chromedp.Sleep(500*time.Millisecond))
		_ = chromedp.Run(ctx, chromedp.Evaluate(`(function(){
			if (typeof SNB === 'undefined') return '';
			if (SNB.cubeTreeData && Array.isArray(SNB.cubeTreeData) && SNB.cubeTreeData.length > 0)
				return 'tree:' + JSON.stringify(SNB.cubeTreeData);
			if (SNB.cubeInfo && SNB.cubeInfo.view_rebalancing && SNB.cubeInfo.view_rebalancing.holdings && SNB.cubeInfo.view_rebalancing.holdings.length > 0)
				return 'holdings:' + JSON.stringify(SNB.cubeInfo.view_rebalancing.holdings);
			return '';
		})()`, &rawResult))
		if rawResult != "" {
			break
		}
	}

	if rawResult == "" {
		return nil, "", fmt.Errorf("页面内未找到 SNB.cubeTreeData 或 view_rebalancing.holdings（可稍后重试或检查登录）")
	}

	// 取组合名
	_ = chromedp.Run(ctx, chromedp.Evaluate(`(function(){
		try {
			if (typeof cubeName !== 'undefined' && cubeName) return cubeName;
			if (typeof SNB !== 'undefined' && SNB.cubeInfo && SNB.cubeInfo.name) return SNB.cubeInfo.name;
			return '';
		} catch(e) { return ''; }
	})()`, &nameStr))
	if nameStr != "" {
		cubeName = nameStr
	}

	// 解析 tree:... 或 holdings:...
	if len(rawResult) > 6 && rawResult[:6] == "tree:" {
		var tree cubeTreeNodes
		if err := json.Unmarshal([]byte(rawResult[5:]), &tree); err != nil {
			return nil, "", fmt.Errorf("解析 cubeTreeData: %w", err)
		}
		snapshot = &HoldingsSnapshot{CubeSymbol: cubeSymbol, Holdings: make([]Holding, 0)}
		tree.flattenInto(&snapshot.Holdings)
	} else if len(rawResult) > 9 && rawResult[:9] == "holdings:" {
		snapshot, err = parseHoldingsFromViewRebalancing([]byte(rawResult[9:]), cubeSymbol)
		if err != nil {
			return nil, "", err
		}
	} else {
		return nil, "", fmt.Errorf("无法解析页面数据: %s", rawResult[:min(80, len(rawResult))])
	}
	if len(snapshot.Holdings) == 0 {
		return nil, "", fmt.Errorf("持仓列表为空")
	}
	return snapshot, cubeName, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// viewRebalancingHolding 与 SNB.cubeInfo.view_rebalancing.holdings[] 对齐
type viewRebalancingHolding struct {
	StockName   string  `json:"stock_name"`
	StockSymbol string  `json:"stock_symbol"`
	Symbol      string  `json:"symbol"`
	Weight      float64 `json:"weight"`
}

func parseHoldingsFromViewRebalancing(data []byte, cubeSymbol string) (*HoldingsSnapshot, error) {
	var list []viewRebalancingHolding
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("解析 view_rebalancing.holdings: %w", err)
	}
	s := &HoldingsSnapshot{CubeSymbol: cubeSymbol, Holdings: make([]Holding, 0, len(list))}
	for _, h := range list {
		sym := h.StockSymbol
		if sym == "" {
			sym = h.Symbol
		}
		if sym == "" {
			continue
		}
		s.Holdings = append(s.Holdings, Holding{Symbol: sym, Name: h.StockName, Weight: h.Weight})
	}
	return s, nil
}
