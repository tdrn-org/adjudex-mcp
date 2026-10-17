SELECT
    a.listing,
    a.provider,
    a.instrument_name,
    a.instrument_type
FROM
    listing a
WHERE
    a.id = $1