package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"time"
)

type TileRepository interface {
	GetMetadata(ctx context.Context) (map[string]string, error)
	GetTile(ctx context.Context, z, x, y int) ([]byte, string, bool, error)
}

type tileRepository struct {
	db *sql.DB
	c  cache.Cache
}

func NewTileRepository(db *sql.DB, c cache.Cache) TileRepository {
	return &tileRepository{
		db: db,
		c:  c,
	}
}

func (r *tileRepository) GetMetadata(ctx context.Context) (map[string]string, error) {
	cacheId := "tile:metadata"

	var cached map[string]string
	err := r.c.Get(ctx, cacheId, &cached)
	if err == nil {
		return cached, nil
	}

	rows, err := r.db.QueryContext(ctx, "SELECT name, value FROM metadata")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metadata := make(map[string]string)

	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		metadata[name] = value
	}

	_ = r.c.Set(ctx, cacheId, metadata, constants.NormalCacheDuration)

	return metadata, nil
}

func (r *tileRepository) GetTile(ctx context.Context, z, x, y int) ([]byte, string, bool, error) {
	if z < 0 || x < 0 || y < 0 {
		return nil, "", false, fmt.Errorf("invalid tile coordinates")
	}

	// cache key
	cacheId := fmt.Sprintf("tile:%d:%d:%d", z, x, y)

	var cached []byte
	err := r.c.Get(ctx, cacheId, &cached)
	if err == nil {
		meta, _ := r.GetMetadata(ctx)
		return cached, meta["format"], meta["format"] == "pbf", nil
	}

	// XYZ -> TMS
	tmsY := (1 << z) - 1 - y

	var tileData []byte

	err = r.db.QueryRowContext(ctx, `
        SELECT tile_data
        FROM tiles
        WHERE zoom_level = ?
          AND tile_column = ?
          AND tile_row = ?
    `, z, x, tmsY).Scan(&tileData)

	if err != nil {
		return nil, "", false, err
	}

	meta, err := r.GetMetadata(ctx)
	if err != nil {
		return nil, "", false, err
	}

	_ = r.c.Set(ctx, cacheId, tileData, 5*time.Minute)

	return tileData, meta["format"], meta["format"] == "pbf", nil
}
