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

package metrics

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

const Namespace string = "adjudex"
const SubsystemQuote string = "quote"
const SubsystemPortfolio string = "portfolio"

type Recorder struct {
	enabled              bool
	quotePrice           *prometheus.GaugeVec
	quotePriceHigh       *prometheus.GaugeVec
	quotePriceLow        *prometheus.GaugeVec
	quoteVolumeShares    *prometheus.GaugeVec
	portfolioValueAmount *prometheus.GaugeVec
	portfolioPnLAmount   *prometheus.GaugeVec
	portfolioPnLRatio    *prometheus.GaugeVec
}

func NewRecorder(registry *prometheus.Registry) *Recorder {
	recorder := &Recorder{
		enabled: registry != nil,
	}
	if !recorder.enabled {
		return recorder
	}
	factory := promauto.With(registry)
	quoteLabels := []string{"symbol", "currency", "source"}
	recorder.quotePrice = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: SubsystemQuote,
		Name:      "price",
	}, quoteLabels)
	recorder.quotePriceHigh = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: SubsystemQuote,
		Name:      "price_high",
	}, quoteLabels)
	recorder.quotePriceLow = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: SubsystemQuote,
		Name:      "price_low",
	}, quoteLabels)
	recorder.quoteVolumeShares = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: SubsystemQuote,
		Name:      "volume_shares",
	}, quoteLabels)
	portfolioLabels := []string{"name", "currency"}
	recorder.portfolioValueAmount = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: SubsystemPortfolio,
		Name:      "value_amount",
	}, portfolioLabels)
	recorder.portfolioPnLAmount = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: SubsystemPortfolio,
		Name:      "pnl_amount",
	}, portfolioLabels)
	recorder.portfolioPnLRatio = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: SubsystemPortfolio,
		Name:      "pnl_ratio",
	}, portfolioLabels)
	return recorder
}

func (r *Recorder) RecordQuote(quote *domain.Quote) {
	if !r.enabled {
		return
	}

	quoteLabels := []string{quote.Symbol, quote.Currency, quote.Source}
	r.setGauge("quote price", r.quotePrice, quote.Price, quoteLabels...)
	r.setGauge("quote price high", r.quotePriceHigh, quote.High, quoteLabels...)
	r.setGauge("quote price low", r.quotePriceHigh, quote.Low, quoteLabels...)
	r.setGauge("quote volume", r.quoteVolumeShares, float64(quote.Volume), quoteLabels...)
}

func (r *Recorder) RecordPortfolio(portfolio *domain.Portfolio, holdingsSummary domain.HoldingsSummary) {
	if !r.enabled {
		return
	}

	// TODO: Use actual currency
	portfolioLabels := []string{portfolio.Name, "EUR"}
	r.setGauge("portfolio market value", r.portfolioValueAmount, holdingsSummary.MarketValue, portfolioLabels...)
	r.setGauge("portfolio PnL amount", r.portfolioPnLAmount, holdingsSummary.PnL, portfolioLabels...)
	r.setGauge("portfolio PnL ratio", r.portfolioPnLRatio, holdingsSummary.PnLPercent, portfolioLabels...)
}

func (r *Recorder) setGauge(name string, gaugeVec *prometheus.GaugeVec, value float64, labels ...string) {
	gauge, err := gaugeVec.GetMetricWithLabelValues(labels...)
	if err == nil {
		gauge.Set(value)
	} else {
		slog.Warn("collect '"+name+"' metric failure", slog.Any("err", err))
	}
}
