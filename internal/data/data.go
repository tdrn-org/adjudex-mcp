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

func (s *Store) MergeSymbol(ctx context.Context, sym *finance.Symbol) (*finance.Symbol, error) {
	if sym.IsEmpty() {
		return nil, fmt.Errorf("cannot merge empty symbol")
	}

	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	storeSym, err := model.SelectSymbolBySymbol(txCtx, tx, sym)
	if err != nil {
		return nil, err
	}
	if storeSym != nil {
		storeSym.MergeSymbol(sym)
		err = storeSym.Update(txCtx, tx)
	} else {
		storeSym, err = model.InsertSymbol(txCtx, tx, sym)
	}
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return s.modelToSymbol(storeSym), nil
}

func (s *Store) modelToSymbol(storeSym *model.Symbol) *finance.Symbol {
	if storeSym == nil {
		return nil
	}
	return &finance.Symbol{
		Exchange: storeSym.Exchange,
		Ticker:   storeSym.Ticker,
		ISIN:     storeSym.ISIN,
		WKN:      storeSym.WKN,
		FIGI:     storeSym.FIGI,
		Name:     storeSym.Name,
		Type:     finance.SecurityType(storeSym.Type),
	}
}

func (s *Store) CreatePortfolio(ctx context.Context, p *domain.Portfolio) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	storeP, err := model.InsertPortfolio(txCtx, tx, p)
	if err != nil {
		return err
	}
	storePoss := make([]*model.Position, 0, len(p.Positions))
	for _, pos := range p.Positions {
		storePos, err := model.InsertPosition(txCtx, tx, storeP.ID, &pos)
		if err != nil {
			return err
		}
		storePoss = append(storePoss, storePos)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}

	p.ID, p.CreatedAt, p.UpdatedAt = storeP.ID, database.DB2Time(storeP.CreatedAt), database.DB2Time(storeP.UpdatedAt)
	for i, storePos := range storePoss {
		p.Positions[i].ID, p.Positions[i].CreatedAt, p.Positions[i].UpdatedAt = storePos.ID, database.DB2Time(storePos.CreatedAt), database.DB2Time(storePos.UpdatedAt)
	}
	return nil
}

func (s *Store) GetPortfolio(ctx context.Context, id string) (*domain.Portfolio, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	storeP, err := model.SelectPortfolioByID(txCtx, tx, id)
	if err != nil {
		return nil, err
	}
	storePoss, err := model.SelectPositionsByPortfolioID(txCtx, tx, storeP.ID)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return s.modelToPortfolio(storeP, storePoss), nil
}

func (s *Store) modelToPortfolio(storeP *model.Portfolio, storePoss []*model.Position) *domain.Portfolio {
	if storeP == nil {
		return nil
	}
	p := &domain.Portfolio{
		ID:          storeP.ID,
		Name:        storeP.Name,
		Description: storeP.Description,
		Positions:   make([]domain.Position, 0, len(storePoss)),
		CreatedAt:   database.DB2Time(storeP.CreatedAt),
		UpdatedAt:   database.DB2Time(storeP.UpdatedAt),
	}
	for _, storePos := range storePoss {
		pos := domain.Position{
			ID:         storePos.ID,
			Symbol:     storePos.Symbol,
			Currency:   storePos.Currency,
			Quantity:   storePos.Quantity,
			EntryPrice: storePos.EntryPrice,
			EntryDate:  database.DB2Time(storePos.EntryDate),
			Notes:      storePos.Notes,
			CreatedAt:  database.DB2Time(storePos.CreatedAt),
			UpdatedAt:  database.DB2Time(storePos.UpdatedAt),
		}
		p.Positions = append(p.Positions, pos)
	}
	return p
}

