package xueqiu

import (
	"bufio"
	"os"
	"strings"
)

// LoadCubeSymbolsFromFile 从文件读取组合 symbol 列表，每行一个，支持 # 注释（# 后为组合显示名）。返回 symbols 与 symbol -> 显示名 的映射。
func LoadCubeSymbolsFromFile(path string) (symbols []string, names map[string]string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	names = make(map[string]string)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sym := line
		if i := strings.Index(line, "#"); i >= 0 {
			sym = strings.TrimSpace(line[:i])
			if namePart := strings.TrimSpace(line[i+1:]); namePart != "" {
				names[sym] = namePart
			}
		}
		if sym != "" {
			symbols = append(symbols, sym)
		}
	}
	return symbols, names, sc.Err()
}
