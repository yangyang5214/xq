package xueqiu

import (
	"bufio"
	"os"
	"strings"
)

// LoadCubeSymbolsFromFile 从文件读取组合 symbol 列表，每行一个，支持 # 注释与空行。
func LoadCubeSymbolsFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var symbols []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 新格式：symbol 在前，# 后为注释（名称）
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line != "" {
			symbols = append(symbols, line)
		}
	}
	return symbols, sc.Err()
}
