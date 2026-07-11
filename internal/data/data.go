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
	"log/slog"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/data/model"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/go-database"
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

func (s *Store) CreatePortfolio(ctx context.Context, p *domain.Portfolio) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	storeP, err := model.InsertPortfolio(txCtx, s.driver, p)
	if err != nil {
		return err
	}
	storePoss := make([]*model.Position, 0, len(p.Positions))
	for _, pos := range p.Positions {
		storePos, err := model.InsertPosition(txCtx, s.driver, storeP.ID, &pos)
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

	storeP, err := model.SelectPortfolioByID(txCtx, s.driver, id)
	if err != nil {
		return nil, err
	}
	storePoss, err := model.SelectPositionsByPortfolioID(txCtx, s.driver, storeP.ID)
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

	storePs, err := model.SelectPortfolios(txCtx, s.driver)
	if err != nil {
		return nil, err
	}
	ps := make([]domain.Portfolio, 0, len(storePs))
	for _, storeP := range storePs {
		storePoss, err := model.SelectPositionsByPortfolioID(txCtx, s.driver, storeP.ID)
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

	err = model.DeletePositionsByPortfolioID(txCtx, s.driver, id)
	if err != nil {
		return err
	}
	err = model.DeletePortfolioByID(txCtx, s.driver, id)
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

	_, err = model.InsertPosition(txCtx, s.driver, portfolioID, pos)
	if err != nil {
		return err
	}
	err = model.TouchPortfolioByID(txCtx, s.driver, portfolioID)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}
func (s *Store) RemovePosition(ctx context.Context, portfolioID string, positionID string) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = model.DeletePositionByID(txCtx, s.driver, positionID)
	if err != nil {
		return err
	}
	err = model.TouchPortfolioByID(txCtx, s.driver, portfolioID)
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

	_, err = model.UpdatePosition(txCtx, s.driver, portfolioID, pos)
	if err != nil {
		return err
	}
	err = model.TouchPortfolioByID(txCtx, s.driver, portfolioID)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}

func (s *Store) ListSymbols(ctx context.Context) ([]string, error) {
	return model.SelectPositionSymbols(ctx, s.driver)
}

func (s *Store) SaveQuote(ctx context.Context, q *domain.Quote) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = model.DeleteQuoteByPK(txCtx, s.driver, q.Symbol, q.Timestamp)
	if err != nil {
		return err
	}
	_, err = model.InsertQuote(txCtx, s.driver, q)
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
	storeQuotes, err := model.SelectQuotes(ctx, s.driver, symbol, from, to)
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
	storeQuote, err := model.SelectLatestQuote(ctx, s.driver, symbol)
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
