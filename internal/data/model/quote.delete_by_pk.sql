DELETE FROM
    quote
WHERE
    symbol = $1
    AND timestamp = $2