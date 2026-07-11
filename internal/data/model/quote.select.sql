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
    a.symbol = $1
ORDER BY
    a.timestamp DESC