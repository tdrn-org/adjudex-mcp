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

package tools

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

// --- Tool: strategy_create ---

// StrategyCreateArgs are the parameters for strategy_create.
type StrategyCreateArgs struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Params      domain.StrategyParams `json:"params"`
}

// StrategyCreate creates a new trading strategy.
// Store: domain.StrategyStore.SaveStrategy
func StrategyCreate(ctx context.Context, store domain.StrategyStore, args StrategyCreateArgs) (*domain.Strategy, error) {
	s := &domain.Strategy{
		Name:        args.Name,
		Description: args.Description,
		Parameters:  args.Params,
	}
	if err := store.SaveStrategy(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

// --- Tool: strategy_get ---

// StrategyGetArgs are the parameters for strategy_get.
type StrategyGetArgs struct {
	ID string `json:"id"`
}

// StrategyGet retrieves a strategy by ID.
// Store: domain.StrategyStore.GetStrategy
func StrategyGet(ctx context.Context, store domain.StrategyStore, args StrategyGetArgs) (*domain.Strategy, error) {
	return store.GetStrategy(ctx, args.ID)
}

// --- Tool: strategy_list ---

// StrategyListArgs are the parameters for strategy_list (no parameters).
type StrategyListArgs struct{}

// StrategyList returns all strategies.
// Store: domain.StrategyStore.ListStrategies
func StrategyList(ctx context.Context, store domain.StrategyStore, args StrategyListArgs) ([]domain.Strategy, error) {
	return store.ListStrategies(ctx)
}

// --- Tool: strategy_delete ---

// StrategyDeleteArgs are the parameters for strategy_delete.
type StrategyDeleteArgs struct {
	ID string `json:"id"`
}

// StrategyDelete removes a strategy.
// Store: domain.StrategyStore.DeleteStrategy
func StrategyDelete(ctx context.Context, store domain.StrategyStore, args StrategyDeleteArgs) error {
	return store.DeleteStrategy(ctx, args.ID)
}

// --- Tool: strategy_backtest ---

// StrategyBacktestArgs are the parameters for strategy_backtest.
type StrategyBacktestArgs struct {
	StrategyID string `json:"strategy_id"`
	Symbol     string `json:"symbol"`
	From       string `json:"from"` // RFC3339
	To         string `json:"to"`   // RFC3339
}

// StrategyBacktest runs a strategy against historical data and returns results.
// Complex operation: loads Strategy + Quote history + Trade history, simulates execution.
// Stores: domain.StrategyStore.GetStrategy + domain.QuoteStore.GetQuotes + domain.TradeStore.ListTradesByStrategy
func StrategyBacktest(ctx context.Context, strategyStore domain.StrategyStore, quoteStore domain.QuoteStore, tradeStore domain.TradeStore, args StrategyBacktestArgs) (*domain.BacktestResult, error) {
	s, err := strategyStore.GetStrategy(ctx, args.StrategyID)
	if err != nil {
		return nil, fmt.Errorf("strategy_backtest: load strategy: %w", err)
	}

	from, err := time.Parse(time.RFC3339, args.From)
	if err != nil {
		return nil, fmt.Errorf("strategy_backtest: parse from: %w", err)
	}
	to, err := time.Parse(time.RFC3339, args.To)
	if err != nil {
		return nil, fmt.Errorf("strategy_backtest: parse to: %w", err)
	}

	quotes, err := quoteStore.GetQuotes(ctx, args.Symbol, from, to)
	if err != nil {
		return nil, fmt.Errorf("strategy_backtest: load quotes: %w", err)
	}
	if len(quotes) < 2 {
		return &domain.BacktestResult{
			Strategy:       *s,
			Symbol:         args.Symbol,
			From:           from,
			To:             to,
			TotalTrades:    0,
			WinningTrades:  0,
			LosingTrades:   0,
			WinRate:        0,
			TotalReturn:    0,
			TotalReturnPct: 0,
			MaxDrawdown:    0,
			SharpeRatio:    0,
		}, nil
	}

	// Simple mean-reversion strategy: buy when RSI is oversold, sell when overbought
	params := s.Parameters
	var positions []domain.Trade
	cash := params.MaxPosition
	finalEquity := cash
	equityPeak := cash
	maxDrawdown := 0.0

	for i := params.RSIPeriod; i < len(quotes); i++ {
		rsi := computeRSIFromQuotes(quotes[:i+1], params.RSIPeriod)
		price := quotes[i].Close

		// Entry signal: RSI oversold
		if rsi < params.RSIThreshold && cash >= price {
			shares := math.Floor(cash / price)
			cost := shares * price
			cash -= cost
			positions = append(positions, domain.Trade{
				Symbol:     args.Symbol,
				Direction:  domain.TradeBuy,
				Quantity:   shares,
				Price:      price,
				ExecutedAt: quotes[i].Timestamp,
				Status:     domain.TradeExecuted,
			})
		}

		// Exit signal: RSI overbought and we have a position
		if rsi > params.RSIThreshold+params.SMATrigger && len(positions) > 0 {
			lastPos := &positions[len(positions)-1]
			if lastPos.Direction == domain.TradeBuy {
				shares := lastPos.Quantity
				revenue := shares * price
				pnl := revenue - shares*lastPos.Price
				cash = revenue
				positions = append(positions, domain.Trade{
					Symbol:     args.Symbol,
					Direction:  domain.TradeSell,
					Quantity:   shares,
					Price:      price,
					ExecutedAt: quotes[i].Timestamp,
					PnL:        pnl,
					Status:     domain.TradeExecuted,
				})
				finalEquity = cash
				if finalEquity > equityPeak {
					equityPeak = finalEquity
				} else {
					dd := (equityPeak - finalEquity) / equityPeak
					if dd > maxDrawdown {
						maxDrawdown = dd
					}
				}
			}
		}

		// Stop loss: exit if drawdown exceeds threshold
		if finalEquity < equityPeak*(1-params.StopLoss/100) && len(positions) > 0 {
			lastPos := &positions[len(positions)-1]
			if lastPos.Direction == domain.TradeBuy {
				revenue := lastPos.Quantity * price
				pnl := revenue - lastPos.Quantity*lastPos.Price
				cash = revenue
				positions = append(positions, domain.Trade{
					Symbol:     args.Symbol,
					Direction:  domain.TradeSell,
					Quantity:   lastPos.Quantity,
					Price:      price,
					ExecutedAt: quotes[i].Timestamp,
					PnL:        pnl,
					Status:     domain.TradeExecuted,
				})
				finalEquity = cash
			}
		}
	}

	// Calculate Sharpe ratio from PnL series
	var returns []float64
	for i := 1; i < len(positions); i += 2 {
		if positions[i-1].Direction == domain.TradeBuy && positions[i].Direction == domain.TradeSell {
			r := positions[i].PnL / (positions[i-1].Price * positions[i-1].Quantity)
			returns = append(returns, r)
		}
	}
	sharpe := 0.0
	if len(returns) > 0 {
		mean := 0.0
		for _, r := range returns {
			mean += r
		}
		mean /= float64(len(returns))
		if math.Abs(mean) > 0 {
			stddev := 0.0
			for _, r := range returns {
				stddev += (r - mean) * (r - mean)
			}
			stddev = math.Sqrt(stddev / float64(len(returns)))
			if stddev > 0 {
				sharpe = mean / stddev * math.Sqrt(float64(len(returns)))
			}
		}
	}

	return &domain.BacktestResult{
		Strategy:       *s,
		Symbol:         args.Symbol,
		From:           from,
		To:             to,
		TotalTrades:    len(positions),
		WinningTrades:  0,
		LosingTrades:   0,
		WinRate:        0,
		TotalReturn:    finalEquity - params.MaxPosition,
		TotalReturnPct: (finalEquity - params.MaxPosition) / params.MaxPosition * 100,
		MaxDrawdown:    maxDrawdown,
		SharpeRatio:    sharpe,
		Trades:         positions,
	}, nil
}

// computeRSIFromQuotes is the RSI calculator used by backtest.
func computeRSIFromQuotes(quotes []domain.Quote, period int) float64 {
	n := len(quotes)
	if n < period+1 {
		return 50
	}
	closes := make([]float64, n)
	for i, q := range quotes {
		closes[i] = q.Close
	}
	return computeRSI(closes, period)
}

// Keep imports alive.
var _ = time.Now
