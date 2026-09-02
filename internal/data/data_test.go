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

package data_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/adjudex-mcp/internal/data"
	"github.com/tdrn-org/adjudex-mcp/internal/data/model"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-database/memory"
	"github.com/tdrn-org/go-finance"
)

func TestUpdateScheme(t *testing.T) {
	store := newDataStore(t)
	err := store.Close()
	require.NoError(t, err)
}

func TestSymbol(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()

	s1 := &finance.Symbol{
		Exchange: "XNGS",
		Ticker:   "AAPL",
		ISIN:     "US0378331005",
		WKN:      "865985",
		FIGI:     "BBG000B9XRY4",
		Name:     "Apple Inc.",
		Type:     finance.SecurityTypeEquity,
	}
	s2, err := store.MergeSymbol(t.Context(), s1)
	require.NoError(t, err)
	require.Equal(t, s1, s2)
	s3, err := store.MergeSymbol(t.Context(), s1)
	require.NoError(t, err)
	require.Equal(t, s1, s3)
}

func TestPortfolio(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()

	// Create
	p1 := &domain.Portfolio{
		Name:        t.Name(),
		Description: t.Name() + " portfolio",
		Positions: []domain.Position{
			{
				Symbol:     "XZY",
				Currency:   "USD",
				Quantity:   12.0,
				EntryPrice: 123.4,
				EntryDate:  time.Date(2026, 7, 10, 23, 56, 0, 0, time.Local),
				Notes:      t.Name() + "notes",
			},
		},
	}
	err := store.CreatePortfolio(t.Context(), p1)
	require.NoError(t, err)

	// Get
	p2, err := store.GetPortfolio(t.Context(), p1.ID)
	require.NoError(t, err)
	require.Equal(t, p1, p2)

	// List (1 result)
	ps, err := store.ListPortfolios(t.Context())
	require.NoError(t, err)
	require.Len(t, ps, 1)
	require.Equal(t, p1, &ps[0])

	// Add
	p2.Positions = append(p2.Positions, domain.Position{
		Symbol:     "ABC",
		Currency:   "USD",
		Quantity:   24.0,
		EntryPrice: 246.8,
		EntryDate:  time.Date(2026, 7, 11, 06, 49, 0, 0, time.Local),
		Notes:      t.Name() + "notes (2)",
	})
	err = store.AddPosition(t.Context(), p2.ID, &p2.Positions[1])
	require.NoError(t, err)

	// Update
	p2.Positions[1].Quantity *= 2
	err = store.UpdatePosition(t.Context(), p2.ID, &p2.Positions[1])
	require.NoError(t, err)

	// List symbols (2 entries)
	symbolMap, err := store.ListSymbols(t.Context())
	require.NoError(t, err)
	require.Len(t, symbolMap, 2)

	// Remove
	err = store.RemovePosition(t.Context(), p2.ID, p2.Positions[1].ID)
	require.NoError(t, err)

	// Delete
	err = store.DeletePortfolio(t.Context(), p1.ID)
	require.NoError(t, err)

	// List (0 results)
	ps, err = store.ListPortfolios(t.Context())
	require.NoError(t, err)
	require.Len(t, ps, 0)
}

func TestQuote(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()

	quotes := []domain.Quote{
		{
			Symbol:          "XYZ",
			Timestamp:       time.Date(2026, 7, 11, 8, 12, 0, 0, time.Local),
			Currency:        "USD",
			Open:            1.0,
			High:            10.0,
			Low:             0.0,
			Close:           2.0,
			Price:           2.0,
			Volume:          100,
			Source:          t.Name(),
			SourceTimestamp: time.Date(2026, 7, 11, 8, 14, 0, 0, time.Local),
		},
		{
			Symbol:          "ABC",
			Timestamp:       time.Date(2026, 7, 11, 8, 15, 0, 0, time.Local),
			Currency:        "USD",
			Open:            2.0,
			High:            5.0,
			Low:             1.0,
			Close:           3.0,
			Price:           3.0,
			Volume:          1000,
			Source:          t.Name(),
			SourceTimestamp: time.Date(2026, 7, 11, 8, 16, 0, 0, time.Local),
		},
	}

	// Add (multiple)
	err := store.SaveQuotes(t.Context(), quotes)
	require.NoError(t, err)

	// Select (latest)
	quote, err := store.GetLatestQuote(t.Context(), "ABC")
	require.NoError(t, err)
	require.Equal(t, &quotes[1], quote)
}

