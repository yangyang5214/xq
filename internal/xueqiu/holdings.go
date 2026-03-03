package xueqiu

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Holding 单只标的的持仓权重
type Holding struct {
	Symbol string  `json:"symbol"`
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

// HoldingsSnapshot 组合持仓分布快照（用于与上次对比）
type HoldingsSnapshot struct {
	CubeSymbol string    `json:"cube_symbol"`
	UpdatedAt  int64     `json:"updated_at"`
	Holdings   []Holding `json:"holdings"`
}

// cubeTreeNode 与 SNB.cubeTreeData 单节点对齐（雪球可能用 symbol/name 或 stock_symbol/stock_name）
type cubeTreeNode struct {
	Name        string         `json:"name"`
	Symbol      string         `json:"symbol"`
	StockName   string         `json:"stock_name"`
	StockSymbol string         `json:"stock_symbol"`
	Weight      float64        `json:"weight"`
	Children    []cubeTreeNode `json:"children"`
}

func (n *cubeTreeNode) symbol() string {
	if n.StockSymbol != "" {
		return n.StockSymbol
	}
	return n.Symbol
}

func (n *cubeTreeNode) name() string {
	if n.StockName != "" {
		return n.StockName
	}
	return n.Name
}

func (t cubeTreeNodes) flattenInto(out *[]Holding) {
	for i := range t {
		t[i].flattenInto(out)
	}
}

type cubeTreeNodes []cubeTreeNode

func (n *cubeTreeNode) flattenInto(out *[]Holding) {
	sym := n.symbol()
	if sym != "" && (n.Weight > 0 || n.name() != "") {
		*out = append(*out, Holding{Symbol: sym, Name: n.name(), Weight: n.Weight})
	}
	for i := range n.Children {
		n.Children[i].flattenInto(out)
	}
}

// current.json 响应（与 easytrader 使用的 portfolio_url_new 一致）
type currentRebalancingResp struct {
	LastRB *struct {
		Cash     float64              `json:"cash"`
		Holdings []currentAPIHolding  `json:"holdings"`
	} `json:"last_rb"`
}

type currentAPIHolding struct {
	StockSymbol string  `json:"stock_symbol"`
	StockName   string  `json:"stock_name"`
	Weight      float64 `json:"weight"`
}

// ParseCurrentRebalancing 从 /cubes/rebalancing/current.json 响应解析持仓（easytrader 核心逻辑）
func ParseCurrentRebalancing(raw []byte, cubeSymbol string) (*HoldingsSnapshot, error) {
	var resp currentRebalancingResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parse current.json: %w", err)
	}
	if resp.LastRB == nil || len(resp.LastRB.Holdings) == 0 {
		return nil, fmt.Errorf("current.json 无 last_rb 或 holdings 为空")
	}
	s := &HoldingsSnapshot{CubeSymbol: cubeSymbol, Holdings: make([]Holding, 0, len(resp.LastRB.Holdings))}
	for _, h := range resp.LastRB.Holdings {
		if h.StockSymbol == "" {
			continue
		}
		s.Holdings = append(s.Holdings, Holding{Symbol: h.StockSymbol, Name: h.StockName, Weight: h.Weight})
	}
	if len(s.Holdings) == 0 {
		return nil, fmt.Errorf("current.json holdings 解析后为空")
	}
	return s, nil
}

// LoadSnapshot 从文件加载上次持仓快照
func LoadSnapshot(path string) (*HoldingsSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var s HoldingsSnapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// SaveSnapshot 保存持仓快照到文件
func SaveSnapshot(path string, s *HoldingsSnapshot) error {
	if s == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// SnapshotPath 返回某组合的快照文件路径
func SnapshotPath(dir, cubeSymbol string) string {
	return filepath.Join(dir, cubeSymbol+".json")
}
