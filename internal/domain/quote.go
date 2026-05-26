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

import "time"

// Quote represents a single price snapshot for a security.
type Quote struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    int64
	Source    string // provider identifier (e.g., "consorsbank", "yahoo", "mock")
}

// PriceHistory is a chronological series of quotes for a single symbol.
type PriceHistory struct {
	Symbol string
	Quotes []Quote
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
	Symbol    string
	Type      IndicatorType
	Period    int
	Value     float64
	Timestamp time.Time
}
