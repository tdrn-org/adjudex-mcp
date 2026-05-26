/*
 * Copyright 2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package domain

import "time"

// TradeDirection indicates buy or sell.
type TradeDirection string

const (
	TradeBuy  TradeDirection = "buy"
	TradeSell TradeDirection = "sell"
)

// TradeStatus tracks the lifecycle of a trade.
type TradeStatus string

const (
	TradePending   TradeStatus = "pending"
	TradeExecuted  TradeStatus = "executed"
	TradeCancelled TradeStatus = "cancelled"
)

// Strategy represents a trading strategy definition (e.g., mean-reversion).
type Strategy struct {
	ID          string
	Name        string
	Description string
	Parameters  StrategyParams
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// StrategyParams holds the configurable parameters for a strategy.
type StrategyParams struct {
	RSIPeriod    int     `json:"rsi_period"`
	RSIThreshold float64 `json:"rsi_threshold"` // oversold threshold (e.g., 30)
	SMAPeriod    int     `json:"sma_period"`
	SMATrigger   float64 `json:"sma_trigger"` // deviation from SMA to trigger
	MaxPosition  float64 `json:"max_position"` // max amount per trade
	StopLoss     float64 `json:"stop_loss"`    // stop-loss percentage
}

// Trade represents a single executed or pending trade.
type Trade struct {
	ID         string
	StrategyID string
	Symbol     string
	Direction  TradeDirection
	Quantity   float64
	Price      float64
	ExecutedAt time.Time
	Status     TradeStatus
	PnL        float64 // filled after sell (exit) trade
	Notes      string
}

// BacktestResult holds the outcome of a strategy backtest.
type BacktestResult struct {
	Strategy     Strategy
	Symbol       string
	From         time.Time
	To           time.Time
	TotalTrades  int
	WinningTrades int
	LosingTrades  int
	WinRate      float64 // percentage
	TotalReturn  float64 // absolute
	TotalReturnPct float64 // percentage
	MaxDrawdown  float64 // maximum peak-to-trough decline
	SharpeRatio  float64
	Trades       []Trade
}
