SELECT DISTINCT
    a.symbol,
    (SELECT MAX(b.timestamp) FROM quote b WHERE b.symbol = a.symbol) timestamp
FROM
    position a
ORDER BY
    a.symbol ASC