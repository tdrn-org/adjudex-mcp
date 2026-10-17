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

package model_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

var testQuote *domain.Quote = func() *domain.Quote {
	// One timestamp for both columns on purpose: DeleteQuoteByPK filters on
	// source_timestamp while the caller passes Timestamp, and Time2DB reduces to
	// seconds. Two separate time.Now() calls land in different seconds whenever
	// the clock ticks between them, the delete then matches nothing and the test
	// fails once in a few thousand runs. Truncating keeps them identical.
	now := time.Now().UTC().Truncate(time.Second)
	return &domain.Quote{
		Listing:         testListing,
		Timestamp:       now,
		Open:            1.0,
		High:            1.2,
		Low:             0.9,
		Close:           1.1,
		Price:           123.456,
		Volume:          42,
		Currency:        finance.CurrencyUSD,
		Source:          testListing.Provider(),
		SourceTimestamp: now,
	}
}()

var testPortfolio *domain.Portfolio = &domain.Portfolio{
	Name:        "Test portfolio",
	Description: "Test portfolio description",
	Positions: []domain.Position{{
		Listing:    testListing,
		Currency:   testQuote.Currency,
		Quantity:   12.0,
		EntryPrice: 21.0,
		EntryDate:  time.Now().UTC(),
		Notes:      "Test note",
	}},
}

var testAlert *domain.Alert = &domain.Alert{
	Name:      "Test alert",
	Listing:   testListing,
	Currency:  finance.CurrencyUSD,
	Condition: domain.AlertPriceAbove,
	Threshold: 12.0,
	State:     domain.AlertStateDisabled,
	Message:   "Test alert triggered",
}

var testStrategy *domain.Strategy = &domain.Strategy{
	Name:        "Test strategy",
	Description: "Test strategy description",
	Parameters:  domain.StrategyParams{},
}

func TestUpdateScheme(t *testing.T) {
	driver := newDatabase(t)
	err := driver.Close()
	require.NoError(t, err)
}

func TestListing(t *testing.T) {
	driver := newDatabase(t)
	defer driver.Close()

	runTestTx(t, driver, func(t *testing.T, ctx context.Context, tx *database.Tx) {
		l1, err := model.InsertListing(ctx, tx, testListing)
		require.NoError(t, err)
		require.Equal(t, testListing.ID(), l1.Listing.ID())

		l2, err := model.SelectListing(ctx, tx, l1.ID)
		require.NoError(t, err)
		require.Equal(t, l1.ID, l2.ID)
	})
}

func TestQuote(t *testing.T) {
	driver := newDatabase(t)
	defer driver.Close()

	runTestTx(t, driver, func(t *testing.T, ctx context.Context, tx *database.Tx) {
		l, err := model.InsertListing(ctx, tx, testListing)
		require.NoError(t, err)

		q1, err := model.InsertQuote(ctx, tx, testQuote)
		require.NoError(t, err)
		require.Equal(t, l.Listing, q1.Listing)

		q2, err := model.SelectLatestQuote(ctx, tx, l.Listing)
		require.NoError(t, err)
		require.Equal(t, q1.Listing, q2.Listing)
		require.Equal(t, q1.Timestamp, q2.Timestamp)

		now := time.Now()
		qs, err := model.SelectQuotes(ctx, tx, l.Listing, now.Add(-1*time.Hour), now.Add(1*time.Hour))
		require.NoError(t, err)
		require.Len(t, qs, 1)

		err = model.DeleteQuoteByPK(ctx, tx, l.Listing, database.DB2Time(q1.Timestamp))
		require.NoError(t, err)

		qs, err = model.SelectQuotes(ctx, tx, l.Listing, now.Add(-1*time.Hour), now.Add(1*time.Hour))
		require.NoError(t, err)
		require.Empty(t, qs)
	})
}

