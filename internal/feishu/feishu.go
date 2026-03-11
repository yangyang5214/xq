package feishu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/beer/xq/internal/logger"
)

// Config 飞书通知配置
type Config struct {
	WebhookURL string `json:"webhook_url"` // 飞书群机器人的 webhook URL
}

// Message 飞书消息
type Message struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

// Send 发送飞书消息
func (c *Config) Send(text string) error {
	if c.WebhookURL == "" {
		return fmt.Errorf("飞书 webhook URL 未配置")
	}

	msg := Message{MsgType: "text"}
	msg.Content.Text = text

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	logger.Log.Printf("[feishu] 发送消息: %s", text)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(c.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("post webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("飞书 API 返回 status=%d", resp.StatusCode)
	}

	var result struct {
		StatusCode int    `json:"StatusCode"`
		StatusMsg  string `json:"StatusMsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Log.Printf("[feishu] 解析响应失败: %v", err)
		return nil // 仍然视为成功，因为已发送
	}

	if result.StatusCode != 0 && result.StatusMsg != "success" {
		return fmt.Errorf("飞书 API 错误: %s", result.StatusMsg)
	}

	logger.Log.Printf("[feishu] 消息已发送")
	return nil
}

// SendText 发送纯文本消息
func (c *Config) SendText(subject, content string) error {
	text := fmt.Sprintf("%s\n\n%s", subject, content)
	return c.Send(text)
}
