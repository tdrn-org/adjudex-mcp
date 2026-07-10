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

package twelvedata

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

const parseErrorFormat string = "failed to parse '%s' value '%s' for symbol '%s' (cause: %w)"

const longDateTimeLayout string = "2006-01-02 15:04:05"
const shortDateTimeLayout string = "2006-01-02"

func (p *twelvedataProvider) parseSymbolTimestamp(symbol, field string, value string) (time.Time, error) {
	timestamp, err := time.Parse(longDateTimeLayout, value)
	if err != nil {
		timestamp, err = time.Parse(shortDateTimeLayout, value)
		if err != nil {
			return time.Time{}, fmt.Errorf(parseErrorFormat, field, value, symbol, err)
		}
	}
	return timestamp, nil
}

func (p *twelvedataProvider) parseSymbolCurrency(symbol, field, value string) (float64, error) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return math.NaN(), fmt.Errorf(parseErrorFormat, field, value, symbol, err)
	}
	return parsed, nil
}

func (p *twelvedataProvider) parseSymbolAmount(symbol, field, value string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf(parseErrorFormat, field, value, symbol, err)
	}
	return parsed, nil
}
