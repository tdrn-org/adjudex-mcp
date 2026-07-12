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

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

func addTradeTools(server *mcp.Server, runtime Runtime) {
	addTradeRecordTool(server, runtime)
	addTradeGetTool(server, runtime)
	addTradeListTool(server, runtime)
}

func addTradeRecordTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "trade_record",
		Description: "Records a new trade (buy or sell). The created trade details are returned.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"strategy_id":{"type":"string","description":"ID of the strategy that triggered this trade."},"symbol":{"type":"string","description":"The ticker symbol."},"direction":{"type":"string","description":"Trade direction: buy or sell."},"quantity":{"type":"number","description":"Number of shares."},"price":{"type":"number","description":"Execution price per share."},"executed_at":{"type":"string","description":"Execution timestamp (RFC3339 format)."},"status":{"type":"string","description":"Trade status: pending, executed, cancelled. Defaults to executed."},"pnl":{"type":"number","description":"Profit/loss for sell trades. Defaults to 0."},"notes":{"type":"string","description":"Optional notes."}},"required":["strategy_id","symbol","direction","quantity","price","executed_at"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			StrategyID string  `json:"strategy_id"`
			Symbol     string  `json:"symbol"`
			Direction  string  `json:"direction"`
			Quantity   float64 `json:"quantity"`
			Price      float64 `json:"price"`
			ExecutedAt string  `json:"executed_at"`
			Status     string  `json:"status"`
			PnL        float64 `json:"pnl"`
			Notes      string  `json:"notes"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		executedAt, err := time.Parse(time.RFC3339, args.ExecutedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing executed_at: %w", err)
		}

		direction := domain.TradeDirection(args.Direction)
		switch direction {
		case domain.TradeBuy, domain.TradeSell:
		default:
			return nil, fmt.Errorf("unknown direction %q (valid: buy, sell)", args.Direction)
		}

		status := domain.TradeStatus(args.Status)
		if status == "" {
			status = domain.TradeExecuted
		}

		t := &domain.Trade{
			StrategyID: args.StrategyID,
			Symbol:     args.Symbol,
			Direction:  direction,
			Quantity:   args.Quantity,
			Price:      args.Price,
			ExecutedAt: executedAt,
			Status:     status,
			PnL:        args.PnL,
			Notes:      args.Notes,
		}
		if err := runtime.DataStore().RecordTrade(ctx, t); err != nil {
			return nil, fmt.Errorf("recording trade: %w", err)
		}
		return newToolResult(t)
	})
}

func addTradeGetTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "trade_get",
		Description: "Gets the full trade details for the given ID.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"ID of the trade to return."}},"required":["id"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		t, err := runtime.DataStore().GetTrade(ctx, args.ID)
		if err != nil {
			return nil, fmt.Errorf("getting trade %q: %w", args.ID, err)
		}
		return newToolResult(t)
	})
}

func addTradeListTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "trade_list",
		Description: "Lists trades. Optionally filter by symbol or strategy ID.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"symbol":{"type":"string","description":"Optional symbol filter."},"strategy_id":{"type":"string","description":"Optional strategy ID filter."}}}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Symbol     string `json:"symbol"`
			StrategyID string `json:"strategy_id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		if args.StrategyID != "" {
			trades, err := runtime.DataStore().ListTradesByStrategy(ctx, args.StrategyID)
			if err != nil {
				return nil, fmt.Errorf("listing trades by strategy: %w", err)
			}
			return newToolResult(trades)
		}

		trades, err := runtime.DataStore().ListTrades(ctx, args.Symbol)
		if err != nil {
			return nil, fmt.Errorf("listing trades: %w", err)
		}
		return newToolResult(trades)
	})
}
