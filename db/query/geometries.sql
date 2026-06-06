-- name: CreateGeometry :one
INSERT INTO geometries (
    id, geo_type, draw_geometry, bound_with, time_start, time_end, bbox, project_id
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, uuidv7()), $1, $2, $3, $4, $5, ST_MakeEnvelope(sqlc.arg('min_lng')::float8, sqlc.arg('min_lat')::float8, sqlc.arg('max_lng')::float8, sqlc.arg('max_lat')::float8, 4326), $6
)
RETURNING id, geo_type, draw_geometry, bound_with, time_start, time_end, project_id,
    ST_XMin(bbox)::float8 as min_lng, ST_YMin(bbox)::float8 as min_lat, ST_XMax(bbox)::float8 as max_lng, ST_YMax(bbox)::float8 as max_lat,
    is_deleted, created_at, updated_at;

-- name: GetGeometryById :one
SELECT geometries.id, geometries.geo_type, geometries.draw_geometry, geometries.bound_with, geometries.time_start, geometries.time_end, geometries.project_id,
    ST_XMin(geometries.bbox)::float8 as min_lng, ST_YMin(geometries.bbox)::float8 as min_lat, ST_XMax(geometries.bbox)::float8 as max_lng, ST_YMax(geometries.bbox)::float8 as max_lat,
    geometries.is_deleted, geometries.created_at, geometries.updated_at,
    COALESCE(
        (SELECT ARRAY_AGG(br.id) FROM battle_replays br
         WHERE br.geometry_id = geometries.id
           AND br.is_deleted = false),
        '{}'::uuid[]
    )::uuid[] AS replay_ids
FROM geometries
WHERE geometries.id = $1 AND geometries.is_deleted = false;

-- name: UpdateGeometry :one
UPDATE geometries
SET 
    geo_type = COALESCE(sqlc.narg('geo_type'), geo_type),
    draw_geometry = COALESCE(sqlc.narg('draw_geometry'), draw_geometry),
    bound_with = CASE 
        WHEN sqlc.narg('update_bound_with')::boolean = true THEN 
            sqlc.narg('bound_with')
        ELSE bound_with 
    END,
    time_start = COALESCE(sqlc.narg('time_start'), time_start),
    time_end = COALESCE(sqlc.narg('time_end'), time_end),
    project_id = COALESCE(sqlc.narg('project_id'), project_id),
    bbox = CASE 
        WHEN sqlc.narg('update_bbox')::boolean = true THEN 
            ST_MakeEnvelope(sqlc.narg('min_lng')::float8, sqlc.narg('min_lat')::float8, sqlc.narg('max_lng')::float8, sqlc.narg('max_lat')::float8, 4326)
        ELSE bbox 
    END,
    updated_at = now()
WHERE id = sqlc.arg('id') AND is_deleted = false
RETURNING id, geo_type, draw_geometry, bound_with, time_start, time_end, project_id,
    ST_XMin(bbox)::float8 as min_lng, ST_YMin(bbox)::float8 as min_lat, ST_XMax(bbox)::float8 as max_lng, ST_YMax(bbox)::float8 as max_lat,
    is_deleted, created_at, updated_at;

-- name: DeleteGeometry :exec
UPDATE geometries
SET 
    is_deleted = true
WHERE id = $1;

-- name: SearchGeometries :many
SELECT 
    g.id, g.geo_type, g.draw_geometry, g.bound_with, g.time_start, g.time_end, g.project_id,
    ST_XMin(g.bbox)::float8 as min_lng, 
    ST_YMin(g.bbox)::float8 as min_lat, 
    ST_XMax(g.bbox)::float8 as max_lng, 
    ST_YMax(g.bbox)::float8 as max_lat,
    g.is_deleted, g.created_at, g.updated_at,
    COALESCE(
        (SELECT ARRAY_AGG(br.id) FROM battle_replays br
         WHERE br.geometry_id = g.id
           AND br.is_deleted = false),
        '{}'::uuid[]
    )::uuid[] AS replay_ids
