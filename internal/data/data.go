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

type QuoteTrackerConfig map[string][]string

func (s *Store) GetQuoteTrackingConfig(ctx context.Context) (QuoteTrackerConfig, error) {
	return nil, nil
}

func (s *Store) SetQuoteTrackingConfig(ctx context.Context, config QuoteTrackerConfig) error {
	return nil
}

func (s *Store) ListQuoteSymbols() ([][]string, error) {
	return nil, nil
}

func (s *Store) RegisterQuoteSymbol(symbol, alias string) error {
	return nil
}

func (s *Store) RecordQuote(symbol string, quote *domain.Quote) error {
	return nil
}
