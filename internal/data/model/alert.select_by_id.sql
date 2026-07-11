SELECT
    a.name,
    a.symbol,
    a.condition,
    a.threshold,
    a.indicator_type,
    a.indicator_period,
    a.state,
    a.triggered_at,
    a.message,
    a.created_at,
    a.updated_at
FROM
    alert a
WHERE
    a.id = $1