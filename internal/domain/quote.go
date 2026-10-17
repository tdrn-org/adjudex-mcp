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
	"time"

	"github.com/tdrn-org/go-finance"
)

type Quote finance.Quote

func (q *Quote) ApplyCurrency(currency finance.Currency, exchangeRate float64) {
	q.Currency = currency
	q.Open *= exchangeRate
	q.High *= exchangeRate
	q.Low *= exchangeRate
	q.Close *= exchangeRate
	q.Price *= exchangeRate
}

type Quotes []Quote

func (qs Quotes) ApplyCurrency(currency finance.Currency, exchangeRate float64) {
	for i := range qs {
		qs[i].ApplyCurrency(currency, exchangeRate)
	}
}

func (qs Quotes) Currency() finance.Currency {
	if len(qs) == 0 {
		return ""
	}
	return qs[0].Currency
}

// PriceHistory is a chronological series of quotes for a single symbol.
type PriceHistory struct {
	Listing finance.Listing `json:"listing"`
	Quotes  []Quote         `json:"quotes"`
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
	Listing   finance.Listing  `json:"listing"`
	Currency  finance.Currency `json:"currency"`
	Type      IndicatorType    `json:"type"`
	Period    int              `json:"period"`
	Value     float64          `json:"value"`
	Timestamp time.Time        `json:"timestamp"`
}
