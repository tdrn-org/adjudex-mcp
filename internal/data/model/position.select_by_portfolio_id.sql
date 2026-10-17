SELECT
    a.id,
    b.listing,
    a.currency,
    a.quantity,
    a.entry_price,
    a.entry_date,
    a.notes,
    a.created_at,
    a.updated_at
FROM
    position a, listing b
WHERE
    a.listing_id = b.id AND
    a.portfolio_id = $1