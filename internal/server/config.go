package server

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/beer/xq/internal/logger"
)

// NotifyConfig 页面可配置的提醒参数
type NotifyConfig struct {
	Enabled           bool    `json:"enabled"`
	FeishuAppID       string  `json:"feishu_app_id"`      // 飞书应用 App ID
	FeishuAppSecret   string  `json:"feishu_app_secret"`  // 飞书应用 App Secret
	FeishuReceiveID   string  `json:"feishu_receive_id"`   // 接收者 ID（用户ID或群ID）- 主动推送时使用
	FeishuReceiveType string  `json:"feishu_receive_type"` // 接收者类型：open_id、user_id、union_id、chat_id
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

// EnvPath 返回 .env 文件路径
// 优先使用环境变量 XQ_ENV，否则用当前目录的 .env
func EnvPath() string {
	if p := os.Getenv("XQ_ENV"); p != "" {
		return p
	}
	return ".env"
}

// LoadEnvFile 加载 .env 文件到 map
func LoadEnvFile(path string) map[string]string {
	result := make(map[string]string)
	if path == "" {
		return result
	}
	file, err := os.Open(path)
	if err != nil {
		logger.Log.Printf("[config] 读取 .env 文件失败 path=%s err=%v", path, err)
		return result
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			// 去除引号
			if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
				val = val[1 : len(val)-1]
			} else if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
				val = val[1 : len(val)-1]
			}
			result[key] = val
		}
	}
	return result
}

// GetEnvStr 从 env map 中获取字符串
func GetEnvStr(env map[string]string, key, defaultValue string) string {
	if val, ok := env[key]; ok && val != "" {
		return val
	}
	return defaultValue
}

// getEnvBool 从 env map 中获取布尔值
func getEnvBool(env map[string]string, key string, defaultValue bool) bool {
	val := env[key]
	if val == "" {
		return defaultValue
	}
	return strings.ToLower(val) == "true" || val == "1" || val == "yes"
}

// getEnvFloat 从 env map 中获取浮点值
func getEnvFloat(env map[string]string, key string, defaultValue float64) float64 {
	val := env[key]
	if val == "" {
		return defaultValue
	}
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return f
	}
	logger.Log.Printf("[config] .env 中 %s=%s 解析失败，使用默认值 %.1f", key, val, defaultValue)
	return defaultValue
}

// getEnvInt 从 env map 中获取整数值
func getEnvInt(env map[string]string, key string, defaultValue int) int {
	val := env[key]
	if val == "" {
		return defaultValue
	}
	if i, err := strconv.Atoi(val); err == nil {
		return i
	}
	logger.Log.Printf("[config] .env 中 %s=%s 解析失败，使用默认值 %d", key, val, defaultValue)
	return defaultValue
}

// loadNotifyFromEnvFile 从 .env 文件加载配置
func loadNotifyFromEnvFile(path string) NotifyConfig {
	env := LoadEnvFile(path)
	cfg := defaultNotifyConfig()
	cfg.Enabled = getEnvBool(env, "XQ_NOTIFY_ENABLED", cfg.Enabled)
	cfg.FeishuAppID = GetEnvStr(env, "XQ_FEISHU_APP_ID", cfg.FeishuAppID)
	cfg.FeishuAppSecret = GetEnvStr(env, "XQ_FEISHU_APP_SECRET", cfg.FeishuAppSecret)
	cfg.FeishuReceiveID = GetEnvStr(env, "XQ_FEISHU_RECEIVE_ID", cfg.FeishuReceiveID)
	cfg.FeishuReceiveType = GetEnvStr(env, "XQ_FEISHU_RECEIVE_TYPE", cfg.FeishuReceiveType)
	cfg.WeightThreshold = getEnvFloat(env, "XQ_WEIGHT_THRESHOLD", cfg.WeightThreshold)
	cfg.IntervalMinutes = getEnvInt(env, "XQ_INTERVAL_MINUTES", cfg.IntervalMinutes)

	// 校验并修正边界值
	if cfg.WeightThreshold < 0 {
		cfg.WeightThreshold = 5
	}
	if cfg.IntervalMinutes < 1 {
		cfg.IntervalMinutes = 30
	}
	if cfg.FeishuReceiveType == "" {
		cfg.FeishuReceiveType = "open_id"
	}

	logger.Log.Printf("[config] 已从 .env 加载配置 path=%s enabled=%v interval=%dm threshold=%.1f%%",
		path, cfg.Enabled, cfg.IntervalMinutes, cfg.WeightThreshold)

	return cfg
}

// configStore 配置存储（只读，来自 .env 文件）
type configStore struct {
	notify NotifyConfig
	path   string // .env 文件路径
}

func (c *configStore) load() {
	c.path = EnvPath()
	c.notify = loadNotifyFromEnvFile(c.path)
}

func (c *configStore) save(notify NotifyConfig) error {
	// .env 文件模式不支持保存配置
	logger.Log.Printf("[config] .env 文件模式不支持保存配置，请直接编辑 %s 文件", c.path)
	// 返回 nil 以避免报错，但配置不会变更
	return nil
}

func (c *configStore) getNotify() NotifyConfig {
	return c.notify
}
