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

package store

import (
	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-database/sqlite"
)

// Config returns a SQLite database configuration with the adjudex schema.
func Config(dbPath string) database.Config {
	return sqlite.NewConfig(dbPath, sqlite.ModeRWC)
}

// Schema returns the adjudex database schema as SQL scripts.
func Schema() [][]byte {
	return [][]byte{
		[]byte(schemaV1),
	}
}

const schemaV1 = `
CREATE TABLE IF NOT EXISTS portfolios (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS positions (
    id           TEXT PRIMARY KEY,
    portfolio_id TEXT NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    symbol       TEXT NOT NULL,
    quantity     REAL NOT NULL,
    entry_price  REAL NOT NULL,
    entry_date   INTEGER NOT NULL,
    notes        TEXT NOT NULL DEFAULT '',
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_positions_portfolio ON positions(portfolio_id);

CREATE TABLE IF NOT EXISTS quotes (
    symbol    TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    open      REAL NOT NULL,
    high      REAL NOT NULL,
    low       REAL NOT NULL,
    close     REAL NOT NULL,
    volume    INTEGER NOT NULL,
    source    TEXT NOT NULL,
    PRIMARY KEY (symbol, timestamp)
);

CREATE INDEX IF NOT EXISTS idx_quotes_symbol ON quotes(symbol);
CREATE INDEX IF NOT EXISTS idx_quotes_timestamp ON quotes(timestamp);

CREATE TABLE IF NOT EXISTS alerts (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    symbol          TEXT NOT NULL,
    condition       TEXT NOT NULL,
    threshold       REAL NOT NULL,
    indicator_type  TEXT NOT NULL DEFAULT '',
    indicator_period TEXT NOT NULL DEFAULT '0',
    state           TEXT NOT NULL DEFAULT 'armed',
    triggered_at    INTEGER,
    message         TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_alerts_symbol ON alerts(symbol);
CREATE INDEX IF NOT EXISTS idx_alerts_state ON alerts(state);

CREATE TABLE IF NOT EXISTS trades (
    id          TEXT PRIMARY KEY,
    strategy_id TEXT NOT NULL,
    symbol      TEXT NOT NULL,
    direction   TEXT NOT NULL,
    quantity    REAL NOT NULL,
    price       REAL NOT NULL,
    executed_at INTEGER NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    pnl         REAL NOT NULL DEFAULT 0,
    notes       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol);
CREATE INDEX IF NOT EXISTS idx_trades_strategy ON trades(strategy_id);

CREATE TABLE IF NOT EXISTS strategies (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    rsi_period    INTEGER NOT NULL DEFAULT 14,
    rsi_threshold REAL NOT NULL DEFAULT 30.0,
    sma_period    INTEGER NOT NULL DEFAULT 20,
    sma_trigger   REAL NOT NULL DEFAULT 5.0,
    max_position  REAL NOT NULL DEFAULT 1000.0,
    stop_loss     REAL NOT NULL DEFAULT 5.0,
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL
);
`
