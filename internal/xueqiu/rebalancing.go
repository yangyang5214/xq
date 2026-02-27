package xueqiu

import (
	"encoding/json"
	"fmt"
)

// HistoryResp 调仓历史接口返回（与雪球实际返回字段对齐）
type HistoryResp struct {
	Data struct {
		List       []RebalanceItem `json:"list"`
		Count      int64           `json:"count"`
		TotalCount int64           `json:"totalCount"`
		Page       int64           `json:"page"`
		MaxPage    int64           `json:"maxPage"`
	} `json:"data"`
}

// 兼容无 data 包裹的返回
type HistoryRespFlat struct {
	List       []RebalanceItem `json:"list"`
	Count      int64           `json:"count"`
	TotalCount int64           `json:"totalCount"`
	Page       int64           `json:"page"`
	MaxPage    int64           `json:"maxPage"`
}

// RebalanceItem 单条调仓记录
type RebalanceItem struct {
	ID                   int64            `json:"id"`
	Status               string           `json:"status"`
	Category             string           `json:"category"`
	CreatedAt            int64            `json:"created_at"`
	UpdatedAt            int64            `json:"updated_at"`
	PrevWeight           float64          `json:"prev_weight"`
	CashValue            float64          `json:"cash_value"`
	Cash                 float64          `json:"cash"`
	ErrorMsg             string           `json:"error_message"`
	Comment              string           `json:"comment"`
	RebalancingHistories []RebalanceStock `json:"rebalancing_histories"`
}

type RebalanceStock struct {
	Id                 int     `json:"id"`
	RebalancingId      int     `json:"rebalancing_id"`
	StockId            int     `json:"stock_id"`
	StockName          string  `json:"stock_name"`
	StockSymbol        string  `json:"stock_symbol"`
	Volume             float64 `json:"volume"`
	Price              float64 `json:"price"`
	NetValue           float64 `json:"net_value"`
	Weight             float64 `json:"weight"`
	TargetWeight       float64 `json:"target_weight"`
	PrevWeight         float64 `json:"prev_weight"`
	PrevTargetWeight   float64 `json:"prev_target_weight"`   //调仓后
	PrevWeightAdjusted float64 `json:"prev_weight_adjusted"` //调仓前
	PrevVolume         float64 `json:"prev_volume"`
	PrevPrice          float64 `json:"prev_price"`
	PrevNetValue       float64 `json:"prev_net_value"`
	Proactive          bool    `json:"proactive"`
	CreatedAt          int64   `json:"created_at"`
	UpdatedAt          int64   `json:"updated_at"`
	TargetVolume       float64 `json:"target_volume"`
	PrevTargetVolume   float64 `json:"prev_target_volume"`
}

// OriginResp 单次调仓详情 show_origin 返回
type OriginResp struct {
	Data struct {
		ID                   int64            `json:"id"`
		Status               string           `json:"status"`
		Category             string           `json:"category"`
		CreatedAt            int64            `json:"created_at"`
		UpdatedAt            int64            `json:"updated_at"`
		CashValue            float64          `json:"cash_value"`
		Comment              string           `json:"comment"`
		RebalancingHistories []RebalanceStock `json:"rebalancing_histories"`
	} `json:"data"`
}

// ParseHistory 解析调仓历史 JSON，兼容有无 data 包裹
func ParseHistory(raw []byte) (*HistoryRespFlat, error) {
	var withData HistoryResp
	if err := json.Unmarshal(raw, &withData); err == nil && len(withData.Data.List) > 0 {
		return &HistoryRespFlat{
			List:       withData.Data.List,
			Count:      withData.Data.Count,
			TotalCount: withData.Data.TotalCount,
			Page:       withData.Data.Page,
			MaxPage:    withData.Data.MaxPage,
		}, nil
	}
	var flat HistoryRespFlat
	if err := json.Unmarshal(raw, &flat); err != nil {
		return nil, fmt.Errorf("parse history: %w", err)
	}
	return &flat, nil
}

// ParseOrigin 解析单次调仓详情
func ParseOrigin(raw []byte) (*OriginResp, error) {
	var out OriginResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse show_origin: %w", err)
	}
	return &out, nil
}
