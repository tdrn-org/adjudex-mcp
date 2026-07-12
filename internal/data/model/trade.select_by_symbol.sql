SELECT
    a.id,
    a.strategy_id,
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
    a.symbol = $1