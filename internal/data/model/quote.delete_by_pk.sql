DELETE FROM
    quote
WHERE
    listing_id = $1
    AND source_timestamp = $2