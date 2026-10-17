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

package data

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/data/model"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-finance"
)

type Store struct {
	driver *database.Driver
	logger *slog.Logger
}

func NewStore(name string, driver *database.Driver) *Store {
	return &Store{
		driver: driver,
		logger: slog.With(slog.String("name", name)),
	}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.driver.Ping(ctx)
}

func (s *Store) Close() error {
	return s.driver.Close()
}

// ErrUnknownListing indicates a listing whose ID the server did not issue, and
// which therefore cannot be resolved.
var ErrUnknownListing error = errors.New("unknown listing")

// AddListing registers a listing, so that a client can refer to it by its ID.
//
// This is the only way a listing enters the store on behalf of a client: the
// client receives an ID from listing_search and hands it back later. Only complete
// listings are accepted - an incomplete one would be stored under an ID that
// identifies nothing, and it could never be resolved again.
func (s *Store) AddListing(ctx context.Context, listing finance.Listing) error {
	if _, err := listing.Validate(); err != nil {
		return fmt.Errorf("registering listing: %w", err)
	}
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = ensureListing(txCtx, tx, listing)
	if err != nil {
		return err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}

	return nil
}

// ensureListing makes sure the row a listing-referencing table points to exists.
//
// Position, alert and trade carry a foreign key to listing, so every insert has to
// create the listing on first sight. Without this the very first row for a new
// instrument failed with "FOREIGN KEY constraint failed" — the instrument had never
// been written because a listing is only stored when something references it.
func ensureListing(ctx context.Context, tx *database.Tx, listing finance.Listing) error {
	existing, err := model.SelectListing(ctx, tx, listing.ID())
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	_, err = model.InsertListing(ctx, tx, listing)
	return err
}

// ResolveListing returns the stored form of the given listing.
//
// A listing that arrives from a client carries the ID only - instrument name and
// type are not part of it. The store is the only trustworthy source for them, and
// resolving against it also proves that the server issued this listing in the
// first place: a caller cannot introduce an instrument by assembling an ID.
func (s *Store) ResolveListing(ctx context.Context, listing finance.Listing) (finance.Listing, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return listing, err
	}
	defer tx.RollbackUncommitedTx(txCtx)
	return resolveListing(txCtx, tx, listing)
}

// resolveListing is ResolveListing within an existing transaction.
func resolveListing(ctx context.Context, tx *database.Tx, listing finance.Listing) (finance.Listing, error) {
	existing, err := model.SelectListing(ctx, tx, listing.ID())
	if err != nil {
		return listing, err
	}
	if existing == nil {
		return listing, fmt.Errorf("%w: %q: obtain the listing via listing_search", ErrUnknownListing, listing.ID())
	}
	// Deliberately the stored listing and not the caller's one: whatever the
	// caller says about the instrument is replaced by what was registered.
	return existing.Listing, nil
}

func (s *Store) CreatePortfolio(ctx context.Context, portfolio *domain.Portfolio) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	p, err := model.InsertPortfolio(txCtx, tx, portfolio)
	if err != nil {
		return err
	}
	poss := make([]*model.Position, 0, len(portfolio.Positions))
	for i := range portfolio.Positions {
		resolved, err := resolveListing(txCtx, tx, portfolio.Positions[i].Listing)
		if err != nil {
			return err
		}
		portfolio.Positions[i].Listing = resolved
		pos, err := model.InsertPosition(txCtx, tx, p.ID, &portfolio.Positions[i])
		if err != nil {
			return err
		}
		poss = append(poss, pos)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}

	portfolio.ID, portfolio.CreatedAt, portfolio.UpdatedAt = p.ID, database.DB2Time(p.CreatedAt), database.DB2Time(p.UpdatedAt)
	for i, storePos := range poss {
		portfolio.Positions[i].ID, portfolio.Positions[i].CreatedAt, portfolio.Positions[i].UpdatedAt = storePos.ID, database.DB2Time(storePos.CreatedAt), database.DB2Time(storePos.UpdatedAt)
	}
	return nil
}

