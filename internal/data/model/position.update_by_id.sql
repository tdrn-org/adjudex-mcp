UPDATE
    position
SET
    symbol = $1,
    currency = $2,
    quantity = $3,
    entry_price = $4,
    entry_date = $5,
    notes = $6,
    updated_at = $7
WHERE
    id = $8