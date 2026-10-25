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

var testListing finance.Listing = *finance.NewListing("test", finance.ListingAttributeKeySymbol).
	InstrumentName("test instrument").
	InstrumentType(finance.InstrumentTypeEquity).
	Attribute(finance.ListingAttributeKeySymbol, "test").
	Build().MustValidate()

// TestAddPositionRequiresRegisteredListing pins the registry contract: a listing
// only enters the store via AddListing, i.e. as a result the server handed out.
// A position that refers to an unknown ID is rejected instead of creating the
// instrument on the fly - a client must not be able to define what an instrument
// is called.
func TestAddPositionRequiresRegisteredListing(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()

	portfolio := &domain.Portfolio{
		Name:        t.Name(),
		Description: t.Name() + " portfolio",
	}
	err := store.CreatePortfolio(t.Context(), portfolio)
	require.NoError(t, err)

	position := &domain.Position{
		Listing:    testListing,
		Currency:   finance.CurrencyUSD,
		Quantity:   1.0,
		EntryPrice: 100.0,
		EntryDate:  time.Date(2026, 10, 17, 0, 0, 0, 0, time.Local),
	}
	err = store.AddPosition(t.Context(), portfolio.ID, position)
	require.ErrorIs(t, err, data.ErrUnknownListing)
	require.Empty(t, position.ID)
}

// TestAddPositionUsesRegisteredListing makes sure a position carries the stored
// description of its instrument, not whatever the caller supplied.
func TestAddPositionUsesRegisteredListing(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()

	require.NoError(t, store.AddListing(t.Context(), testListing))

	portfolio := &domain.Portfolio{
		Name:        t.Name(),
		Description: t.Name() + " portfolio",
	}
	require.NoError(t, store.CreatePortfolio(t.Context(), portfolio))

	// The caller passes an identity-only listing, as it arrives over the wire.
	identityOnly, err := finance.ParseListingID(testListing.ID())
	require.NoError(t, err)
	require.Empty(t, identityOnly.InstrumentName())

	position := &domain.Position{
		Listing:    identityOnly,
		Currency:   finance.CurrencyUSD,
		Quantity:   1.0,
		EntryPrice: 100.0,
		EntryDate:  time.Date(2026, 10, 17, 0, 0, 0, 0, time.Local),
	}
	err = store.AddPosition(t.Context(), portfolio.ID, position)
	require.NoError(t, err)
	require.NotEmpty(t, position.ID)
	require.Equal(t, testListing.ID(), position.Listing.ID())
	require.Equal(t, "test instrument", position.Listing.InstrumentName())

	listings, err := store.ListPositionListings(t.Context())
	require.NoError(t, err)
	require.Len(t, listings, 1)
}

// TestResolveListing covers the lookup that replaces trusting the wire form.
func TestResolveListing(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()
	require.NoError(t, store.AddListing(t.Context(), testListing))

	identityOnly, err := finance.ParseListingID(testListing.ID())
	require.NoError(t, err)

	resolved, err := store.ResolveListing(t.Context(), identityOnly)
	require.NoError(t, err)
	require.Equal(t, testListing.String(), resolved.String())
	require.Equal(t, testListing.InstrumentName(), resolved.InstrumentName())
	require.Equal(t, testListing.InstrumentType(), resolved.InstrumentType())

	unknown, err := finance.ParseListingID("test:symbol=unknown")
	require.NoError(t, err)
	_, err = store.ResolveListing(t.Context(), unknown)
	require.ErrorIs(t, err, data.ErrUnknownListing)
}

// TestAddListingRejectsIncomplete makes sure an identity-only listing never
// reaches the store: it would be a row without a description, and Validate is
// what keeps that out.
func TestAddListingRejectsIncomplete(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()

	identityOnly, err := finance.ParseListingID(testListing.ID())
	require.NoError(t, err)
	err = store.AddListing(t.Context(), identityOnly)
	require.Error(t, err)
}