func (s *Store) GetPortfolio(ctx context.Context, id string) (*domain.Portfolio, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	p, err := model.SelectPortfolioByID(txCtx, tx, id)
	if err != nil {
		return nil, err
	}
	poss, err := model.SelectPositionsByPortfolioID(txCtx, tx, p.ID)
	if err != nil {
		return nil, err
	}
	positions := make([]domain.Position, 0, len(poss))
	for _, pos := range poss {
		positions = append(positions, *s.modelToPosition(pos))
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return s.modelToPortfolio(p, positions), nil
}

func (s *Store) modelToPosition(pos *model.Position) *domain.Position {
	return &domain.Position{
		ID:         pos.ID,
		Listing:    pos.Listing,
		Currency:   pos.Currency,
		Quantity:   pos.Quantity,
		EntryPrice: pos.EntryPrice,
		EntryDate:  database.DB2Time(pos.EntryDate),
		Notes:      pos.Notes,
		CreatedAt:  database.DB2Time(pos.CreatedAt),
		UpdatedAt:  database.DB2Time(pos.UpdatedAt),
	}
}

func (s *Store) modelToPortfolio(p *model.Portfolio, positions []domain.Position) *domain.Portfolio {
	if p == nil {
		return nil
	}
	return &domain.Portfolio{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Positions:   positions,
		CreatedAt:   database.DB2Time(p.CreatedAt),
		UpdatedAt:   database.DB2Time(p.UpdatedAt),
	}
}

func (s *Store) ListPortfolios(ctx context.Context) ([]domain.Portfolio, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	ps, err := model.SelectPortfolios(txCtx, tx)
	if err != nil {
		return nil, err
	}
	portfolios := make([]domain.Portfolio, 0, len(ps))
	for _, p := range ps {
		poss, err := model.SelectPositionsByPortfolioID(txCtx, tx, p.ID)
		if err != nil {
			return nil, err
		}
		positions := make([]domain.Position, 0, len(poss))
		for _, pos := range poss {
			positions = append(positions, *s.modelToPosition(pos))
		}
		portfolios = append(portfolios, *s.modelToPortfolio(p, positions))
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return portfolios, nil
}

func (s *Store) DeletePortfolio(ctx context.Context, id string) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = model.DeletePositionsByPortfolioID(txCtx, tx, id)
	if err != nil {
		return err
	}
	err = model.DeletePortfolioByID(txCtx, tx, id)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}

func (s *Store) AddPosition(ctx context.Context, portfolioID string, position *domain.Position) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	resolved, err := resolveListing(txCtx, tx, position.Listing)
	if err != nil {
		return err
	}
	position.Listing = resolved

	pos, err := model.InsertPosition(txCtx, tx, portfolioID, position)
	if err != nil {
		return err
	}
	err = model.TouchPortfolioByID(txCtx, tx, portfolioID)
	if err != nil {
		return err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}

	position.ID, position.CreatedAt, position.UpdatedAt = pos.ID, database.DB2Time(pos.CreatedAt), database.DB2Time(pos.UpdatedAt)
	return nil
}

func (s *Store) RemovePosition(ctx context.Context, portfolioID string, positionID string) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = model.DeletePositionByID(txCtx, tx, positionID)
	if err != nil {
		return err
	}
	err = model.TouchPortfolioByID(txCtx, tx, portfolioID)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}

func (s *Store) UpdatePosition(ctx context.Context, portfolioID string, position *domain.Position) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	pos, err := model.UpdatePosition(txCtx, tx, portfolioID, position)
	if err != nil {
		return err
	}
	err = model.TouchPortfolioByID(txCtx, tx, portfolioID)
	if err != nil {
		return err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}

	position.UpdatedAt = database.DB2Time(pos.UpdatedAt)
	return nil
}

func (s *Store) ListPositionListings(ctx context.Context) ([]finance.Listing, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	ls, err := model.SelectListingsByPositions(txCtx, tx)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	listings := make([]finance.Listing, 0, len(ls))
	for _, l := range ls {
		listings = append(listings, l.Listing)
	}
	return listings, nil
}

func (s *Store) SaveQuote(ctx context.Context, quote *domain.Quote) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = model.DeleteQuoteByPK(txCtx, tx, quote.Listing, quote.SourceTimestamp)
	if err != nil {
		return err
	}
	_, err = model.InsertQuote(txCtx, tx, quote)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}

