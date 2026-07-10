--
-- Quote
--
CREATE TABLE quote(
    id TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    open REAL NOT NULL,
    high REAL NOT NULL,
    low REAL NOT NULL,
    close REAL NOT NULL,
    volume INTEGER NOT NULL,
    source TEXT NOT NULL,
    PRIMARY KEY(id, timestamp)
);
--
-- Quote Symbol (maps symbols to latest quotes for tracking)
--
CREATE TABLE quote_symbol(
    symbol TEXT NOT NULL,
    quote_id TEXT NOT NULL,
    PRIMARY KEY(symbol),
    FOREIGN KEY(quote_id) REFERENCES quote(id)
);
--
-- Quote Symbol Source (which providers track which symbols)
--
CREATE TABLE quote_symbol_source(
    symbol TEXT NOT NULL,
    source TEXT NOT NULL,
    PRIMARY KEY(symbol, source),
    FOREIGN KEY(symbol) REFERENCES quote_symbol(symbol)
);
--
-- Portfolio
--
CREATE TABLE portfolio(
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(id)
);
--
-- Position
--
CREATE TABLE position(
    id TEXT NOT NULL,
    portfolio_id TEXT NOT NULL,
    symbol TEXT NOT NULL,
    quantity REAL NOT NULL,
    entry_price REAL NOT NULL,
    entry_date INTEGER NOT NULL,
    notes TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(id),
    FOREIGN KEY(portfolio_id) REFERENCES portfolio(id)
);
CREATE INDEX idx_position_portfolio_id ON position(portfolio_id);
CREATE INDEX idx_position_symbol ON position(symbol);
--
-- Alert
--
CREATE TABLE alert(
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    condition TEXT NOT NULL,
    threshold REAL NOT NULL,
    indicator_type TEXT,
    indicator_period INTEGER,
    state TEXT NOT NULL,
    triggered_at INTEGER,
    message TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(id)
);
CREATE INDEX idx_alert_symbol ON alert(symbol);
CREATE INDEX idx_alert_state ON alert(state);
--
-- Strategy
--
CREATE TABLE strategy(
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    rsi_period INTEGER NOT NULL,
    rsi_threshold REAL NOT NULL,
    sma_period INTEGER NOT NULL,
    sma_trigger REAL NOT NULL,
    max_position REAL NOT NULL,
    stop_loss REAL NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(id)
);
--
-- Trade
--
CREATE TABLE trade(
    id TEXT NOT NULL,
    strategy_id TEXT NOT NULL,
    symbol TEXT NOT NULL,
    direction TEXT NOT NULL,
    quantity REAL NOT NULL,
    price REAL NOT NULL,
    executed_at INTEGER NOT NULL,
    status TEXT NOT NULL,
    pnl REAL NOT NULL,
    notes TEXT NOT NULL,
    PRIMARY KEY(id),
    FOREIGN KEY(strategy_id) REFERENCES strategy(id)
);
CREATE INDEX idx_trade_strategy_id ON trade(strategy_id);
CREATE INDEX idx_trade_symbol ON trade(symbol);
--
-- EOF
--