func TestPortfolio(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()

	err := store.AddListing(t.Context(), testListing)
	require.NoError(t, err)

	// Create
	p1 := &domain.Portfolio{
		Name:        t.Name(),
		Description: t.Name() + " portfolio",
		Positions: []domain.Position{
			{
				Listing:    testListing,
				Currency:   finance.CurrencyUSD,
				Quantity:   12.0,
				EntryPrice: 123.4,
				EntryDate:  time.Date(2026, 7, 10, 23, 56, 0, 0, time.Local),
				Notes:      t.Name() + "notes",
			},
		},
	}
	err = store.CreatePortfolio(t.Context(), p1)
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
		Listing:    testListing,
		Currency:   finance.CurrencyUSD,
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

	// List symbols (1 entries)
	listings, err := store.ListPositionListings(t.Context())
	require.NoError(t, err)
	require.Len(t, listings, 1)

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

	err := store.AddListing(t.Context(), testListing)
	require.NoError(t, err)

	// Add (multiple)
	quotes := []domain.Quote{
		{
			Listing:         testListing,
			Timestamp:       time.Date(2026, 7, 11, 8, 12, 0, 0, time.Local),
			Currency:        "USD",
			Open:            1.0,
			High:            10.0,
			Low:             0.0,
			PreviousClose:   2.0,
			Price:           2.0,
			Volume:          100,
			Source:          t.Name(),
			SourceTimestamp: time.Date(2026, 7, 11, 8, 14, 0, 0, time.Local),
		},
		{
			Listing:         testListing,
			Timestamp:       time.Date(2026, 7, 11, 8, 15, 0, 0, time.Local),
			Currency:        "USD",
			Open:            2.0,
			High:            5.0,
			Low:             1.0,
			PreviousClose:   3.0,
			Price:           3.0,
			Volume:          1000,
			Source:          t.Name(),
			SourceTimestamp: time.Date(2026, 7, 11, 8, 16, 0, 0, time.Local),
		},
	}
	err = store.SaveQuotes(t.Context(), quotes)
	require.NoError(t, err)

	// Select (latest)
	quote, err := store.GetLatestQuote(t.Context(), testListing)
	require.NoError(t, err)
	require.Equal(t, &quotes[1], quote)
}

func TestAlert(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()

	err := store.AddListing(t.Context(), testListing)
	require.NoError(t, err)

	// Create
	a1 := &domain.Alert{
		Name:      t.Name() + " alert",
		Listing:   testListing,
		Currency:  "USD",
		Condition: domain.AlertPriceBelow,
		Threshold: 35.0,
		State:     domain.AlertStateArmed,
		Message:   t.Name() + " alert",
	}
	err = store.CreateAlert(t.Context(), a1)
	require.NoError(t, err)

	// Get
	a2, err := store.GetAlert(t.Context(), a1.ID)
	require.NoError(t, err)
	require.Equal(t, a1, a2)

	// List (symbol)
	as, err := store.ListAlerts(t.Context(), testListing)
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
	as, err = store.ListAlerts(t.Context(), testListing)
	require.NoError(t, err)
	require.Len(t, as, 0)
}

func TestTrade(t *testing.T) {
	store := newDataStore(t)
	defer store.Close()

	err := store.AddListing(t.Context(), testListing)
	require.NoError(t, err)
	st1 := &domain.Strategy{
		Name:        t.Name() + " strategy",
		Description: "strategy description",
		Parameters:  domain.StrategyParams{},
	}
	err = store.SaveStrategy(t.Context(), st1)
	require.NoError(t, err)

	// Record
	t1 := &domain.Trade{
		StrategyID: st1.ID,
		Listing:    testListing,
		Currency:   "USD",
		Direction:  domain.TradeBuy,
		Quantity:   12.3,
		Price:      45.6,
		ExecutedAt: time.Date(2026, 7, 12, 10, 53, 0, 0, time.Local),
		Status:     domain.TradeExecuted,
		PnL:        7.8,
		Notes:      t.Name() + " note",
	}
	err = store.RecordTrade(t.Context(), t1)
	require.NoError(t, err)

	// Get
	t2, err := store.GetTrade(t.Context(), t1.ID)
	require.NoError(t, err)
	require.Equal(t, t1, t2)

	// List (1 entry)
	ts, err := store.ListTrades(t.Context(), t1.Listing)
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

	// Save
	st1 := &domain.Strategy{
		Name:        t.Name() + " strategy",
		Description: "strategy description",
		Parameters:  domain.StrategyParams{},
	}
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