func TestPortfolio(t *testing.T) {
	driver := newDatabase(t)
	defer driver.Close()

	runTestTx(t, driver, func(t *testing.T, ctx context.Context, tx *database.Tx) {
		_, err := model.InsertListing(ctx, tx, testListing)
		require.NoError(t, err)

		p1, err := model.InsertPortfolio(ctx, tx, testPortfolio)
		require.NoError(t, err)

		p2, err := model.SelectPortfolioByID(ctx, tx, p1.ID)
		require.NoError(t, err)
		require.Equal(t, p1, p2)

		ps, err := model.SelectPortfolios(ctx, tx)
		require.NoError(t, err)
		require.Len(t, ps, 1)

		pos1, err := model.InsertPosition(ctx, tx, p1.ID, &testPortfolio.Positions[0])
		require.NoError(t, err)
		require.Equal(t, p1.ID, pos1.PortfolioID)

		pos2, err := model.UpdatePosition(ctx, tx, p1.ID, &testPortfolio.Positions[0])
		require.NoError(t, err)
		require.Equal(t, p1.ID, pos2.PortfolioID)

		poss, err := model.SelectPositionsByPortfolioID(ctx, tx, p1.ID)
		require.NoError(t, err)
		require.Len(t, poss, 1)
		require.Equal(t, pos1.ID, poss[0].ID)

		err = model.TouchPortfolioByID(ctx, tx, p1.ID)
		require.NoError(t, err)

		err = model.DeletePositionsByPortfolioID(ctx, tx, p1.ID)
		require.NoError(t, err)

		err = model.DeletePortfolioByID(ctx, tx, p1.ID)
		require.NoError(t, err)

		ps, err = model.SelectPortfolios(ctx, tx)
		require.NoError(t, err)
		require.Empty(t, ps)
	})
}

func TestAlert(t *testing.T) {
	driver := newDatabase(t)
	defer driver.Close()

	runTestTx(t, driver, func(t *testing.T, ctx context.Context, tx *database.Tx) {
		_, err := model.InsertListing(ctx, tx, testListing)
		require.NoError(t, err)

		a1, err := model.InsertAlert(ctx, tx, testAlert)
		require.NoError(t, err)

		a2, err := model.SelectAlertByID(ctx, tx, a1.ID)
		require.NoError(t, err)
		require.Equal(t, a1, a2)

		as, err := model.SelectAlertsByListing(ctx, tx, testAlert.Listing)
		require.NoError(t, err)
		require.Len(t, as, 1)
		require.Equal(t, a1, as[0])

		as, err = model.SelectAlertsByState(ctx, tx, testAlert.State)
		require.NoError(t, err)
		require.Len(t, as, 1)
		require.Equal(t, a1, as[0])

		err = model.DeleteAlertByID(ctx, tx, a1.ID)
		require.NoError(t, err)

		as, err = model.SelectAlertsByListing(ctx, tx, testAlert.Listing)
		require.NoError(t, err)
		require.Empty(t, as)
	})
}

func TestStrategy(t *testing.T) {
	driver := newDatabase(t)
	defer driver.Close()

	runTestTx(t, driver, func(t *testing.T, ctx context.Context, tx *database.Tx) {
		s1, err := model.InsertStrategy(ctx, tx, testStrategy)
		require.NoError(t, err)

		s2, err := model.SelectStrategyByID(ctx, tx, s1.ID)
		require.NoError(t, err)
		require.Equal(t, s1, s2)

		ss, err := model.SelectStrategies(ctx, tx)
		require.NoError(t, err)
		require.Len(t, ss, 1)
		require.Equal(t, s1, ss[0])

		err = model.DeleteStrategyByID(ctx, tx, s1.ID)
		require.NoError(t, err)

		ss, err = model.SelectStrategies(ctx, tx)
		require.NoError(t, err)
		require.Empty(t, ss)
	})
}

func runTestTx(t *testing.T, driver *database.Driver, testFunc func(*testing.T, context.Context, *database.Tx)) {
	txCtx, tx, err := driver.BeginTx(t.Context())
	require.NoError(t, err)
	defer tx.RollbackUncommitedTx(txCtx)
	testFunc(t, txCtx, tx)
	err = tx.CommitTx(txCtx)
	require.NoError(t, err)
}

func newDatabase(t *testing.T) *database.Driver {
	driver, err := database.Open(memory.NewConfig(model.SqliteSchemaScriptOption))
	require.NoError(t, err)
	from, to, err := driver.UpdateSchema(t.Context())
	require.NoError(t, err)
	require.Equal(t, database.SchemaNone, from)
	require.Equal(t, 1, to)
	return driver
}
