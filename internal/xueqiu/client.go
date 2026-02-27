package xueqiu

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	baseURL     = "https://xueqiu.com"
	historyPath = "/cubes/rebalancing/history.json"
	originPath  = "/cubes/rebalancing/show_origin.json"
	userAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// Client 雪球 API 客户端，带 Cookie 认证、重试与限速
type Client struct {
	http    *http.Client
	cookie  string
	limiter *time.Ticker
}

// NewClient 使用 Cookie 创建客户端。cookie 可为完整 Cookie 或仅 xq_a_token=xxx
func NewClient(cookie string) *Client {
	cookie = strings.TrimSpace(cookie)
	if cookie != "" && !strings.Contains(cookie, "=") {
		cookie = "xq_a_token=" + cookie
	}
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        2,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
			},
		},
		cookie:  cookie,
		limiter: time.NewTicker(400 * time.Millisecond), // 限速，避免 429
	}
}

func (c *Client) do(method, path string, params url.Values) ([]byte, int, error) {
	<-c.limiter.C
	rawURL := baseURL + path
	if len(params) > 0 {
		rawURL += "?" + params.Encode()
	}
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	req.Header.Set("Referer", baseURL+"/")

	var resp *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = c.http.Do(req)
		if err != nil {
			if attempt < 2 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
			}
			continue
		}
		break
	}
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// GetHistory 拉取组合调仓历史（分页）
func (c *Client) GetHistory(cubeSymbol string, page, count int) ([]byte, int, error) {
	params := url.Values{}
	params.Set("cube_symbol", cubeSymbol)
	if page <= 0 {
		page = 1
	}
	if count <= 0 || count > 50 {
		count = 20
	}
	params.Set("page", strconv.Itoa(page))
	params.Set("count", strconv.Itoa(count))
	return c.do(http.MethodGet, historyPath, params)
}

// GetShowOrigin 拉取单次调仓详情（需 rb_id）
func (c *Client) GetShowOrigin(cubeSymbol string, rbID int64) ([]byte, int, error) {
	params := url.Values{}
	params.Set("cube_symbol", cubeSymbol)
	params.Set("rb_id", strconv.FormatInt(rbID, 10))
	return c.do(http.MethodGet, originPath, params)
}
