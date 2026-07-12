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

package domain

import (
	"errors"
	"time"
)

var ErrNoQuote error = errors.New("no quote")
var ErrQuoteRateLimit error = errors.New("quote rate limit")

type SymbolInfo struct {
	Symbol string
	Source string
	Info   string
}

// Quote represents a single price snapshot for a symbol.
type Quote struct {
	Symbol          string    `json:"symbol"`
	Timestamp       time.Time `json:"timestamp"`
	Currency        string    `json:"currency"`
	Open            float64   `json:"open"`
	High            float64   `json:"high"`
	Low             float64   `json:"low"`
	Close           float64   `json:"close"`
	Price           float64   `json:"price"`
	Volume          int64     `json:"volume"`
	Source          string    `json:"source"`
	SourceTimestamp time.Time `json:"source_timestamp"`
}

func (q *Quote) ApplyCurrency(currency string, exchangeRate float64) {
	q.Currency = currency
	q.Open *= exchangeRate
	q.High *= exchangeRate
	q.Low *= exchangeRate
	q.Close *= exchangeRate
	q.Price *= exchangeRate
}

type Quotes []Quote

func (qs Quotes) ApplyCurrency(currency string, exchangeRate float64) {
	for i := range qs {
		qs[i].ApplyCurrency(currency, exchangeRate)
	}
}

func (qs Quotes) Currency() string {
	if len(qs) == 0 {
		return ""
	}
	return qs[0].Currency
}

// PriceHistory is a chronological series of quotes for a single symbol.
type PriceHistory struct {
	Symbol string  `json:"symbol"`
	Quotes []Quote `json:"quotes"`
}

// IndicatorType names the technical indicator.
type IndicatorType string

const (
	IndicatorRSI  IndicatorType = "rsi"
	IndicatorSMA  IndicatorType = "sma"
	IndicatorEMA  IndicatorType = "ema"
	IndicatorMACD IndicatorType = "macd"
)

// IndicatorValue holds a computed indicator value at a point in time.
type IndicatorValue struct {
	Symbol    string        `json:"symbol"`
	Type      IndicatorType `json:"type"`
	Period    int           `json:"period"`
	Value     float64       `json:"value"`
	Timestamp time.Time     `json:"timestamp"`
}
