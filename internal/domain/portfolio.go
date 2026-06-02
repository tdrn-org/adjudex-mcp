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

// Package domain defines the core types and interfaces for adjudex.
// This package MUST NOT import anything outside the Go standard library.
// This ensures the domain model remains pure and free of infrastructure concerns.
//
// See AGENT.md §5 "Domain layer isolation" for the full rationale.
package domain

import "time"

// Portfolio represents a named collection of positions (watchlist or holdings).
type Portfolio struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Positions   []Position `json:"positions"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Position represents a tracked security within a portfolio.
type Position struct {
	ID         string    `json:"id"`
	Symbol     string    `json:"symbol"` // WKN or ticker (e.g., "A413X6", "NVDA")
	Quantity   float64   `json:"quantity"`
	EntryPrice float64   `json:"entry_price"`
	EntryDate  time.Time `json:"entry_date"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Holding is a Position enriched with current market data.
// It is not persisted — derived at query time from Position + Quote.
type Holding struct {
	Position
	CurrentPrice float64 `json:"current_price"`
	MarketValue  float64 `json:"market_value"`
	PnL          float64 `json:"pnl"`
	PnLPercent   float64 `json:"pnl_pct"`
}

// NewHolding derives a Holding from a Position and current price.
// Returns by-value (not pointer) because Holding is a derived Value Object —
// it is never persisted or mutated, only computed and consumed. This avoids
// nil-check boilerplate and reduces GC pressure (stack allocation for ~224 bytes).
func NewHolding(pos Position, currentPrice float64) Holding {
	marketValue := pos.Quantity * currentPrice
	pnl := marketValue - (pos.Quantity * pos.EntryPrice)
	var pnlPct float64
	if pos.EntryPrice > 0 {
		pnlPct = ((currentPrice - pos.EntryPrice) / pos.EntryPrice) * 100
	}
	return Holding{
		Position:     pos,
		CurrentPrice: currentPrice,
		MarketValue:  marketValue,
		PnL:          pnl,
		PnLPercent:   pnlPct,
	}
}
