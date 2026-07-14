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
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  StrategyParams `json:"parameters"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// StrategyParams holds the configurable parameters for a strategy.
type StrategyParams struct {
	RSIPeriod    int     `json:"rsi_period"`
	RSIThreshold float64 `json:"rsi_threshold"` // oversold threshold (e.g., 30)
	SMAPeriod    int     `json:"sma_period"`
	SMATrigger   float64 `json:"sma_trigger"`  // deviation from SMA to trigger
	MaxPosition  float64 `json:"max_position"` // max amount per trade
	StopLoss     float64 `json:"stop_loss"`    // stop-loss percentage
}

// Trade represents a single executed or pending trade.
type Trade struct {
	ID         string         `json:"id"`
	StrategyID string         `json:"strategy_id"`
	Symbol     string         `json:"symbol"`
	Currency   string         `json:"currency"`
	Direction  TradeDirection `json:"direction"`
	Quantity   float64        `json:"quantity"`
	Price      float64        `json:"price"`
	ExecutedAt time.Time      `json:"executed_at"`
	Status     TradeStatus    `json:"status"`
	PnL        float64        `json:"pnl"`
	Notes      string         `json:"notes,omitempty"`
}

// BacktestResult holds the outcome of a strategy backtest.
type BacktestResult struct {
	Strategy       Strategy  `json:"strategy"`
	From           time.Time `json:"from"`
	To             time.Time `json:"to"`
	TotalTrades    int       `json:"total_trades"`
	WinningTrades  int       `json:"winning_trades"`
	LosingTrades   int       `json:"losing_trades"`
	WinRate        float64   `json:"win_rate"`
	TotalReturn    float64   `json:"total_return"`
	TotalReturnPct float64   `json:"total_return_pct"`
	MaxDrawdown    float64   `json:"max_drawdown"`
	SharpeRatio    float64   `json:"sharpe_ratio"`
	Trades         []Trade   `json:"trades"`
}
