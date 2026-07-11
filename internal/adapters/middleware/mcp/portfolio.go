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

func addPortfolioTools(server *mcp.Server, runtime Runtime) {
	addPortfolioCreateTool(server, runtime)
	addPortfolioGetTool(server, runtime)
	addPortfolioListTool(server, runtime)
	addPortfolioDeleteTool(server, runtime)
	addPositionAddTool(server, runtime)
	addPositionRemoveTool(server, runtime)
	addPositionUpdateTool(server, runtime)
	addPortfolioHoldingsTool(server, runtime)
}

func addPortfolioCreateTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "portfolio_create",
		Description: "Creates a new portfolio. The created portfolio details are returned.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"The name of the portfolio."},"description":{"type":"string","description":"An optional description for the portfolio."}},"required":["name"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		p := &domain.Portfolio{
			Name:        args.Name,
			Description: args.Description,
		}
		if err := runtime.DataStore().CreatePortfolio(ctx, p); err != nil {
			return nil, fmt.Errorf("creating portfolio: %w", err)
		}
		return newToolResult(p)
	})
}

func addPortfolioGetTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "portfolio_get",
		Description: "Gets the full portfolio details for the given ID including positions.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"ID of the portfolio to return."}},"required":["id"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		p, err := runtime.DataStore().GetPortfolio(ctx, args.ID)
		if err != nil {
			return nil, fmt.Errorf("getting portfolio %q: %w", args.ID, err)
		}
		return newToolResult(p)
	})
}

func addPortfolioListTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "portfolio_list",
		Description: "Lists all portfolios. Returns summaries without positions for performance.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ps, err := runtime.DataStore().ListPortfolios(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing portfolios: %w", err)
		}
		return newToolResult(ps)
	})
}

func addPortfolioDeleteTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "portfolio_delete",
		Description: "Deletes a portfolio and all its positions.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"ID of the portfolio to delete."}},"required":["id"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		if err := runtime.DataStore().DeletePortfolio(ctx, args.ID); err != nil {
			return nil, fmt.Errorf("deleting portfolio %q: %w", args.ID, err)
		}
		return newToolResult(map[string]string{"status": "deleted", "id": args.ID})
	})
}

func addPositionAddTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "position_add",
		Description: "Adds a new position to a portfolio. The created position details are returned.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"portfolio_id":{"type":"string","description":"ID of the portfolio to add the position to."},"symbol":{"type":"string","description":"The ticker symbol or WKN of the security."},"quantity":{"type":"number","description":"The number of shares or units."},"entry_price":{"type":"number","description":"The purchase price per share."},"entry_date":{"type":"string","description":"The date of purchase (RFC3339 format, e.g. 2026-07-11T00:00:00Z)."},"notes":{"type":"string","description":"Optional notes about this position."}},"required":["portfolio_id","symbol","quantity","entry_price","entry_date"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			PortfolioID string  `json:"portfolio_id"`
			Symbol      string  `json:"symbol"`
			Quantity    float64 `json:"quantity"`
			EntryPrice  float64 `json:"entry_price"`
			EntryDate   string  `json:"entry_date"`
			Notes       string  `json:"notes"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		entryDate, err := time.Parse(time.RFC3339, args.EntryDate)
		if err != nil {
			return nil, fmt.Errorf("parsing entry_date: %w", err)
		}

		pos := &domain.Position{
			Symbol:     args.Symbol,
			Quantity:   args.Quantity,
			EntryPrice: args.EntryPrice,
			EntryDate:  entryDate,
			Notes:      args.Notes,
		}
		if err := runtime.DataStore().AddPosition(ctx, args.PortfolioID, pos); err != nil {
			return nil, fmt.Errorf("adding position to portfolio %q: %w", args.PortfolioID, err)
		}
		return newToolResult(pos)
	})
}

func addPositionRemoveTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "position_remove",
		Description: "Removes a position from a portfolio.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"portfolio_id":{"type":"string","description":"ID of the portfolio."},"position_id":{"type":"string","description":"ID of the position to remove."}},"required":["portfolio_id","position_id"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			PortfolioID string `json:"portfolio_id"`
			PositionID  string `json:"position_id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		if err := runtime.DataStore().RemovePosition(ctx, args.PortfolioID, args.PositionID); err != nil {
			return nil, fmt.Errorf("removing position %q: %w", args.PositionID, err)
		}
		return newToolResult(map[string]string{"status": "removed", "id": args.PositionID})
	})
}

func addPositionUpdateTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "position_update",
		Description: "Updates an existing position. Only the provided fields are updated (PATCH semantics).",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"portfolio_id":{"type":"string","description":"ID of the portfolio."},"position_id":{"type":"string","description":"ID of the position to update."},"quantity":{"type":"number","description":"New quantity."},"entry_price":{"type":"number","description":"New entry price."},"notes":{"type":"string","description":"New notes."}},"required":["portfolio_id","position_id"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			PortfolioID string   `json:"portfolio_id"`
			PositionID  string   `json:"position_id"`
			Quantity    *float64 `json:"quantity"`
			EntryPrice  *float64 `json:"entry_price"`
			Notes       *string  `json:"notes"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		p, err := runtime.DataStore().GetPortfolio(ctx, args.PortfolioID)
		if err != nil {
			return nil, fmt.Errorf("getting portfolio %q: %w", args.PortfolioID, err)
		}
		var pos *domain.Position
		for i := range p.Positions {
			if p.Positions[i].ID == args.PositionID {
				pos = &p.Positions[i]
				break
			}
		}
		if pos == nil {
			return nil, fmt.Errorf("position %q not found in portfolio %q", args.PositionID, args.PortfolioID)
		}

		if args.Quantity != nil {
			pos.Quantity = *args.Quantity
		}
		if args.EntryPrice != nil {
			pos.EntryPrice = *args.EntryPrice
		}
		if args.Notes != nil {
			pos.Notes = *args.Notes
		}

		if err := runtime.DataStore().UpdatePosition(ctx, args.PortfolioID, pos); err != nil {
			return nil, fmt.Errorf("updating position %q: %w", args.PositionID, err)
		}
		return newToolResult(pos)
	})
}

func addPortfolioHoldingsTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "portfolio_holdings",
		Description: "Returns the portfolio with all positions enriched with current market data (PnL, market value). Fetches live quotes for each position.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"portfolio_id":{"type":"string","description":"ID of the portfolio."}},"required":["portfolio_id"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			PortfolioID string `json:"portfolio_id"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		p, err := runtime.DataStore().GetPortfolio(ctx, args.PortfolioID)
		if err != nil {
			return nil, fmt.Errorf("getting portfolio %q: %w", args.PortfolioID, err)
		}

		holdings := make([]domain.Holding, 0, len(p.Positions))
		for _, pos := range p.Positions {
			quote, err := runtime.StockTracker().FetchQuote(ctx, pos.Symbol)
			if err != nil {
				runtime.Logger().Warn("failed to fetch quote for holding", "symbol", pos.Symbol, "err", err)
				continue
			}
			holding := domain.NewHolding(pos, quote.Price)
			holdings = append(holdings, holding)
		}

		return newToolResult(holdings)
	})
}

func newToolResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshaling result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(b)},
		},
	}, nil
}
