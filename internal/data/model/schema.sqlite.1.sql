--
-- Listing
--
CREATE TABLE listing(
    id TEXT NOT NULL,
    listing TEXT NOT NULL,
    provider TEXT NOT NULL,
    instrument_name TEXT NOT NULL,
    instrument_type TEXT NOT NULL,
    PRIMARY KEY(id)
);
CREATE TABLE listing_attribute(
    listing_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY(listing_id,key,key),
    FOREIGN KEY(listing_id) REFERENCES listing(id)
);
--
-- Quote
--
CREATE TABLE quote(
    listing_id TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    currency TEXT NOT NULL,
    open REAL NOT NULL,
    high REAL NOT NULL,
    low REAL NOT NULL,
    close REAL NOT NULL,
    price REAL NOT NULL,
    volume INTEGER NOT NULL,
    source TEXT NOT NULL,
    source_timestamp INTEGER NOT NULL,
    PRIMARY KEY(listing_id, timestamp),
    FOREIGN KEY(listing_id) REFERENCES listing(id)
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
    listing_id TEXT NOT NULL,
    currency TEXT NOT NULL,
    quantity REAL NOT NULL,
    entry_price REAL NOT NULL,
    entry_date INTEGER NOT NULL,
    notes TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(id),
    FOREIGN KEY(portfolio_id) REFERENCES portfolio(id),
    FOREIGN KEY(listing_id) REFERENCES listing(id)
);
CREATE INDEX idx_position_portfolio_id ON position(portfolio_id);
CREATE INDEX idx_position_listing_id ON position(listing_id);
--
-- Alert
--
CREATE TABLE alert(
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    listing_id TEXT NOT NULL,
    currency TEXT NOT NULL,
    condition TEXT NOT NULL,
    threshold REAL NOT NULL,
    indicator_type TEXT,
    indicator_period INTEGER,
    state TEXT NOT NULL,
    triggered_at INTEGER,
    message TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(id),
    FOREIGN KEY(listing_id) REFERENCES listing(id)
);
CREATE INDEX idx_alert_listing_id ON alert(listing_id);
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
    listing_id TEXT NOT NULL,
    currency TEXT NOT NULL,
    direction TEXT NOT NULL,
    quantity REAL NOT NULL,
    price REAL NOT NULL,
    executed_at INTEGER NOT NULL,
    status TEXT NOT NULL,
    pnl REAL NOT NULL,
    notes TEXT NOT NULL,
    PRIMARY KEY(id),
    FOREIGN KEY(strategy_id) REFERENCES strategy(id),
    FOREIGN KEY(listing_id) REFERENCES listing(id)
);
CREATE INDEX idx_trade_strategy_id ON trade(strategy_id);
CREATE INDEX idx_trade_listing_id ON trade(listing_id);
--
-- EOF
--