func TestAlert(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()

	a1 := &domain.Alert{
		Name:      t.Name() + " alert",
		Symbol:    "XYZ",
		Currency:  "USD",
		Condition: domain.AlertPriceBelow,
		Threshold: 35.0,
		State:     domain.AlertStateArmed,
		Message:   t.Name() + " alert",
	}

	// Create
	err := store.CreateAlert(t.Context(), a1)
	require.NoError(t, err)

	// Get
	a2, err := store.GetAlert(t.Context(), a1.ID)
	require.NoError(t, err)
	require.Equal(t, a1, a2)

	// List (symbol)
	as, err := store.ListAlerts(t.Context(), "XYZ")
	require.NoError(t, err)
	require.Len(t, as, 1)
	require.Equal(t, a1, &as[0])

	// List (armed)
	as, err = store.ListArmedAlerts(t.Context())
	require.NoError(t, err)
	require.Len(t, as, 1)
	require.Equal(t, a1, &as[0])

	// Update
	a2.Message = a2.Message + " (updated)"
	err = store.UpdateAlert(t.Context(), a2)
	require.NoError(t, err)

	// Delete
	err = store.DeleteAlert(t.Context(), a1.ID)
	require.NoError(t, err)

	// List
	as, err = store.ListAlerts(t.Context(), "XYZ")
	require.NoError(t, err)
	require.Len(t, as, 0)
}

func TestTrade(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()

	st1 := &domain.Strategy{
		Name:        t.Name() + " strategy",
		Description: "strategy description",
		Parameters:  domain.StrategyParams{},
	}
	err := store.SaveStrategy(t.Context(), st1)
	require.NoError(t, err)

	t1 := &domain.Trade{
		StrategyID: st1.ID,
		Symbol:     "XZY",
		Currency:   "USD",
		Direction:  domain.TradeBuy,
		Quantity:   12.3,
		Price:      45.6,
		ExecutedAt: time.Date(2026, 7, 12, 10, 53, 0, 0, time.Local),
		Status:     domain.TradeExecuted,
		PnL:        7.8,
		Notes:      t.Name() + " note",
	}

	// Record
	err = store.RecordTrade(t.Context(), t1)
	require.NoError(t, err)

	// Get
	t2, err := store.GetTrade(t.Context(), t1.ID)
	require.NoError(t, err)
	require.Equal(t, t1, t2)

	// List (1 entry)
	ts, err := store.ListTrades(t.Context(), t1.Symbol)
	require.NoError(t, err)
	require.Len(t, ts, 1)
	require.Equal(t, t1, &ts[0])

	// List by strategy (1 entry)
	ts, err = store.ListTradesByStrategy(t.Context(), t1.StrategyID)
	require.NoError(t, err)
	require.Len(t, ts, 1)
	require.Equal(t, t1, &ts[0])
}

func TestStrategy(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()

	st1 := &domain.Strategy{
		Name:        t.Name() + " strategy",
		Description: "strategy description",
		Parameters:  domain.StrategyParams{},
	}

	// Save
	err := store.SaveStrategy(t.Context(), st1)
	require.NoError(t, err)

	// Get
	st2, err := store.GetStrategy(t.Context(), st1.ID)
	require.NoError(t, err)
	require.Equal(t, st1, st2)

	// List (1 entry)
	sts, err := store.ListStrategies(t.Context())
	require.NoError(t, err)
	require.Len(t, sts, 1)
	require.Equal(t, st1, &sts[0])

	// Delete
	err = store.DeleteStrategy(t.Context(), st1.ID)
	require.NoError(t, err)

	// List (0 entries)
	sts, err = store.ListStrategies(t.Context())
	require.NoError(t, err)
	require.Len(t, sts, 0)
}

func newDataStore(t *testing.T) *data.Store {
	driver, err := database.Open(memory.NewConfig(model.SqliteSchemaScriptOption))
	require.NoError(t, err)
	from, to, err := driver.UpdateSchema(t.Context())
	require.NoError(t, err)
	require.Equal(t, database.SchemaNone, from)
	require.Equal(t, 1, to)
	return data.NewStore(t.Name(), driver)
}
