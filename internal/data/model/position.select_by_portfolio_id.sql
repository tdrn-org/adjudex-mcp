SELECT
    a.id,
    a.symbol,
    a.currency,
    a.quantity,
    a.entry_price,
    a.entry_date,
    a.notes,
    a.created_at,
    a.updated_at
FROM
    position a
WHERE
    a.portfolio_id = $1