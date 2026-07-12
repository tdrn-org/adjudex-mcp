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

package adjudexmcp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

type jobFunc func(ctx context.Context)

func (s *Server) runJobs() {
	s.logger.Info("running jobs...")
	ctx := context.Background()
	for _, job := range s.jobs {
		job(ctx)
	}
}

func (s *Server) evalAlerts(ctx context.Context) {
	s.logger.Info("evaluating alerts...")

	alerts, err := s.dataStore.ListArmedAlerts(ctx)
	if err != nil {
		s.logger.Error("failed to list armed alerts", slog.Any("err", err))
		return
	}

	for _, alert := range alerts {
		quote, err := s.dataStore.GetLatestQuote(ctx, alert.Symbol)
		if err != nil {
			s.logger.Warn("failed to get latest quote for alert", slog.String("symbol", alert.Symbol), slog.Any("err", err))
			continue
		}
		if alert.Evaluate(quote.Price) {
			s.logger.Info("alert triggered!", slog.String("alert", alert.Name), slog.String("symbol", alert.Symbol), "price", quote.Price)
			now := time.Now()
			alert.Fire(now, fmt.Sprintf("%s: %.2f", alert.Condition, quote.Price))
			err := s.dataStore.UpdateAlert(ctx, &alert)
			if err != nil {
				s.logger.Error("failed to update triggered alert", slog.String("alert", alert.Name), slog.Any("err", err))
			}
		}
	}
}

func (s *Server) collectMetrics(ctx context.Context) {
	s.logger.Info("collecting metrics...")

	symbolMap, err := s.dataStore.ListSymbols(ctx)
	if err != nil {
		s.logger.Error("failed list symbols for metric collection", slog.Any("err", err))
		return
	}
	symbolQuotes := make(map[string]*domain.Quote, len(symbolMap))
	for symbol := range symbolMap {
		quote, err := s.dataStore.GetLatestQuote(ctx, symbol)
		if err != nil {
			s.logger.Error("failed get latest quote for metric collection", slog.String("symbol", symbol), slog.Any("err", err))
			return
		}
		symbolQuotes[symbol] = quote
		s.metricsRecorder.RecordQuote(quote)
	}
	portfolios, err := s.dataStore.ListPortfolios(ctx)
	if err != nil {
		s.logger.Error("failed list portfolios for metric collection", slog.Any("err", err))
		return
	}
	for _, portfolio := range portfolios {
		holdings := make(domain.Holdings, 0, len(portfolio.Positions))
		for _, position := range portfolio.Positions {
			positionQuote := symbolQuotes[position.Symbol]
			if positionQuote != nil {
				holding := domain.NewHolding(position, positionQuote.Price)
				holdings = append(holdings, holding)
			} else {
				s.logger.Error("missing symbol quote in metric collection", slog.String("symbol", position.Symbol))
			}
		}
		s.metricsRecorder.RecordPortfolio(&portfolio, holdings.Summarize())
	}
}
