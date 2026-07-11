SELECT
    a.name,
    a.description,
    a.created_at,
    a.updated_at
FROM
    portfolio a
WHERE
    a.id = $1