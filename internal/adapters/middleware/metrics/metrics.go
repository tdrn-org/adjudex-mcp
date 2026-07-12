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

type Recorder struct {
	enabled           bool
	quotePriceCurrent *prometheus.GaugeVec
}

func NewRecorder(registry *prometheus.Registry) *Recorder {
	recorder := &Recorder{
		enabled: registry != nil,
	}
	if recorder.enabled {
		return recorder
	}
	factory := promauto.With(registry)
	recorder.quotePriceCurrent = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: SubsystemQuote,
		Name:      "price_current",
	}, []string{"symbol", "currency", "source"})
	return recorder
}

func (r *Recorder) RecordQuote(quote *domain.Quote) {
	if !r.enabled {
		return
	}

	price, err := r.quotePriceCurrent.GetMetricWithLabelValues(quote.Symbol, quote.Currency, quote.Source)
	if err == nil {
		price.Set(quote.Price)
	} else {
		slog.Warn("price metric failure", slog.Any("err", err))
	}
}