FROM geometries g
WHERE g.is_deleted = false
  AND (sqlc.narg('project_id')::uuid IS NULL OR g.project_id = sqlc.narg('project_id')::uuid)
  AND (
    sqlc.narg('search_min_lng')::float8 IS NULL OR
    sqlc.narg('search_min_lat')::float8 IS NULL OR
    sqlc.narg('search_max_lng')::float8 IS NULL OR
    sqlc.narg('search_max_lat')::float8 IS NULL OR
    g.bbox && ST_MakeEnvelope(
        sqlc.narg('search_min_lng')::float8, 
        sqlc.narg('search_min_lat')::float8, 
        sqlc.narg('search_max_lng')::float8, 
        sqlc.narg('search_max_lat')::float8, 
        4326
    )
  )
  AND (
    sqlc.narg('time_point')::int IS NULL OR
    int4range(g.time_start, g.time_end, '[]') && int4range(
        sqlc.narg('time_point')::int - COALESCE(sqlc.narg('time_range')::int, 0),
        sqlc.narg('time_point')::int + COALESCE(sqlc.narg('time_range')::int, 0),
        '[]'
    )
  )
  AND (
    sqlc.narg('entity_id')::uuid IS NULL OR
    EXISTS (
        SELECT 1 
        FROM entity_geometries eg 
        WHERE eg.geometry_id = g.id 
          AND eg.entity_id = sqlc.narg('entity_id')::uuid
    )
  )
  AND (
    sqlc.narg('has_bound')::boolean IS NULL OR
    sqlc.narg('has_bound')::boolean = true OR
    g.bound_with IS NULL
  )
ORDER BY g.id DESC
LIMIT NULLIF(sqlc.arg('limit_count')::int, 0);

-- name: SearchGeometriesByEntityName :many
WITH matched_entities AS (
    SELECT
        e.id,
        e.name,
        e.description
    FROM entities e
    WHERE e.is_deleted = false
      AND (sqlc.narg('name')::text IS NULL OR e.name ILIKE '%' || sqlc.narg('name')::text || '%')
      AND (sqlc.narg('cursor_id')::uuid IS NULL OR e.id < sqlc.narg('cursor_id')::uuid)
    ORDER BY e.id DESC
    LIMIT sqlc.arg('limit_count')
)
SELECT
    me.id AS entity_id,
    me.name AS entity_name,
    me.description AS entity_description,
    g.id AS geometry_id,
    g.geo_type,
    g.draw_geometry,
    g.bound_with,
    g.time_start,
    g.time_end,
    COALESCE(
        (SELECT ARRAY_AGG(br.id) FROM battle_replays br
         WHERE br.geometry_id = g.id
           AND br.is_deleted = false),
        '{}'::uuid[]
    )::uuid[] AS replay_ids
FROM matched_entities me
LEFT JOIN entity_geometries eg
    ON eg.entity_id = me.id
LEFT JOIN geometries g
    ON g.id = eg.geometry_id
   AND g.is_deleted = false
ORDER BY me.id DESC, g.id DESC;

-- name: BulkDeleteEntityGeometriesByEntityId :many
DELETE FROM entity_geometries
WHERE entity_id = $1
RETURNING geometry_id;

-- name: CreateEntityGeometries :exec
INSERT INTO entity_geometries (
    entity_id, geometry_id, project_id
)
SELECT $1, unnest(@geometry_ids::uuid[]), $2
ON CONFLICT DO NOTHING;

-- name: DeleteEntityGeometriesByProjectID :exec
DELETE FROM entity_geometries
WHERE project_id = $1;

