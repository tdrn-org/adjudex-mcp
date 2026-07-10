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
    PRIMARY KEY(id,timestamp)
);
CREATE TABLE quote_symbol(
    symbol TEXT NOT NULL,
    quote_id TEXT NOT NULL,
    PRIMARY KEY(symbol),
    FOREIGN KEY(quote_id) REFERENCES quote(id)    
);
CREATE TABLE quote_symbol_source(
    symbol TEXT NOT NULL,
    source TEXT NOT NULL,
    PRIMARY KEY(symbol,source),
    FOREIGN KEY(symbol) REFERENCES quote_symbol(symbol)    
);
--
-- EOF
--
