SELECT
    a.id,
    a.strategy_id,
    b.listing,
    a.currency,
    a.direction,
    a.quantity,
    a.price,
    a.executed_at,
    a.status,
    a.pnl,
    a.notes
FROM
    trade a, listing b
WHERE
    a.listing_id = b.id AND
    a.listing_id = $1