func (s *Store) SaveQuotes(ctx context.Context, quotes []domain.Quote) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	for _, quote := range quotes {
		err = s.SaveQuote(txCtx, &quote)
		if err != nil {
			return err
		}
	}

	return tx.CommitTx(txCtx)
}

func (s *Store) GetQuotes(ctx context.Context, listing finance.Listing, from, to time.Time) ([]domain.Quote, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	qs, err := model.SelectQuotes(txCtx, tx, listing, from, to)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	quotes := make([]domain.Quote, 0, len(qs))
	for _, q := range qs {
		quotes = append(quotes, *s.modelToQuote(q, listing))
	}
	return quotes, nil
}

func (s *Store) GetLatestQuote(ctx context.Context, listing finance.Listing) (*domain.Quote, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	q, err := model.SelectLatestQuote(txCtx, tx, listing)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return s.modelToQuote(q, listing), nil
}

func (s *Store) modelToQuote(q *model.Quote, listing finance.Listing) *domain.Quote {
	if q == nil {
		return nil
	}
	return &domain.Quote{
		Listing:         listing,
		Timestamp:       database.DB2Time(q.Timestamp),
		Currency:        q.Currency,
		Open:            q.Open,
		High:            q.High,
		Low:             q.Low,
		Close:           q.Close,
		Price:           q.Price,
		Volume:          q.Volume,
		Source:          q.Source,
		SourceTimestamp: database.DB2Time(q.SourceTimestamp),
	}
}

func (s *Store) CreateAlert(ctx context.Context, alert *domain.Alert) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	resolved, err := resolveListing(txCtx, tx, alert.Listing)
	if err != nil {
		return err
	}
	alert.Listing = resolved

	a, err := model.InsertAlert(txCtx, tx, alert)
	if err != nil {
		return err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}

	alert.ID, alert.CreatedAt, alert.UpdatedAt = a.ID, database.DB2Time(a.CreatedAt), database.DB2Time(a.UpdatedAt)
	return nil
}

func (s *Store) GetAlert(ctx context.Context, id string) (*domain.Alert, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	a, err := model.SelectAlertByID(txCtx, tx, id)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return s.modelToAlert(a), nil
}

func (s *Store) modelToAlert(a *model.Alert) *domain.Alert {
	if a == nil {
		return nil
	}
	var indicator *domain.IndicatorSpec
	if a.IndicatorType != nil && a.IndicatorPeriod != nil {
		indicator = &domain.IndicatorSpec{
			Type:   domain.IndicatorType(*a.IndicatorType),
			Period: *a.IndicatorPeriod,
		}
	}
	var triggeredAt *time.Time
	if a.TriggeredAt != nil {
		time := database.DB2Time(*a.TriggeredAt)
		triggeredAt = &time
	}
	alert := &domain.Alert{
		ID:          a.ID,
		Name:        a.Name,
		Listing:     a.Listing,
		Currency:    a.Currency,
		Condition:   domain.AlertCondition(a.Condition),
		Threshold:   a.Threshold,
		Indicator:   indicator,
		State:       domain.AlertState(a.State),
		TriggeredAt: triggeredAt,
		Message:     a.Message,
		CreatedAt:   database.DB2Time(a.CreatedAt),
		UpdatedAt:   database.DB2Time(a.UpdatedAt),
	}
	return alert
}

