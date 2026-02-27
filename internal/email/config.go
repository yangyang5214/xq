package email

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 从 $HOME/.email 读取的邮箱配置（JSON）
// 格式示例：{"password":"...","smtp_host":"smtp.126.com","smtp_port":25,"from":"...","to":["..."]}
type Config struct {
	Password string   `json:"password"`
	SMTPHost string   `json:"smtp_host"`
	SMTPPort int      `json:"smtp_port"`
	From     string   `json:"from"`
	To       []string `json:"to"`
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
