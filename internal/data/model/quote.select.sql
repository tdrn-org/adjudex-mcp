SELECT
    b.listing,
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
    quote a, listing b
WHERE
    a.listing_id = b.id AND
    a.listing_id = $1
ORDER BY
    a.timestamp DESC