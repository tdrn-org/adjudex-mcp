UPDATE
    alert
SET
    name = $1,
    symbol = $2,
    condition = $3,
    threshold = $4,
    indicator_type = $5,
    indicator_period = $6,
    state = $7,
    triggered_at = $8,
    message = $9,
    updated_at = $10
WHERE
    id = $11