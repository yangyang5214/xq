package main

import (
	"github.com/beer/xq/internal/logger"
)

func main() {
	logger.Init("runtime.log")
	if err := rootCmd.Execute(); err != nil {
		logger.Log.Fatal(err)
	}
}
