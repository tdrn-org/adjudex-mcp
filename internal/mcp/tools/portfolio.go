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

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

// --- Tool: portfolio_create ---

// PortfolioCreateArgs are the parameters for portfolio_create.
type PortfolioCreateArgs struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// PortfolioCreate creates a new portfolio.
// Store: domain.PortfolioStore.CreatePortfolio
func PortfolioCreate(ctx context.Context, store domain.PortfolioStore, args PortfolioCreateArgs) (*domain.Portfolio, error) {
	panic("not implemented: Phase 5")
}

// --- Tool: portfolio_get ---

// PortfolioGetArgs are the parameters for portfolio_get.
type PortfolioGetArgs struct {
	ID string `json:"id"`
}

// PortfolioGet retrieves a portfolio by ID, including its positions.
// Store: domain.PortfolioStore.GetPortfolio
func PortfolioGet(ctx context.Context, store domain.PortfolioStore, args PortfolioGetArgs) (*domain.Portfolio, error) {
	panic("not implemented: Phase 5")
}

// --- Tool: portfolio_list ---

// PortfolioListArgs are the parameters for portfolio_list (no parameters).
type PortfolioListArgs struct{}

// PortfolioList returns all portfolios.
// Store: domain.PortfolioStore.ListPortfolios
func PortfolioList(ctx context.Context, store domain.PortfolioStore, args PortfolioListArgs) ([]domain.Portfolio, error) {
	panic("not implemented: Phase 5")
}

// --- Tool: portfolio_delete ---

// PortfolioDeleteArgs are the parameters for portfolio_delete.
type PortfolioDeleteArgs struct {
	ID string `json:"id"`
}

// PortfolioDelete removes a portfolio and its positions.
// Store: domain.PortfolioStore.DeletePortfolio
func PortfolioDelete(ctx context.Context, store domain.PortfolioStore, args PortfolioDeleteArgs) error {
	panic("not implemented: Phase 5")
}

// --- Tool: portfolio_add_position ---

// PortfolioAddPositionArgs are the parameters for portfolio_add_position.
type PortfolioAddPositionArgs struct {
	PortfolioID string  `json:"portfolio_id"`
	Symbol      string  `json:"symbol"`
	Quantity    float64 `json:"quantity"`
	EntryPrice  float64 `json:"entry_price"`
	EntryDate   string  `json:"entry_date"` // RFC3339
	Notes       string  `json:"notes,omitempty"`
}

// PortfolioAddPosition adds a position to a portfolio.
// Store: domain.PortfolioStore.AddPosition
func PortfolioAddPosition(ctx context.Context, store domain.PortfolioStore, args PortfolioAddPositionArgs) (*domain.Position, error) {
	panic("not implemented: Phase 5")
}

// --- Tool: portfolio_remove_position ---

// PortfolioRemovePositionArgs are the parameters for portfolio_remove_position.
type PortfolioRemovePositionArgs struct {
	PortfolioID string `json:"portfolio_id"`
	PositionID  string `json:"position_id"`
}

// PortfolioRemovePosition removes a position from a portfolio.
// Store: domain.PortfolioStore.RemovePosition
func PortfolioRemovePosition(ctx context.Context, store domain.PortfolioStore, args PortfolioRemovePositionArgs) error {
	panic("not implemented: Phase 5")
}

// --- Tool: portfolio_get_holdings ---

// PortfolioGetHoldingsArgs are the parameters for portfolio_get_holdings.
type PortfolioGetHoldingsArgs struct {
	PortfolioID string `json:"portfolio_id"`
}

// PortfolioGetHoldings returns positions enriched with current market prices.
// Derived operation: queries Positions + latest Quotes per symbol.
// Stores: domain.PortfolioStore.GetPortfolio + domain.QuoteStore.GetLatestQuote
func PortfolioGetHoldings(ctx context.Context, portfolioStore domain.PortfolioStore, quoteStore domain.QuoteStore, args PortfolioGetHoldingsArgs) ([]domain.Holding, error) {
	_ = ctx
	_ = portfolioStore
	_ = quoteStore
	_ = args
	panic("not implemented: Phase 5")
}
