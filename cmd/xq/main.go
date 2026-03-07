package main

import (
	"github.com/beer/xq/internal/logger"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Log.Fatal(err)
	}
}
