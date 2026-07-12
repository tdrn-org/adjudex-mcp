SELECT
    a.name,
    a.description,
    a.rsi_period,
    a.rsi_threshold,
    a.sma_period,
    a.sma_trigger,
    a.max_position,
    a.stop_loss,
    a.created_at,
    a.updated_at
FROM
    strategy a
WHERE
    a.id = $1