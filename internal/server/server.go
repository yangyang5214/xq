package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/beer/xq/internal/logger"
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
	cfg         Config
	configStore *configStore
}

// New 创建 Server
func New(cfg Config) *Server {
	cs := &configStore{}
	cs.load() // 从 $HOME/.xq_config.json 加载
	return &Server{cfg: cfg, configStore: cs}
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
		logger.Log.Printf("[server] LoadCubeSymbolsFromFile %s: %v，使用默认组合列表", s.cfg.CubesFile, err)
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
		logger.Log.Printf("[server] FetchCubeViaAPI %s: %v", symbol, err)
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

// ConfigGet 返回当前 notify 配置
func (s *Server) ConfigGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"notify": s.configStore.getNotify(),
	})
}

// ConfigPut 保存 notify 配置
func (s *Server) ConfigPut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Notify *NotifyConfig `json:"notify"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Notify == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的请求体，需包含 notify 对象"})
		return
	}
	nc := *body.Notify
	if nc.WeightThreshold < 0 {
		nc.WeightThreshold = 5
	}
	if nc.IntervalMinutes < 1 {
		nc.IntervalMinutes = 30
	}
	if err := s.configStore.save(nc); err != nil {
		logger.Log.Printf("[server] 保存配置失败 err=%v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存配置失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"notify": nc})
}

// NotifyRun 手动触发一次提醒
func (s *Server) NotifyRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	go s.runNotify()
	writeJSON(w, http.StatusOK, map[string]string{"message": "已触发提醒"})
}

// Mux 返回配置好的 http.ServeMux
func (s *Server) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/cubes", s.ListCubes)
	mux.HandleFunc("/api/cubes/", s.CubeHoldings)
	mux.HandleFunc("/api/config", s.configHandler)
	mux.HandleFunc("/api/notify/run", s.NotifyRun)
	web, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(web)))
	return mux
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.ConfigGet(w, r)
	case http.MethodPut, http.MethodPost:
		s.ConfigPut(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// Run 启动 HTTP 服务
func (s *Server) Run() error {
	logger.Log.Printf("server listening on %s", s.cfg.Addr)
	s.startNotifyLoop()
	s.startDailySummaryLoop()
	return http.ListenAndServe(s.cfg.Addr, s.Mux())
}
