SELECT
    a.id,
    a.symbol,
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
    alert a
WHERE
    a.state = $1