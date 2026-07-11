SELECT
    a.timestamp,
    a.currency,
    a.open,
    a.high,
    a.low,
    a.close,
    a.price,
    a.volume,
    a.source,
    a.source_timestamp
FROM
    quote a
WHERE
    a.symbol = $1 AND
    a.timestamp BETWEEN $2 AND $3
ORDER BY
    a.timestamp ASC