func (s *Store) ListAlerts(ctx context.Context, listing finance.Listing) ([]domain.Alert, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	as, err := model.SelectAlertsByListing(txCtx, tx, listing)
	if err != nil {
		return nil, err
	}
	alerts := make([]domain.Alert, 0, len(as))
	for _, a := range as {
		alerts = append(alerts, *s.modelToAlert(a))
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

func (s *Store) ListArmedAlerts(ctx context.Context) ([]domain.Alert, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	as, err := model.SelectAlertsByState(txCtx, tx, domain.AlertStateArmed)
	if err != nil {
		return nil, err
	}
	alerts := make([]domain.Alert, 0, len(as))
	for _, a := range as {
		alerts = append(alerts, *s.modelToAlert(a))
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

func (s *Store) UpdateAlert(ctx context.Context, alert *domain.Alert) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	a, err := model.UpdateAlert(txCtx, tx, alert)
	if err != nil {
		return err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}

	alert.UpdatedAt = database.DB2Time(a.UpdatedAt)
	return nil
}

func (s *Store) DeleteAlert(ctx context.Context, id string) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = model.DeleteAlertByID(txCtx, tx, id)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}

func (s *Store) RecordTrade(ctx context.Context, trade *domain.Trade) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	resolved, err := resolveListing(txCtx, tx, trade.Listing)
	if err != nil {
		return err
	}
	trade.Listing = resolved

	t, err := model.InsertTrade(txCtx, tx, trade)
	if err != nil {
		return err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}

	trade.ID = t.ID
	return nil
}

func (s *Store) GetTrade(ctx context.Context, id string) (*domain.Trade, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	t, err := model.SelectTradeByID(txCtx, tx, id)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return s.modelToTrade(t), nil
}

func (s *Store) modelToTrade(t *model.Trade) *domain.Trade {
	if t == nil {
		return nil
	}
	trade := &domain.Trade{
		ID:         t.ID,
		StrategyID: t.StrategyID,
		Listing:    t.Listing,
		Currency:   t.Currency,
		Direction:  domain.TradeDirection(t.Direction),
		Quantity:   t.Quantity,
		Price:      t.Price,
		ExecutedAt: database.DB2Time(t.ExecutedAt),
		Status:     domain.TradeStatus(t.Status),
		PnL:        t.PnL,
		Notes:      t.Notes,
	}
	return trade
}

func (s *Store) ListTrades(ctx context.Context, listing finance.Listing) ([]domain.Trade, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	ts, err := model.SelectTradesByListing(txCtx, tx, listing)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	trades := make([]domain.Trade, 0, len(ts))
	for _, t := range ts {
		trades = append(trades, *s.modelToTrade(t))
	}
	return trades, nil
}

func (s *Store) ListTradesByStrategy(ctx context.Context, strategyID string) ([]domain.Trade, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	ts, err := model.SelectTradesByStrategyID(txCtx, tx, strategyID)
	if err != nil {
		return nil, err
	}
	trades := make([]domain.Trade, 0, len(ts))
	for _, t := range ts {
		trades = append(trades, *s.modelToTrade(t))
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return trades, nil
}

func (s *Store) SaveStrategy(ctx context.Context, st *domain.Strategy) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	strategy, err := model.InsertStrategy(txCtx, tx, st)
	if err != nil {
		return err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}

	st.ID, st.CreatedAt, st.UpdatedAt = strategy.ID, database.DB2Time(strategy.CreatedAt), database.DB2Time(strategy.UpdatedAt)
	return nil
}

func (s *Store) GetStrategy(ctx context.Context, id string) (*domain.Strategy, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	strategy, err := model.SelectStrategyByID(txCtx, tx, id)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return s.modelToStrategy(strategy), nil
}

func (s *Store) modelToStrategy(storeStrategy *model.Strategy) *domain.Strategy {
	if storeStrategy == nil {
		return nil
	}
	strategy := &domain.Strategy{
		ID:          storeStrategy.ID,
		Name:        storeStrategy.Name,
		Description: storeStrategy.Description,
		Parameters: domain.StrategyParams{
			RSIPeriod:    storeStrategy.RSIPeriod,
			RSIThreshold: storeStrategy.RSIThreshold,
			SMAPeriod:    storeStrategy.SMAPeriod,
			SMATrigger:   storeStrategy.SMATrigger,
			MaxPosition:  storeStrategy.MaxPosition,
			StopLoss:     storeStrategy.StopLoss,
		},
		CreatedAt: database.DB2Time(storeStrategy.CreatedAt),
		UpdatedAt: database.DB2Time(storeStrategy.UpdatedAt),
	}
	return strategy
}

func (s *Store) ListStrategies(ctx context.Context) ([]domain.Strategy, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	storeStrategies, err := model.SelectStrategies(txCtx, tx)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	strategies := make([]domain.Strategy, 0, len(storeStrategies))
	for _, storeStrategy := range storeStrategies {
		strategies = append(strategies, *s.modelToStrategy(storeStrategy))
	}
	return strategies, nil
}

func (s *Store) DeleteStrategy(ctx context.Context, id string) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = model.DeleteStrategyByID(txCtx, tx, id)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}
