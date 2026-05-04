-- name: CreateGeometry :one
INSERT INTO geometries (
    id, geo_type, draw_geometry, binding, time_start, time_end, bbox, project_id
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, uuidv7()), $1, $2, $3, $4, $5, ST_MakeEnvelope(sqlc.arg('min_lng')::float8, sqlc.arg('min_lat')::float8, sqlc.arg('max_lng')::float8, sqlc.arg('max_lat')::float8, 4326), $6
)
RETURNING id, geo_type, draw_geometry, binding, time_start, time_end, project_id,
    ST_XMin(bbox)::float8 as min_lng, ST_YMin(bbox)::float8 as min_lat, ST_XMax(bbox)::float8 as max_lng, ST_YMax(bbox)::float8 as max_lat,
    is_deleted, created_at, updated_at;

-- name: GetGeometryById :one
SELECT id, geo_type, draw_geometry, binding, time_start, time_end, project_id,
    ST_XMin(bbox)::float8 as min_lng, ST_YMin(bbox)::float8 as min_lat, ST_XMax(bbox)::float8 as max_lng, ST_YMax(bbox)::float8 as max_lat,
    is_deleted, created_at, updated_at
FROM geometries
WHERE id = $1 AND is_deleted = false;

-- name: UpdateGeometry :one
UPDATE geometries
SET 
    geo_type = COALESCE(sqlc.narg('geo_type'), geo_type),
    draw_geometry = COALESCE(sqlc.narg('draw_geometry'), draw_geometry),
    binding = COALESCE(sqlc.narg('binding'), binding),
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
RETURNING id, geo_type, draw_geometry, binding, time_start, time_end, project_id,
    ST_XMin(bbox)::float8 as min_lng, ST_YMin(bbox)::float8 as min_lat, ST_XMax(bbox)::float8 as max_lng, ST_YMax(bbox)::float8 as max_lat,
    is_deleted, created_at, updated_at;

-- name: DeleteGeometry :exec
UPDATE geometries
SET 
    is_deleted = true
WHERE id = $1;

-- name: SearchGeometries :many
SELECT 
    g.id, g.geo_type, g.draw_geometry, g.binding, g.time_start, g.time_end, g.project_id,
    ST_XMin(g.bbox)::float8 as min_lng, 
    ST_YMin(g.bbox)::float8 as min_lat, 
    ST_XMax(g.bbox)::float8 as max_lng, 
    ST_YMax(g.bbox)::float8 as max_lat,
    g.is_deleted, g.created_at, g.updated_at
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
    int4range(g.time_start, g.time_end, '[]') @> sqlc.narg('time_point')::int
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
ORDER BY g.id DESC;

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
    id, geo_type, draw_geometry, binding, time_start, time_end, project_id,
    ST_XMin(bbox)::float8 as min_lng, 
    ST_YMin(bbox)::float8 as min_lat, 
    ST_XMax(bbox)::float8 as max_lng, 
    ST_YMax(bbox)::float8 as max_lat,
    is_deleted, created_at, updated_at
FROM geometries 
WHERE id = ANY($1::uuid[]) AND is_deleted = false;

-- name: GetGeometriesByProjectId :many
SELECT 
    id, geo_type, draw_geometry, binding, time_start, time_end, project_id,
    ST_XMin(bbox)::float8 as min_lng, 
    ST_YMin(bbox)::float8 as min_lat, 
    ST_XMax(bbox)::float8 as max_lng, 
    ST_YMax(bbox)::float8 as max_lat,
    is_deleted, created_at, updated_at
FROM geometries
WHERE project_id = $1 AND is_deleted = false;

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
