package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// NotifyConfig 页面可配置的提醒参数
type NotifyConfig struct {
	Enabled         bool    `json:"enabled"`
	EmailTo         string  `json:"email_to"`
	WeightThreshold float64 `json:"weight_threshold"`
	IntervalMinutes int     `json:"interval_minutes"` // 检查间隔（分钟）
}

// defaultNotifyConfig 默认提醒配置
func defaultNotifyConfig() NotifyConfig {
	return NotifyConfig{
		Enabled:         false,
		EmailTo:         "",
		WeightThreshold: 5,
		IntervalMinutes: 30,
	}
}

// configPath 返回配置文件路径
func configPath() string {
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
	data, err := os.ReadFile(configPath())
	if err != nil {
		c.notify = defaultNotifyConfig()
		return
	}
	var raw struct {
		Notify *NotifyConfig `json:"notify"`
	}
	if json.Unmarshal(data, &raw) == nil && raw.Notify != nil {
		c.notify = *raw.Notify
		if c.notify.WeightThreshold < 0 {
			c.notify.WeightThreshold = 5
		}
		if c.notify.IntervalMinutes < 1 {
			c.notify.IntervalMinutes = 30
		}
	} else {
		c.notify = defaultNotifyConfig()
	}
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
	return os.WriteFile(path, data, 0600)
}

func (c *configStore) getNotify() NotifyConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.notify
}
