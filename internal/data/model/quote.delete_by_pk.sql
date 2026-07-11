DELETE FROM
    quote
WHERE
    symbol = $1
    AND source_timestamp = $2