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
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

// RegisterPortfolioTools registers all portfolio-related MCP tools.
func RegisterPortfolioTools(srv *server.MCPServer, portfolioStore domain.PortfolioStore, quoteStore domain.QuoteStore) {
	srv.AddTool(
		mcp.NewTool("portfolio_create",
			mcp.WithDescription("Create a new portfolio for tracking investments."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Portfolio name.")),
			mcp.WithString("description", mcp.Description("Optional portfolio description.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			name, _ := args["name"].(string)
			desc, _ := args["description"].(string)
			p, err := PortfolioCreate(ctx, portfolioStore, PortfolioCreateArgs{Name: name, Description: desc})
			if err != nil {
				return nil, err
			}
			return marshalResult(p)
		},
	)

	srv.AddTool(
		mcp.NewTool("portfolio_get",
			mcp.WithDescription("Retrieve a portfolio by ID, including all positions."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Portfolio ID.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			id, _ := args["id"].(string)
			p, err := PortfolioGet(ctx, portfolioStore, PortfolioGetArgs{ID: id})
			if err != nil {
				return nil, err
			}
			return marshalResult(p)
		},
	)

	srv.AddTool(
		mcp.NewTool("portfolio_list",
			mcp.WithDescription("List all portfolios."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			portfolios, err := PortfolioList(ctx, portfolioStore, PortfolioListArgs{})
			if err != nil {
				return nil, err
			}
			return marshalResult(portfolios)
		},
	)

	srv.AddTool(
		mcp.NewTool("portfolio_delete",
			mcp.WithDescription("Delete a portfolio and all its positions."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Portfolio ID to delete.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			id, _ := args["id"].(string)
			if err := PortfolioDelete(ctx, portfolioStore, PortfolioDeleteArgs{ID: id}); err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent("Portfolio deleted successfully.")}}, nil
		},
	)

	srv.AddTool(
		mcp.NewTool("portfolio_add_position",
			mcp.WithDescription("Add a stock position to a portfolio."),
			mcp.WithString("portfolio_id", mcp.Required(), mcp.Description("Target portfolio ID.")),
			mcp.WithString("symbol", mcp.Required(), mcp.Description("Stock ticker symbol.")),
			mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Number of shares.")),
			mcp.WithNumber("entry_price", mcp.Required(), mcp.Description("Entry price per share.")),
			mcp.WithString("entry_date", mcp.Required(), mcp.Description("Entry date (RFC3339).")),
			mcp.WithString("notes", mcp.Description("Optional notes.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			pid, _ := args["portfolio_id"].(string)
			sym, _ := args["symbol"].(string)
			qty, _ := args["quantity"].(json.Number)
			price, _ := args["entry_price"].(json.Number)
			date, _ := args["entry_date"].(string)
			notes, _ := args["notes"].(string)
			qf, _ := qty.Float64()
			pf, _ := price.Float64()
			pos, err := PortfolioAddPosition(ctx, portfolioStore, PortfolioAddPositionArgs{
				PortfolioID: pid, Symbol: sym, Quantity: qf, EntryPrice: pf, EntryDate: date, Notes: notes,
			})
			if err != nil {
				return nil, err
			}
			return marshalResult(pos)
		},
	)

	srv.AddTool(
		mcp.NewTool("portfolio_remove_position",
			mcp.WithDescription("Remove a position from a portfolio."),
			mcp.WithString("portfolio_id", mcp.Required(), mcp.Description("Portfolio ID.")),
			mcp.WithString("position_id", mcp.Required(), mcp.Description("Position ID to remove.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			pid, _ := args["portfolio_id"].(string)
			posID, _ := args["position_id"].(string)
			if err := PortfolioRemovePosition(ctx, portfolioStore, PortfolioRemovePositionArgs{PortfolioID: pid, PositionID: posID}); err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent("Position removed successfully.")}}, nil
		},
	)

	srv.AddTool(
		mcp.NewTool("portfolio_get_holdings",
			mcp.WithDescription("Get current holdings for a portfolio (positions enriched with latest prices)."),
			mcp.WithString("portfolio_id", mcp.Required(), mcp.Description("Portfolio ID.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			pid, _ := args["portfolio_id"].(string)
			holdings, err := PortfolioGetHoldings(ctx, portfolioStore, quoteStore, PortfolioGetHoldingsArgs{PortfolioID: pid})
			if err != nil {
				return nil, err
			}
			return marshalResult(holdings)
		},
	)
}

// RegisterQuoteTools registers all quote-related MCP tools.
func RegisterQuoteTools(srv *server.MCPServer, quoteStore domain.QuoteStore) {
	srv.AddTool(
		mcp.NewTool("quote_get",
			mcp.WithDescription("Get the latest price quote for a stock symbol."),
			mcp.WithString("symbol", mcp.Required(), mcp.Description("Stock ticker symbol (e.g., AAPL, MSFT).")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			sym, _ := args["symbol"].(string)
			q, err := QuoteGet(ctx, quoteStore, QuoteGetArgs{Symbol: sym})
			if err != nil {
				return nil, err
			}
			return marshalResult(q)
		},
	)

	srv.AddTool(
		mcp.NewTool("quote_history",
			mcp.WithDescription("Get historical price quotes for a stock symbol."),
			mcp.WithString("symbol", mcp.Required(), mcp.Description("Stock ticker symbol.")),
			mcp.WithString("from", mcp.Required(), mcp.Description("Start date (RFC3339).")),
			mcp.WithString("to", mcp.Required(), mcp.Description("End date (RFC3339).")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			sym, _ := args["symbol"].(string)
			from, _ := args["from"].(string)
			to, _ := args["to"].(string)
			quotes, err := QuoteHistory(ctx, quoteStore, QuoteHistoryArgs{Symbol: sym, From: from, To: to})
			if err != nil {
				return nil, err
			}
			return marshalResult(quotes)
		},
	)

	srv.AddTool(
		mcp.NewTool("quote_indicator",
			mcp.WithDescription("Compute a technical indicator from historical quotes."),
			mcp.WithString("symbol", mcp.Required(), mcp.Description("Stock ticker symbol.")),
			mcp.WithString("indicator_type", mcp.Required(), mcp.Description("Indicator type: rsi, sma, ema, macd.")),
			mcp.WithNumber("period", mcp.Required(), mcp.Description("Lookback period (e.g., 14 for RSI).")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			sym, _ := args["symbol"].(string)
			ind, _ := args["indicator_type"].(string)
			periodFloat, _ := args["period"].(json.Number)
			period, _ := periodFloat.Int64()
			result, err := QuoteIndicator(ctx, quoteStore, QuoteIndicatorArgs{Symbol: sym, IndicatorType: ind, Period: int(period)})
			if err != nil {
				return nil, err
			}
			return marshalResult(result)
		},
	)
}

// RegisterAlertTools registers all alert-related MCP tools.
func RegisterAlertTools(srv *server.MCPServer, alertStore domain.AlertStore) {
	srv.AddTool(
		mcp.NewTool("alert_create",
			mcp.WithDescription("Create a new price or indicator alert."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Alert name.")),
			mcp.WithString("symbol", mcp.Required(), mcp.Description("Stock symbol to watch.")),
			mcp.WithString("condition", mcp.Required(), mcp.Description("Condition: price_above, price_below, rsi_above, rsi_below, sma_cross_above, sma_cross_below, volume_spike.")),
			mcp.WithNumber("threshold", mcp.Required(), mcp.Description("Trigger threshold value.")),
			mcp.WithNumber("indicator_period", mcp.Description("Optional indicator period (e.g., 14 for RSI).")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			name, _ := args["name"].(string)
			sym, _ := args["symbol"].(string)
			cond, _ := args["condition"].(string)
			threshFloat, _ := args["threshold"].(json.Number)
			thresh, _ := threshFloat.Float64()
			var indicator *domain.IndicatorSpec
			if periodVal, ok := args["indicator_period"].(json.Number); ok {
				p, err := periodVal.Int64()
				if err == nil {
					indicator = &domain.IndicatorSpec{Type: domain.IndicatorType(cond), Period: int(p)}
				}
			}
			a, err := AlertCreate(ctx, alertStore, AlertCreateArgs{
				Name: name, Symbol: sym, Condition: cond, Threshold: thresh, Indicator: indicator,
			})
			if err != nil {
				return nil, err
			}
			return marshalResult(a)
		},
	)

	srv.AddTool(
		mcp.NewTool("alert_list",
			mcp.WithDescription("List all alerts, optionally filtered by symbol."),
			mcp.WithString("symbol", mcp.Description("Optional filter by stock symbol.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			sym, _ := args["symbol"].(string)
			alerts, err := AlertList(ctx, alertStore, AlertListArgs{Symbol: sym})
			if err != nil {
				return nil, err
			}
			return marshalResult(alerts)
		},
	)

	srv.AddTool(
		mcp.NewTool("alert_acknowledge",
			mcp.WithDescription("Acknowledge a triggered alert."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Alert ID to acknowledge.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			id, _ := args["id"].(string)
			a, err := AlertAcknowledge(ctx, alertStore, AlertAcknowledgeArgs{ID: id})
			if err != nil {
				return nil, err
			}
			return marshalResult(a)
		},
	)

	srv.AddTool(
		mcp.NewTool("alert_delete",
			mcp.WithDescription("Delete an alert."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Alert ID to delete.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			id, _ := args["id"].(string)
			if err := AlertDelete(ctx, alertStore, AlertDeleteArgs{ID: id}); err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent("Alert deleted successfully.")}}, nil
		},
	)
}

// RegisterStrategyTools registers all strategy-related MCP tools.
func RegisterStrategyTools(srv *server.MCPServer, strategyStore domain.StrategyStore, quoteStore domain.QuoteStore, tradeStore domain.TradeStore) {
	srv.AddTool(
		mcp.NewTool("strategy_create",
			mcp.WithDescription("Create a new trading strategy."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Strategy name.")),
			mcp.WithString("description", mcp.Description("Optional description.")),
			mcp.WithNumber("rsi_period", mcp.Description("RSI period (default: 14).")),
			mcp.WithNumber("rsi_threshold", mcp.Description("RSI threshold (default: 30.0).")),
			mcp.WithNumber("sma_period", mcp.Description("SMA period (default: 20).")),
			mcp.WithNumber("sma_trigger", mcp.Description("SMA trigger percentage (default: 5.0).")),
			mcp.WithNumber("max_position", mcp.Description("Max position size (default: 1000.0).")),
			mcp.WithNumber("stop_loss", mcp.Description("Stop loss percentage (default: 5.0).")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			name, _ := args["name"].(string)
			desc, _ := args["description"].(string)
			rsiPer := intOrNil(args["rsi_period"], 14)
			rsiThr := floatOrNil(args["rsi_threshold"], 30.0)
			smaPer := intOrNil(args["sma_period"], 20)
			smaTrg := floatOrNil(args["sma_trigger"], 5.0)
			maxPos := floatOrNil(args["max_position"], 1000.0)
			stopLoss := floatOrNil(args["stop_loss"], 5.0)
			s, err := StrategyCreate(ctx, strategyStore, StrategyCreateArgs{
				Name: name, Description: desc,
				Params: domain.StrategyParams{
					RSIPeriod: rsiPer, RSIThreshold: rsiThr,
					SMAPeriod: smaPer, SMATrigger: smaTrg,
					MaxPosition: maxPos, StopLoss: stopLoss,
				},
			})
			if err != nil {
				return nil, err
			}
			return marshalResult(s)
		},
	)

	srv.AddTool(
		mcp.NewTool("strategy_get",
			mcp.WithDescription("Retrieve a strategy by ID."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Strategy ID.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			id, _ := args["id"].(string)
			s, err := StrategyGet(ctx, strategyStore, StrategyGetArgs{ID: id})
			if err != nil {
				return nil, err
			}
			return marshalResult(s)
		},
	)

	srv.AddTool(
		mcp.NewTool("strategy_list",
			mcp.WithDescription("List all trading strategies."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			strategies, err := StrategyList(ctx, strategyStore, StrategyListArgs{})
			if err != nil {
				return nil, err
			}
			return marshalResult(strategies)
		},
	)

	srv.AddTool(
		mcp.NewTool("strategy_delete",
			mcp.WithDescription("Delete a strategy."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Strategy ID to delete.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			id, _ := args["id"].(string)
			if err := StrategyDelete(ctx, strategyStore, StrategyDeleteArgs{ID: id}); err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent("Strategy deleted successfully.")}}, nil
		},
	)

	srv.AddTool(
		mcp.NewTool("strategy_backtest",
			mcp.WithDescription("Run a strategy backtest against historical data."),
			mcp.WithString("strategy_id", mcp.Required(), mcp.Description("Strategy ID to backtest.")),
			mcp.WithString("symbol", mcp.Required(), mcp.Description("Stock symbol to test.")),
			mcp.WithString("from", mcp.Required(), mcp.Description("Start date (RFC3339).")),
			mcp.WithString("to", mcp.Required(), mcp.Description("End date (RFC3339).")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			sid, _ := args["strategy_id"].(string)
			sym, _ := args["symbol"].(string)
			from, _ := args["from"].(string)
			to, _ := args["to"].(string)
			result, err := StrategyBacktest(ctx, strategyStore, quoteStore, tradeStore, StrategyBacktestArgs{
				StrategyID: sid, Symbol: sym, From: from, To: to,
			})
			if err != nil {
				return nil, err
			}
			return marshalResult(result)
		},
	)
}

// RegisterTradeTools registers all trade-related MCP tools.
func RegisterTradeTools(srv *server.MCPServer, tradeStore domain.TradeStore) {
	srv.AddTool(
		mcp.NewTool("trade_list",
			mcp.WithDescription("List trades, optionally filtered by symbol or strategy."),
			mcp.WithString("symbol", mcp.Description("Optional filter by stock symbol.")),
			mcp.WithString("strategy_id", mcp.Description("Optional filter by strategy ID.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			sym, _ := args["symbol"].(string)
			sid, _ := args["strategy_id"].(string)
			trades, err := TradeList(ctx, tradeStore, TradeListArgs{Symbol: sym, StrategyID: sid})
			if err != nil {
				return nil, err
			}
			return marshalResult(trades)
		},
	)

	srv.AddTool(
		mcp.NewTool("trade_get",
			mcp.WithDescription("Retrieve a single trade by ID."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Trade ID.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			id, _ := args["id"].(string)
			t, err := TradeGet(ctx, tradeStore, TradeGetArgs{ID: id})
			if err != nil {
				return nil, err
			}
			return marshalResult(t)
		},
	)
}

// marshalResult marshals any value to a JSON MCP result.
func marshalResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(b))}}, nil
}

// intOrNil returns the int from a json.Number, or def if missing/invalid.
func intOrNil(v any, def int) int {
	n, ok := v.(json.Number)
	if !ok {
		return def
	}
	i, err := n.Int64()
	if err != nil {
		return def
	}
	return int(i)
}

// floatOrNil returns the float from a json.Number, or def if missing/invalid.
func floatOrNil(v any, def float64) float64 {
	n, ok := v.(json.Number)
	if !ok {
		return def
	}
	f, err := n.Float64()
	if err != nil {
		return def
	}
	return f
}

// Keep imports alive.
var _ = time.Now
