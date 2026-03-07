package logger

import (
	"log"
	"os"
)

// Log 全局唯一 log 对象，全项目统一使用
var Log = log.New(os.Stderr, "", log.LstdFlags)
