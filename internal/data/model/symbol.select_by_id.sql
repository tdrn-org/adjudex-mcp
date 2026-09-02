SELECT
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
    a.id = $1