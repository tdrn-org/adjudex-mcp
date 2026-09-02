UPDATE
    symbol
SET
    exchange = $1,
    ticker = $2,
    isin = $3,
    wkn = $4,
    figi = $5,
    name = $6
WHERE
    id = $7