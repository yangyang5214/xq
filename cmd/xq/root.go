package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "xq",
	Short: "雪球组合持仓分布对比与提醒",
	Long:  `从 cookies.txt 与组合列表拉取雪球组合当前持仓分布，与上次快照对比；比例变化超过阈值时发邮件提醒。支持 server（HTTP 服务）与 notify（拉取并邮件提醒）子命令。`,
}
