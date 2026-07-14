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
	Symbol     string    `json:"symbol"`
	Currency   string    `json:"currency"`
	Quantity   float64   `json:"quantity"`
	EntryPrice float64   `json:"entry_price"`
	EntryDate  time.Time `json:"entry_date"`
	Notes      string    `json:"notes,omitempty"`
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
	PnLPercent   float64 `json:"pnl_percent"`
}

// NewHolding derives a Holding from a Position and current price.
// Returns by-value (not pointer) because Holding is a derived Value Object —
// it is never persisted or mutated, only computed and consumed. This avoids
// nil-check boilerplate and reduces GC pressure (stack allocation for ~224 bytes).
func NewHolding(pos Position, currentPrice float64) Holding {
	totalEntryPrice := pos.Quantity * pos.EntryPrice
	marketValue := pos.Quantity * currentPrice
	pnl := marketValue - totalEntryPrice
	var pnlPercent float64
	if totalEntryPrice > 0 {
		pnlPercent = (pnl / totalEntryPrice) * 100
	}
	return Holding{
		Position:     pos,
		CurrentPrice: currentPrice,
		MarketValue:  marketValue,
		PnL:          pnl,
		PnLPercent:   pnlPercent,
	}
}

type Holdings []Holding

type HoldingsSummary struct {
	Currency    string  `json:"currency"`
	MarketValue float64 `json:"market_value"`
	PnL         float64 `json:"pnl"`
	PnLPercent  float64 `json:"pnl_percent"`
}

func (holdings Holdings) Summarize() HoldingsSummary {
	totalEntryPrice := 0.0
	totalMarketValue := 0.0
	for _, holding := range holdings {
		totalEntryPrice += holding.Quantity * holding.EntryPrice
		totalMarketValue += holding.MarketValue
	}
	pnl := totalMarketValue - totalEntryPrice
	var pnlPercent float64
	if totalEntryPrice > 0 {
		pnlPercent = (pnl / totalEntryPrice) * 100
	}
	summary := HoldingsSummary{
		MarketValue: totalMarketValue,
		PnL:         pnl,
		PnLPercent:  pnlPercent,
	}
	return summary
}