func (s *Store) ListPortfolios(ctx context.Context) ([]domain.Portfolio, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	storePs, err := model.SelectPortfolios(txCtx, tx)
	if err != nil {
		return nil, err
	}
	ps := make([]domain.Portfolio, 0, len(storePs))
	for _, storeP := range storePs {
		storePoss, err := model.SelectPositionsByPortfolioID(txCtx, tx, storeP.ID)
		if err != nil {
			return nil, err
		}
		ps = append(ps, *s.modelToPortfolio(storeP, storePoss))
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return ps, nil
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

func (s *Store) AddPosition(ctx context.Context, portfolioID string, pos *domain.Position) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	position, err := model.InsertPosition(txCtx, tx, portfolioID, pos)
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

	pos.ID, pos.CreatedAt, pos.UpdatedAt = position.ID, database.DB2Time(position.CreatedAt), database.DB2Time(position.UpdatedAt)
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

func (s *Store) UpdatePosition(ctx context.Context, portfolioID string, pos *domain.Position) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	position, err := model.UpdatePosition(txCtx, tx, portfolioID, pos)
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

	pos.UpdatedAt = database.DB2Time(position.UpdatedAt)
	return nil
}

func (s *Store) ListSymbols(ctx context.Context) (map[string]time.Time, error) {
	return model.SelectPositionSymbols(ctx, s.driver)
}

func (s *Store) SaveQuote(ctx context.Context, q *domain.Quote) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = model.DeleteQuoteByPK(txCtx, tx, q.Symbol, q.SourceTimestamp)
	if err != nil {
		return err
	}
	_, err = model.InsertQuote(txCtx, tx, q)
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

	for _, q := range quotes {
		err = s.SaveQuote(txCtx, &q)
		if err != nil {
			return err
		}
	}

	return tx.CommitTx(txCtx)
}

func (s *Store) GetQuotes(ctx context.Context, symbol string, from, to time.Time) ([]domain.Quote, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	storeQuotes, err := model.SelectQuotes(txCtx, tx, symbol, from, to)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	quotes := make([]domain.Quote, 0, len(storeQuotes))
	for _, storeQuote := range storeQuotes {
		quotes = append(quotes, *s.modelToQuote(storeQuote))
	}
	return quotes, nil
}

func (s *Store) GetLatestQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	storeQuote, err := model.SelectLatestQuote(txCtx, tx, symbol)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return s.modelToQuote(storeQuote), nil
}

func (s *Store) modelToQuote(storeQuote *model.Quote) *domain.Quote {
	if storeQuote == nil {
		return nil
	}
	return &domain.Quote{
		Symbol:          storeQuote.Symbol,
		Timestamp:       database.DB2Time(storeQuote.Timestamp),
		Currency:        storeQuote.Currency,
		Open:            storeQuote.Open,
		High:            storeQuote.High,
		Low:             storeQuote.Low,
		Close:           storeQuote.Close,
		Price:           storeQuote.Price,
		Volume:          storeQuote.Volume,
		Source:          storeQuote.Source,
		SourceTimestamp: database.DB2Time(storeQuote.SourceTimestamp),
	}
}

func (s *Store) CreateAlert(ctx context.Context, a *domain.Alert) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	alert, err := model.InsertAlert(txCtx, tx, a)
	if err != nil {
		return err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}

	a.ID, a.CreatedAt, a.UpdatedAt = alert.ID, database.DB2Time(alert.CreatedAt), database.DB2Time(alert.UpdatedAt)
	return nil
}

func (s *Store) GetAlert(ctx context.Context, id string) (*domain.Alert, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	storeAlert, err := model.SelectAlertByID(txCtx, tx, id)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return s.modelToAlert(storeAlert), nil
}

