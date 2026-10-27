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
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/go-finance"
)

func (s *Server) queryAndSaveQuotes(ctx context.Context, last, now time.Time) error {
	s.logger.Info("querying and saving quotes...")

	listings, err := s.dataStore.ListPositionListings(ctx)
	if err != nil {
		s.logger.Error("failed list listings for quote querying", slog.Any("err", err))
		return err
	}
	for _, listing := range listings {
		quote, err := s.marketDataProvider.QueryQuote(ctx, listing)
		if errors.Is(err, finance.ErrRequestPending) {
			s.logger.Error("skipping pending quote", slog.Any("listing", listing.ID()))
			continue
		} else if err != nil {
			s.logger.Error("failed query quote", slog.Any("listing", listing.ID()), slog.Any("err", err))
			continue
		}
		err = s.dataStore.SaveQuote(ctx, quote)
		if err != nil {
			s.logger.Error("failed to save quote", slog.Any("listing", listing.ID()), slog.Any("err", err))
			continue
		}
	}
	return nil
}

func (s *Server) evalAlerts(ctx context.Context, last, now time.Time) error {
	s.logger.Info("evaluating alerts...")

	alerts, err := s.dataStore.ListArmedAlerts(ctx)
	if err != nil {
		s.logger.Error("failed to list armed alerts", slog.Any("err", err))
		return err
	}

	for _, alert := range alerts {
		quote, err := s.dataStore.GetLatestQuote(ctx, alert.Listing)
		if err != nil {
			s.logger.Warn("failed to get latest quote for alert", slog.String("listing", alert.Listing.ID()), slog.Any("err", err))
			continue
		} else if quote == nil {
			continue
		}
		if alert.Evaluate(quote.Price) {
			s.logger.Info("alert triggered!", slog.String("alert", alert.Name), slog.String("listing", alert.Listing.ID()), "price", quote.Price)
			now := time.Now()
			alert.Fire(now, fmt.Sprintf("%s: %.2f", alert.Condition, quote.Price))
			err := s.dataStore.UpdateAlert(ctx, &alert)
			if err != nil {
				s.logger.Error("failed to update triggered alert", slog.String("alert", alert.Name), slog.Any("err", err))
			}
		}
	}
	return nil
}

func (s *Server) collectMetrics(ctx context.Context, last, now time.Time) error {
	s.logger.Info("collecting metrics...")

	listings, err := s.dataStore.ListPositionListings(ctx)
	if err != nil {
		s.logger.Error("failed list listings for metric collection", slog.Any("err", err))
		return err
	}
	quoteMap := make(map[string]*domain.Quote, len(listings))
	for _, listing := range listings {
		quote, err := s.dataStore.GetLatestQuote(ctx, listing)
		if err != nil {
			s.logger.Error("failed get latest quote for metric collection", slog.String("listing", listing.ID()), slog.Any("err", err))
			return err
		} else if quote == nil {
			continue
		}
		quoteMap[quote.Listing.ID()] = quote
		s.metricsRecorder.RecordQuote(quote)
	}
	portfolios, err := s.dataStore.ListPortfolios(ctx)
	if err != nil {
		s.logger.Error("failed list portfolios for metric collection", slog.Any("err", err))
		return err
	}
	for _, portfolio := range portfolios {
		holdings := make(domain.Holdings, 0, len(portfolio.Positions))
		bases := make([]finance.Currency, 0, 2*len(portfolio.Positions))
		for _, position := range portfolio.Positions {
			positionQuote := quoteMap[position.Listing.ID()]
			if positionQuote != nil {
				holdings = append(holdings, domain.NewHolding(position, positionQuote.ListingPrice))
				bases = append(bases, position.Currency, positionQuote.Currency)
			} else {
				s.logger.Error("missing listing quote in metric collection", slog.Any("listing", position.Listing))
			}
		}
		target := s.cfg.Display.Currency
		rates, err := s.marketDataProvider.QueryExchangeRates(ctx, target, bases)
		if err != nil {
			s.logger.Warn("failed to query exchange rates for metric collection", slog.String("portfolio", portfolio.ID), slog.Any("err", err))
			continue
		}
		summary, err := holdings.Summarize(target, rates)
		if err != nil {
			s.logger.Warn("failed to summarize portfolio for metric collection", slog.String("portfolio", portfolio.ID), slog.Any("err", err))
			continue
		}
		s.metricsRecorder.RecordPortfolio(&portfolio, *summary)
	}
	return nil
}
