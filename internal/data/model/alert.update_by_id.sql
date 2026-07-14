UPDATE
    alert
SET
    name = $1,
    symbol = $2,
    currency = $3,
    condition = $4,
    threshold = $5,
    indicator_type = $6,
    indicator_period = $7,
    state = $8,
    triggered_at = $9,
    message = $10,
    updated_at = $11
WHERE
    id = $12