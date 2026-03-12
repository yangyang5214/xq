package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/beer/xq/internal/logger"
)

// tokenResponse 获取 tenant_access_token 的响应
type tokenResponse struct {
	Code int `json:"code"`
	Msg  string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
	Expire int `json:"expire"`
}

// sendMessageResponse 发送消息的响应
type sendMessageResponse struct {
	Code int `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		MsgID string `json:"msg_id"`
	} `json:"data"`
}

// wsClient WebSocket 长连接客户端
type wsClient struct {
	appID     string
	appSecret string
	token     string
	tokenExp  time.Time
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// newWSClient 创建 WebSocket 客户端
func newWSClient(appID, appSecret string) *wsClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &wsClient{
		appID:     appID,
		appSecret: appSecret,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// getTenantAccessToken 获取 tenant_access_token（带缓存）
func (w *wsClient) getTenantAccessToken() (string, error) {
	w.mu.RLock()
	if w.token != "" && time.Now().Before(w.tokenExp) {
		token := w.token
		w.mu.RUnlock()
		return token, nil
	}
	w.mu.RUnlock()

	// 获取新 token
	w.mu.Lock()
	defer w.mu.Unlock()

	// 双重检查
	if w.token != "" && time.Now().Before(w.tokenExp) {
		return w.token, nil
	}

	token, exp, err := w.fetchToken()
	if err != nil {
		return "", err
	}

	w.token = token
	w.tokenExp = time.Now().Add(time.Duration(exp-300) * time.Second) // 提前5分钟过期
	return token, nil
}

// fetchToken 从飞书 API 获取 token
func (w *wsClient) fetchToken() (string, int, error) {
	if w.appID == "" || w.appSecret == "" {
		return "", 0, fmt.Errorf("飞书 App ID 或 App Secret 未配置")
	}

	apiURL := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"

	body := map[string]string{
		"app_id":     w.appID,
		"app_secret": w.appSecret,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", 0, fmt.Errorf("marshal token request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(apiURL, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return "", 0, fmt.Errorf("post token request: %w", err)
	}
	defer resp.Body.Close()

	var result tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, fmt.Errorf("decode token response: %w", err)
	}

	if result.Code != 0 {
		return "", 0, fmt.Errorf("获取 token 失败: %s", result.Msg)
	}

	return result.TenantAccessToken, result.Expire, nil
}

// SendReply 给指定消息发送回复
func (w *wsClient) SendReply(chatID, messageID, text string) error {
	token, err := w.getTenantAccessToken()
	if err != nil {
		return fmt.Errorf("获取 access token: %w", err)
	}

	content := map[string]interface{}{
		"text": text,
	}
	msgBody := map[string]interface{}{
		"msg_type": "text",
		"content":  content,
	}

	jsonBody, err := json.Marshal(msgBody)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	apiURL := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s/reply", messageID)

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	var result sendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Log.Printf("[feishu] 解析响应失败: %v", err)
		return nil
	}

	if result.Code != 0 {
		return fmt.Errorf("飞书 API 错误: %s", result.Msg)
	}

	logger.Log.Printf("[feishu] 回复已发送，msg_id=%s", result.Data.MsgID)
	return nil
}

// Send 发送飞书消息（用于持仓变化提醒，需要预先设置的接收者）
// 注意：长连接模式下，建议通过事件响应回复，而不是主动发送
func (w *wsClient) Send(receiveID, receiveType, text string) error {
	if receiveID == "" {
		return fmt.Errorf("接收者 ID 未配置")
	}
	if receiveType == "" {
		receiveType = "open_id"
	}

	token, err := w.getTenantAccessToken()
	if err != nil {
		return fmt.Errorf("获取 access token: %w", err)
	}

	contentStr, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("marshal content: %w", err)
	}
	msgBody := map[string]interface{}{
		"receive_id":     receiveID,
		"receive_id_type": receiveType,
		"msg_type":       "text",
		"content":       string(contentStr),
	}

	jsonBody, err := json.Marshal(msgBody)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	logger.Log.Printf("[feishu] 发送消息: %s", text)

	apiURL := "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=" + receiveType

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	var result sendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Log.Printf("[feishu] 解析响应失败: %v", err)
		return nil
	}

	if result.Code != 0 {
		return fmt.Errorf("飞书 API 错误: %s", result.Msg)
	}

	logger.Log.Printf("[feishu] 消息已发送，msg_id=%s", result.Data.MsgID)
	return nil
}

// Config 飞书通知配置
type Config struct {
	AppID       string `json:"app_id"`
	AppSecret   string `json:"app_secret"`
	// 长连接模式下仍可配置主动推送的目标（可选）
	ReceiveID   string `json:"receive_id,omitempty"`
	ReceiveType string `json:"receive_type,omitempty"` // open_id, user_id, union_id, chat_id
}

var (
	defaultClient *wsClient
	clientOnce    sync.Once
)

// getClient 获取默认客户端
func (c *Config) getClient() *wsClient {
	clientOnce.Do(func() {
		defaultClient = newWSClient(c.AppID, c.AppSecret)
	})
	return defaultClient
}

// Send 发送消息（主动发送）
func (c *Config) Send(text string) error {
	cli := c.getClient()
	if c.ReceiveID == "" {
		// 长连接模式：没有预设接收者时记录警告
		logger.Log.Printf("[feishu] 警告: 长连接模式下未配置 receive_id，无法主动发送消息")
		return fmt.Errorf("长连接模式下需要配置 receive_id 才能主动发送消息")
	}
	return cli.Send(c.ReceiveID, c.ReceiveType, text)
}

// SendText 发送纯文本消息
func (c *Config) SendText(subject, content string) error {
	text := fmt.Sprintf("%s\n\n%s", subject, content)
	return c.Send(text)
}

// SendReply 回复消息（由事件触发）
func (c *Config) SendReply(chatID, messageID, text string) error {
	cli := c.getClient()
	return cli.SendReply(chatID, messageID, text)
}

// GetToken 获取当前 token（供测试使用）
func (c *Config) GetToken() (string, error) {
	cli := c.getClient()
	return cli.getTenantAccessToken()
}
