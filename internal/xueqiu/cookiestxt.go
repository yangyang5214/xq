package xueqiu

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// CookieDomain 雪球 Cookie 所属域名
const CookieDomain = "xueqiu.com"

// CookieEntry 单条 Cookie（供 chromedp 等设置用）
type CookieEntry struct {
	Domain string
	Path   string
	Name   string
	Value  string
	Expiry int64
}

// LoadCookieEntriesFromTxt 从 Netscape cookies.txt 读取雪球域名的 Cookie 列表（未过滤过期）
func LoadCookieEntriesFromTxt(path string) ([]CookieEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var list []CookieEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			continue
		}
		domain := strings.TrimSpace(parts[0])
		path := strings.TrimSpace(parts[2])
		expiryStr := strings.TrimSpace(parts[4])
		name := strings.TrimSpace(parts[5])
		value := strings.TrimSpace(parts[6])
		if name == "" || !domainMatch(domain, CookieDomain) {
			continue
		}
		exp, _ := strconv.ParseInt(expiryStr, 10, 64)
		list = append(list, CookieEntry{Domain: domain, Path: path, Name: name, Value: value, Expiry: exp})
	}
	return list, sc.Err()
}

// LoadCookieFromTxt 从 Get cookies.txt LOCALLY 等插件导出的 Netscape cookies.txt 中
// 读取指定域名的 Cookie，拼成 Cookie 请求头用的字符串。
// 会过滤已过期的 cookie。
func LoadCookieFromTxt(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	now := time.Now().Unix()
	var pairs []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			continue
		}
		domain := strings.TrimSpace(parts[0])
		path := strings.TrimSpace(parts[2])
		expiryStr := strings.TrimSpace(parts[4])
		name := strings.TrimSpace(parts[5])
		value := strings.TrimSpace(parts[6])
		if name == "" {
			continue
		}
		if !domainMatch(domain, CookieDomain) {
			continue
		}
		_ = path
		if expiryStr != "" && expiryStr != "0" {
			exp, _ := strconv.ParseInt(expiryStr, 10, 64)
			if exp > 0 && now > exp {
				continue
			}
		}
		pairs = append(pairs, name+"="+value)
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return strings.Join(pairs, "; "), nil
}

func domainMatch(cookieDomain, want string) bool {
	cookieDomain = strings.TrimPrefix(cookieDomain, ".")
	want = strings.TrimPrefix(want, ".")
	return cookieDomain == want || strings.HasSuffix(cookieDomain, "."+want)
}
