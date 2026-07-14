SELECT
    a.id,
    a.symbol,
    a.currency,
    a.direction,
    a.quantity,
    a.price,
    a.executed_at,
    a.status,
    a.pnl,
    a.notes
FROM
    trade a
WHERE
    a.strategy_id = $1