package main

import (
	"fmt"

	"github.com/beer/xq/internal/feishu"
	"github.com/beer/xq/internal/logger"
	"github.com/beer/xq/internal/server"
	"github.com/spf13/cobra"
)

var feishuTestCmd = &cobra.Command{
	Use:   "feishu-test [message]",
	Short: "测试飞书消息发送",
	Long:  `向配置的飞书接收者发送测试消息。消息内容可通过参数指定，默认为并发送一条测试消息。`,
	Example: `  xq feishu-test
  xq feishu-test "这是一条测试消息"
  xq feishu-test --message "自定义消息"`,
	RunE: runFeishuTest,
}

var (
	feishuTestMsg string
)

func init() {
	feishuTestCmd.Flags().StringVarP(&feishuTestMsg, "message", "m", "",
		"要发送的消息内容。如果不指定，将使用默认测试消息")
	rootCmd.AddCommand(feishuTestCmd)
}

func runFeishuTest(cmd *cobra.Command, args []string) error {
	// 确定消息内容
	message := feishuTestMsg
	if message == "" {
		if len(args) > 0 {
			message = args[0]
		} else {
			message = "这是一条来自 xq 工具的测试消息，飞书通知配置正常！"
		}
	}

	// 从 .env 加载配置
	envPath := server.EnvPath()
	env := server.LoadEnvFile(envPath)

	// 读取配置
	cfg := &feishu.Config{
		AppID:       server.GetEnvStr(env, "XQ_FEISHU_APP_ID", ""),
		AppSecret:   server.GetEnvStr(env, "XQ_FEISHU_APP_SECRET", ""),
		ReceiveID:   server.GetEnvStr(env, "XQ_FEISHU_RECEIVE_ID", ""),
		ReceiveType: server.GetEnvStr(env, "XQ_FEISHU_RECEIVE_TYPE", "open_id"),
	}

	// 验证配置
	if cfg.AppID == "" {
		return fmt.Errorf("XQ_FEISHU_APP_ID 未配置，请检查 %s 文件", envPath)
	}
	if cfg.AppSecret == "" {
		return fmt.Errorf("XQ_FEISHU_APP_SECRET 未配置，请检查 %s 文件", envPath)
	}
	if cfg.ReceiveID == "" {
		return fmt.Errorf("XQ_FEISHU_RECEIVE_ID 未配置，请检查 %s 文件", envPath)
	}

	logger.Log.Printf("[feishu-test] 加载配置 from=%s receive_type=%s", envPath, cfg.ReceiveType)

	// 发送消息
	logger.Log.Printf("[feishu-test] 正在发送消息...")
	if err := cfg.Send(message); err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}

	logger.Log.Printf("[feishu-test] 消息发送成功")
	return nil
}
