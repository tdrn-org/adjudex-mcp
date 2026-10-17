SELECT
    a.id,
    a.listing,
    a.provider,
    a.instrument_name,
    a.instrument_type
FROM
    listing a
WHERE
    a.id IN (SELECT DISTINCT b.listing_id FROM position b)