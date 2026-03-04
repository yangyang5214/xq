package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/beer/xq/internal/xueqiu"
)

//go:embed static
var staticFS embed.FS

// Config 服务配置
type Config struct {
	CubesFile   string
	CookiesFile string
	Addr        string
}

// Server HTTP 服务，提供组合列表与组合实际持仓 API
type Server struct {
	cfg Config
}

// New 创建 Server
func New(cfg Config) *Server {
	return &Server{cfg: cfg}
}

// CubeItem 组合列表项
type CubeItem struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// ListCubes 返回组合列表
func (s *Server) ListCubes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	symbols, names, err := xueqiu.LoadCubeSymbolsFromFile(s.cfg.CubesFile)
	if err != nil {
		log.Printf("[server] LoadCubeSymbolsFromFile %s: %v，使用默认组合列表", s.cfg.CubesFile, err)
		symbols = nil
		names = nil
	}
	if len(symbols) == 0 {
		symbols = []string{"ZH3347671"}
		if names == nil {
			names = make(map[string]string)
		}
	}
	items := make([]CubeItem, 0, len(symbols))
	for _, sym := range symbols {
		name := names[sym]
		if name == "" {
			name = sym
		}
		items = append(items, CubeItem{Symbol: sym, Name: name})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"cubes": items})
}

// CubeHoldings 返回指定组合的实际持仓
func (s *Server) CubeHoldings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	symbol := strings.TrimPrefix(r.URL.Path, "/api/cubes/")
	if idx := strings.Index(symbol, "/"); idx >= 0 {
		symbol = symbol[:idx]
	}
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少 cube symbol"})
		return
	}
	snapshot, _, err := xueqiu.FetchCubeViaAPI(symbol, s.cfg.CookiesFile)
	if err != nil {
		log.Printf("[server] FetchCubeViaAPI %s: %v", symbol, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "获取持仓失败: " + err.Error()})
		return
	}
	if snapshot.UpdatedAt == 0 {
		snapshot.UpdatedAt = time.Now().Unix()
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Mux 返回配置好的 http.ServeMux
func (s *Server) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/cubes", s.ListCubes)
	mux.HandleFunc("/api/cubes/", s.CubeHoldings)
	web, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(web)))
	return mux
}

// Run 启动 HTTP 服务
func (s *Server) Run() error {
	log.Printf("server listening on %s", s.cfg.Addr)
	return http.ListenAndServe(s.cfg.Addr, s.Mux())
}
