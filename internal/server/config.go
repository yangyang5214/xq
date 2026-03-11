package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/beer/xq/internal/logger"
)

// NotifyConfig 页面可配置的提醒参数
type NotifyConfig struct {
	Enabled           bool    `json:"enabled"`
	FeishuAppID       string  `json:"feishu_app_id"`      // 飞书应用 App ID
	FeishuAppSecret   string  `json:"feishu_app_secret"`  // 飞书应用 App Secret
	FeishuReceiveID   string  `json:"feishu_receive_id"`   // 接收者 ID（用户ID或群ID）- 主动推送时使用
	FeishuReceiveType string `json:"feishu_receive_type"` // 接收者类型：open_id、user_id、union_id、chat_id
	WeightThreshold   float64 `json:"weight_threshold"`
	IntervalMinutes   int     `json:"interval_minutes"` // 检查间隔（分钟）
}

// defaultNotifyConfig 默认提醒配置
func defaultNotifyConfig() NotifyConfig {
	return NotifyConfig{
		Enabled:           false,
		FeishuAppID:       "",
		FeishuAppSecret:   "",
		FeishuReceiveID:   "",
		FeishuReceiveType: "open_id",
		WeightThreshold:   5,
		IntervalMinutes:   30,
	}
}

// configPath 返回配置文件路径
// 优先使用环境变量 XQ_CONFIG，否则用 $HOME/.xq_config.json
func configPath() string {
	if p := os.Getenv("XQ_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".xq_config.json")
}

// configStore 配置存储，支持并发读写
type configStore struct {
	mu     sync.RWMutex
	notify NotifyConfig
}

func (c *configStore) load() {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Log.Printf("[config] 读取配置失败 path=%s err=%v，使用默认配置", path, err)
		c.notify = defaultNotifyConfig()
		return
	}
	var raw struct {
		Notify *NotifyConfig `json:"notify"`
	}
	if json.Unmarshal(data, &raw) != nil || raw.Notify == nil {
		logger.Log.Printf("[config] 解析配置失败 %s，使用默认配置", path)
		c.notify = defaultNotifyConfig()
		return
	}
	c.notify = *raw.Notify
	if c.notify.WeightThreshold < 0 {
		c.notify.WeightThreshold = 5
	}
	if c.notify.IntervalMinutes < 1 {
		c.notify.IntervalMinutes = 30
	}
	logger.Log.Printf("[config] 已加载 path=%s enabled=%v interval=%dm threshold=%.1f%%", path, c.notify.Enabled, c.notify.IntervalMinutes, c.notify.WeightThreshold)
}

func (c *configStore) save(notify NotifyConfig) error {
	c.mu.Lock()
	c.notify = notify
	c.mu.Unlock()
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(map[string]interface{}{"notify": notify}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	logger.Log.Printf("[config] 已保存配置 path=%s", path)
	return nil
}

func (c *configStore) getNotify() NotifyConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.notify
}
