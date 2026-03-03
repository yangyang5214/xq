package xueqiu

import (
	"fmt"
)

// FetchCubeViaAPI 用 easytrader 同款 HTTP API 获取组合持仓（/cubes/rebalancing/current.json），
// 无需浏览器。cookie 从 cookies.txt 读取。参见 https://github.com/shidenggui/easytrader
func FetchCubeViaAPI(cubeSymbol, cookiesTxtPath string) (*HoldingsSnapshot, string, error) {
	cookie, err := LoadCookieFromTxt(cookiesTxtPath)
	if err != nil {
		return nil, "", fmt.Errorf("读取 cookies: %w", err)
	}
	if cookie == "" {
		return nil, "", fmt.Errorf("cookies.txt 中无雪球域名 cookie")
	}
	client := NewClient(cookie)
	body, status, err := client.GetRebalancingCurrent(cubeSymbol)
	if err != nil {
		return nil, "", fmt.Errorf("请求 current.json: %w", err)
	}
	if status != 200 {
		return nil, "", fmt.Errorf("current.json 返回 %d", status)
	}
	snapshot, err := ParseCurrentRebalancing(body, cubeSymbol)
	if err != nil {
		return nil, "", err
	}
	return snapshot, "", nil
}
