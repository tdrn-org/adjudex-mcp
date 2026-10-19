DELETE FROM
    quote
WHERE
    listing_id = $1
    AND timestamp = $2