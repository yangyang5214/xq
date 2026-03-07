package email

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 从 $HOME/.email 读取的邮箱配置（JSON）
// 格式示例：{"password":"...","smtp_host":"smtp.126.com","smtp_port":25,"from":"...","to":["..."]}
// smtp_port 为 25 时，若服务器不支持 STARTTLS，可设置 "allow_plain": true 使用明文认证
type Config struct {
	Password   string   `json:"password"`
	SMTPHost   string   `json:"smtp_host"`
	SMTPPort   int      `json:"smtp_port"`
	From       string   `json:"from"`
	To         []string `json:"to"`
	AllowPlain bool     `json:"allow_plain"` // 端口 25 且服务器不支持 STARTTLS 时设为 true
}

// LoadFromHome 从 $HOME/.email 读取 JSON 配置
func LoadFromHome() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".email")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.SMTPPort == 0 {
		cfg.SMTPPort = 25
	}
	return &cfg, nil
}
