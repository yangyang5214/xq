package main

import (
	"os"
	"path/filepath"

	"github.com/beer/xq/internal/server"
	"github.com/spf13/cobra"
)

var (
	serverCubesFile   string
	serverCookiesFile string
	serverAddr        string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动 HTTP 服务，提供组合列表与持仓 API",
	Long:  `启动 HTTP 服务，对外提供 /api/cubes 组合列表与 /api/cubes/{symbol} 持仓接口，静态资源服务于根路径。`,
	RunE:  runServer,
}

func init() {
	serverCmd.Flags().StringVarP(&serverCubesFile, "cubes-file", "f", "cubes.txt", "组合列表文件路径，每行一个 symbol，支持 # 注释")
	serverCmd.Flags().StringVarP(&serverCookiesFile, "cookies-file", "", "cookies.txt", "Get cookies.txt LOCALLY 导出的 cookies.txt 路径")
	serverCmd.Flags().StringVarP(&serverAddr, "addr", "a", ":8080", "监听地址")
	rootCmd.AddCommand(serverCmd)
}

func runServer(cmd *cobra.Command, args []string) error {
	cubesFile := serverCubesFile
	cookiesFile := serverCookiesFile
	if wd, _ := os.Getwd(); cookiesFile != "" && !filepath.IsAbs(cookiesFile) {
		cookiesFile = filepath.Join(wd, cookiesFile)
	}
	if wd, _ := os.Getwd(); cubesFile != "" && !filepath.IsAbs(cubesFile) {
		cubesFile = filepath.Join(wd, cubesFile)
	}
	srv := server.New(server.Config{
		CubesFile:   cubesFile,
		CookiesFile: cookiesFile,
		Addr:        serverAddr,
	})
	return srv.Run()
}