-- name: GetGeometriesByIDs :many
SELECT 
    geometries.id, geometries.geo_type, geometries.draw_geometry, geometries.bound_with, geometries.time_start, geometries.time_end, geometries.project_id,
    ST_XMin(geometries.bbox)::float8 as min_lng, 
    ST_YMin(geometries.bbox)::float8 as min_lat, 
    ST_XMax(geometries.bbox)::float8 as max_lng, 
    ST_YMax(geometries.bbox)::float8 as max_lat,
    geometries.is_deleted, geometries.created_at, geometries.updated_at,
    COALESCE(
        (SELECT ARRAY_AGG(br.id) FROM battle_replays br
         WHERE br.geometry_id = geometries.id
           AND br.is_deleted = false),
        '{}'::uuid[]
    )::uuid[] AS replay_ids
FROM geometries 
WHERE geometries.id = ANY($1::uuid[]) AND geometries.is_deleted = false;

-- name: GetGeometriesByProjectId :many
SELECT 
    geometries.id, geometries.geo_type, geometries.draw_geometry, geometries.bound_with, geometries.time_start, geometries.time_end, geometries.project_id,
    ST_XMin(geometries.bbox)::float8 as min_lng, 
    ST_YMin(geometries.bbox)::float8 as min_lat, 
    ST_XMax(geometries.bbox)::float8 as max_lng, 
    ST_YMax(geometries.bbox)::float8 as max_lat,
    geometries.is_deleted, geometries.created_at, geometries.updated_at,
    COALESCE(
        (SELECT ARRAY_AGG(br.id) FROM battle_replays br
         WHERE br.geometry_id = geometries.id
           AND br.is_deleted = false),
        '{}'::uuid[]
    )::uuid[] AS replay_ids
FROM geometries
WHERE geometries.project_id = $1 AND geometries.is_deleted = false;

-- name: DeleteGeometriesByIDs :exec
UPDATE geometries
SET is_deleted = true
WHERE id = ANY($1::uuid[]);

-- name: BulkDeleteEntityGeometriesByGeometryID :exec
DELETE FROM entity_geometries
WHERE geometry_id = $1;

-- name: DeleteEntityGeometry :exec
DELETE FROM entity_geometries
WHERE entity_id = $1 AND geometry_id = $2;

-- name: GetEntityGeometriesByPairs :many
SELECT
    e.id AS entity_id,
    e.name AS entity_name,
    e.description AS entity_description,
    g.id AS geometry_id,
    g.geo_type,
    g.draw_geometry,
    g.bound_with,
    g.time_start,
    g.time_end,
    COALESCE(
        (SELECT ARRAY_AGG(br.id) FROM battle_replays br
         WHERE br.geometry_id = g.id
           AND br.is_deleted = false),
        '{}'::uuid[]
    )::uuid[] AS replay_ids
FROM (
    SELECT unnest(@entity_ids::uuid[]) as eid, unnest(@geometry_ids::uuid[]) as gid
) as pairs
JOIN entities e ON e.id = pairs.eid
JOIN entity_geometries eg ON eg.entity_id = e.id AND eg.geometry_id = pairs.gid
JOIN geometries g ON g.id = pairs.gid
WHERE e.is_deleted = false
AND g.is_deleted = false;

-- name: GetGeometriesByBoundWith :many
SELECT 
    geometries.id, geometries.geo_type, geometries.draw_geometry, geometries.bound_with, geometries.time_start, geometries.time_end, geometries.project_id,
    ST_XMin(geometries.bbox)::float8 as min_lng, 
    ST_YMin(geometries.bbox)::float8 as min_lat, 
    ST_XMax(geometries.bbox)::float8 as max_lng, 
    ST_YMax(geometries.bbox)::float8 as max_lat,
    geometries.is_deleted, geometries.created_at, geometries.updated_at,
    COALESCE(
        (SELECT ARRAY_AGG(br.id) FROM battle_replays br
         WHERE br.geometry_id = geometries.id
           AND br.is_deleted = false),
        '{}'::uuid[]
    )::uuid[] AS replay_ids
FROM geometries
WHERE geometries.bound_with = $1 AND geometries.is_deleted = false;

-- name: GetGeometryIDsByEntityIDs :many
SELECT entity_id, geometry_id
FROM entity_geometries
WHERE entity_id = ANY($1::uuid[]);


