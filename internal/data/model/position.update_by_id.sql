UPDATE
    position
SET
    symbol = $1,
    quantity = $2,
    entry_price = $3,
    entry_date = $4,
    notes = $5,
    updated_at = $6
WHERE
    id = $1