func (s *Store) modelToAlert(storeAlert *model.Alert) *domain.Alert {
	if storeAlert == nil {
		return nil
	}
	var indicator *domain.IndicatorSpec
	if storeAlert.IndicatorType != nil && storeAlert.IndicatorPeriod != nil {
		indicator = &domain.IndicatorSpec{
			Type:   domain.IndicatorType(*storeAlert.IndicatorType),
			Period: *storeAlert.IndicatorPeriod,
		}
	}
	var triggeredAt *time.Time
	if storeAlert.TriggeredAt != nil {
		time := database.DB2Time(*storeAlert.TriggeredAt)
		triggeredAt = &time
	}
	alert := &domain.Alert{
		ID:          storeAlert.ID,
		Name:        storeAlert.Name,
		Symbol:      storeAlert.Symbol,
		Currency:    storeAlert.Currency,
		Condition:   domain.AlertCondition(storeAlert.Condition),
		Threshold:   storeAlert.Threshold,
		Indicator:   indicator,
		State:       domain.AlertState(storeAlert.State),
		TriggeredAt: triggeredAt,
		Message:     storeAlert.Message,
		CreatedAt:   database.DB2Time(storeAlert.CreatedAt),
		UpdatedAt:   database.DB2Time(storeAlert.UpdatedAt),
	}
	return alert
}

func (s *Store) ListAlerts(ctx context.Context, symbol string) ([]domain.Alert, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	storeAlerts, err := model.SelectAlertsBySymbol(txCtx, tx, symbol)
	if err != nil {
		return nil, err
	}
	alerts := make([]domain.Alert, 0, len(storeAlerts))
	for _, storeAlert := range storeAlerts {
		alerts = append(alerts, *s.modelToAlert(storeAlert))
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

	storeAlerts, err := model.SelectAlertsByState(txCtx, tx, domain.AlertStateArmed)
	if err != nil {
		return nil, err
	}
	alerts := make([]domain.Alert, 0, len(storeAlerts))
	for _, storeAlert := range storeAlerts {
		alerts = append(alerts, *s.modelToAlert(storeAlert))
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

func (s *Store) UpdateAlert(ctx context.Context, a *domain.Alert) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	alert, err := model.UpdateAlert(txCtx, tx, a)
	if err != nil {
		return err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}

	a.UpdatedAt = database.DB2Time(alert.UpdatedAt)
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

func (s *Store) RecordTrade(ctx context.Context, t *domain.Trade) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	trade, err := model.InsertTrade(txCtx, tx, t)
	if err != nil {
		return err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}

	t.ID = trade.ID
	return nil
}

func (s *Store) GetTrade(ctx context.Context, id string) (*domain.Trade, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	trade, err := model.SelectTradeByID(txCtx, tx, id)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	return s.modelToTrade(trade), nil
}

func (s *Store) modelToTrade(storeTrade *model.Trade) *domain.Trade {
	if storeTrade == nil {
		return nil
	}
	trade := &domain.Trade{
		ID:         storeTrade.ID,
		StrategyID: storeTrade.StrategyID,
		Symbol:     storeTrade.Symbol,
		Currency:   storeTrade.Currency,
		Direction:  domain.TradeDirection(storeTrade.Direction),
		Quantity:   storeTrade.Quantity,
		Price:      storeTrade.Price,
		ExecutedAt: database.DB2Time(storeTrade.ExecutedAt),
		Status:     domain.TradeStatus(storeTrade.Status),
		PnL:        storeTrade.PnL,
		Notes:      storeTrade.Notes,
	}
	return trade
}

func (s *Store) ListTrades(ctx context.Context, symbol string) ([]domain.Trade, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	storeTrades, err := model.SelectTradesBySymbol(txCtx, tx, symbol)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	trades := make([]domain.Trade, 0, len(storeTrades))
	for _, storeTrade := range storeTrades {
		trades = append(trades, *s.modelToTrade(storeTrade))
	}
	return trades, nil
}

func (s *Store) ListTradesByStrategy(ctx context.Context, strategyID string) ([]domain.Trade, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	storeTrades, err := model.SelectTradesByStrategyID(txCtx, tx, strategyID)
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}

	trades := make([]domain.Trade, 0, len(storeTrades))
	for _, storeTrade := range storeTrades {
		trades = append(trades, *s.modelToTrade(storeTrade))
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
