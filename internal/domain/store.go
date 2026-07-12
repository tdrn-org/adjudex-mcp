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

import (
	"context"
	"time"
)

// PortfolioStore defines the persistence interface for portfolios and positions.
type PortfolioStore interface {
	CreatePortfolio(ctx context.Context, p *Portfolio) error
	GetPortfolio(ctx context.Context, id string) (*Portfolio, error)
	ListPortfolios(ctx context.Context) ([]Portfolio, error)
	DeletePortfolio(ctx context.Context, id string) error
	AddPosition(ctx context.Context, portfolioID string, pos *Position) error
	RemovePosition(ctx context.Context, portfolioID string, positionID string) error
	UpdatePosition(ctx context.Context, portfolioID string, pos *Position) error
	// ListSymbols returns all distinct symbols across all portfolios and positions
	// including the timestamp of their latest quote.
	// Used by the background quote poller to know which symbols to fetch.
	ListSymbols(ctx context.Context) (map[string]time.Time, error)
}

// QuoteStore defines the persistence interface for quote data.
type QuoteStore interface {
	SaveQuote(ctx context.Context, q *Quote) error
	SaveQuotes(ctx context.Context, quotes []Quote) error
	GetQuotes(ctx context.Context, symbol string, from, to time.Time) ([]Quote, error)
	GetLatestQuote(ctx context.Context, symbol string) (*Quote, error)
}

// AlertStore defines the persistence interface for alerts.
type AlertStore interface {
	CreateAlert(ctx context.Context, a *Alert) error
	GetAlert(ctx context.Context, id string) (*Alert, error)
	ListAlerts(ctx context.Context, symbol string) ([]Alert, error)
	ListArmedAlerts(ctx context.Context) ([]Alert, error)
	UpdateAlert(ctx context.Context, a *Alert) error
	DeleteAlert(ctx context.Context, id string) error
}

// TradeStore defines the persistence interface for trades.
type TradeStore interface {
	RecordTrade(ctx context.Context, t *Trade) error
	GetTrade(ctx context.Context, id string) (*Trade, error)
	ListTrades(ctx context.Context, symbol string) ([]Trade, error)
	ListTradesByStrategy(ctx context.Context, strategyID string) ([]Trade, error)
}

// StrategyStore defines the persistence interface for strategies.
type StrategyStore interface {
	SaveStrategy(ctx context.Context, s *Strategy) error
	GetStrategy(ctx context.Context, id string) (*Strategy, error)
	ListStrategies(ctx context.Context) ([]Strategy, error)
	DeleteStrategy(ctx context.Context, id string) error
}
