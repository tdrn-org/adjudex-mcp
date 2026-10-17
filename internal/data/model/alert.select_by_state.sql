SELECT
    a.id,
    b.listing,
    a.currency,
    a.name,
    a.condition,
    a.threshold,
    a.indicator_type,
    a.indicator_period,
    a.triggered_at,
    a.message,
    a.created_at,
    a.updated_at
FROM
    alert a, listing b
WHERE
    a.listing_id = b.id AND
    a.state = $1