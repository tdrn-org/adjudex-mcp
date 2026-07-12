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

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

func addStrategyTools(server *mcp.Server, runtime Runtime) {
	addStrategySaveTool(server, runtime)
	addStrategyGetTool(server, runtime)
	addStrategyListTool(server, runtime)
	addStrategyDeleteTool(server, runtime)
}

func addStrategySaveTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "strategy_save",
		Description: "Creates or updates a trading strategy. The saved strategy details are returned.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"The name of the strategy."},"description":{"type":"string","description":"An optional description."},"rsi_period":{"type":"integer","description":"RSI lookback period (e.g. 14)."},"rsi_threshold":{"type":"number","description":"RSI oversold threshold (e.g. 30)."},"sma_period":{"type":"integer","description":"SMA lookback period (e.g. 20)."},"sma_trigger":{"type":"number","description":"Deviation from SMA to trigger (e.g. 5.0 for 5%)."},"max_position":{"type":"number","description":"Maximum amount per trade."},"stop_loss":{"type":"number","description":"Stop-loss percentage (e.g. 5.0 for 5%)."}},"required":["name"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Name         string  `json:"name"`
			Description  string  `json:"description"`
			RSIPeriod    int     `json:"rsi_period"`
			RSIThreshold float64 `json:"rsi_threshold"`
			SMAPeriod    int     `json:"sma_period"`
			SMATrigger   float64 `json:"sma_trigger"`
			MaxPosition  float64 `json:"max_position"`
			StopLoss     float64 `json:"stop_loss"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		st := &domain.Strategy{
			Name:        args.Name,
			Description: args.Description,
			Parameters: domain.StrategyParams{
				RSIPeriod:    args.RSIPeriod,
				RSIThreshold: args.RSIThreshold,
				SMAPeriod:    args.SMAPeriod,
				SMATrigger:   args.SMATrigger,
				MaxPosition:  args.MaxPosition,
				StopLoss:     args.StopLoss,
			},
		}
		if err := runtime.DataStore().SaveStrategy(ctx, st); err != nil {
			return nil, fmt.Errorf("saving strategy: %w", err)
		}
		return newToolResult(st)
	})
}

func addStrategyGetTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "strategy_get",
		Description: "Gets the full strategy details for the given ID.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"ID of the strategy to return."}},"required":["id"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		st, err := runtime.DataStore().GetStrategy(ctx, args.ID)
		if err != nil {
			return nil, fmt.Errorf("getting strategy %q: %w", args.ID, err)
		}
		return newToolResult(st)
	})
}

func addStrategyListTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "strategy_list",
		Description: "Lists all strategies.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sts, err := runtime.DataStore().ListStrategies(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing strategies: %w", err)
		}
		return newToolResult(sts)
	})
}

func addStrategyDeleteTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "strategy_delete",
		Description: "Deletes a strategy.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"ID of the strategy to delete."}},"required":["id"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		if err := runtime.DataStore().DeleteStrategy(ctx, args.ID); err != nil {
			return nil, fmt.Errorf("deleting strategy %q: %w", args.ID, err)
		}
		return newToolResult(map[string]string{"status": "deleted", "id": args.ID})
	})
}
