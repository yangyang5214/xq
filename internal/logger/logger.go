package logger

import (
	"io"
	"log"
	"os"
)

// Log 全局唯一 log 对象，全项目统一使用
var Log = log.New(os.Stderr, "", log.LstdFlags)

// Init 初始化日志输出到文件，同时保留 stderr
// 若 filename 为空则仅使用 stderr
func Init(filename string) {
	if filename == "" {
		return
	}
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		Log.Printf("[logger] 无法打开日志文件 %s: %v", filename, err)
		return
	}
	Log.SetOutput(io.MultiWriter(f, os.Stderr))
}
