SELECT
    a.id,
    a.exchange,
    a.ticker,
    a.isin,
    a.wkn,
    a.figi,
    a.name,
    a.type
FROM
    symbol a
WHERE
    (a.exchange <> '' AND a.exchange = $1 AND a.ticker <> '' AND a.ticker = $2) OR
    (a.isin <> '' AND a.isin = $3) OR
    (a.wkn <> '' AND a.wkn = $4) OR
    (a.figi <> '' AND a.figi = $5)