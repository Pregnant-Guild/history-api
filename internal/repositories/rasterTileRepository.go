package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"strconv"
	"sync"
	"time"
)

const rasterTileCacheDuration = 5 * time.Minute

type RasterTileRepository interface {
	GetMetadata(ctx context.Context) (map[string]string, error)
	GetTile(ctx context.Context, z, x, y int) ([]byte, string, error)
}

type rasterTileRepository struct {
	db         *sql.DB
	c          cache.Cache
	metadataMu sync.RWMutex
	metadata   map[string]string
}

func NewRasterTileRepository(db *sql.DB, c cache.Cache) RasterTileRepository {
	return &rasterTileRepository{
		db: db,
		c:  c,
	}
}

func (r *rasterTileRepository) GetMetadata(ctx context.Context) (map[string]string, error) {
	r.metadataMu.RLock()
	if r.metadata != nil {
		metadata := r.metadata
		r.metadataMu.RUnlock()
		return metadata, nil
	}
	r.metadataMu.RUnlock()

	r.metadataMu.Lock()
	defer r.metadataMu.Unlock()

	if r.metadata != nil {
		return r.metadata, nil
	}

	cacheId := "rasterTile:metadata"

	var cached map[string]string
	err := r.c.Get(ctx, cacheId, &cached)
	if err == nil {
		r.metadata = cached
		return cached, nil
	}

	rows, err := r.db.QueryContext(ctx, "SELECT name, value FROM metadata")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metadata := make(map[string]string, 8)

	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		metadata[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	_ = r.c.Set(ctx, cacheId, metadata, constants.NormalCacheDuration)
	r.metadata = metadata

	return metadata, nil
}

func (r *rasterTileRepository) GetTile(ctx context.Context, z, x, y int) ([]byte, string, error) {
	if z < 0 || x < 0 || y < 0 {
		return nil, "", fmt.Errorf("invalid tile coordinates")
	}

	cacheId := "rasterTile:raw:" + strconv.Itoa(z) + ":" + strconv.Itoa(x) + ":" + strconv.Itoa(y)

	cached, err := r.c.GetRawClient().Get(ctx, cacheId).Bytes()
	if err == nil {
		meta, err := r.GetMetadata(ctx)
		if err != nil {
			return nil, "", err
		}
		return cached, meta["format"], nil
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
		return nil, "", err
	}

	meta, err := r.GetMetadata(ctx)
	if err != nil {
		return nil, "", err
	}

	_ = r.c.GetRawClient().Set(ctx, cacheId, tileData, rasterTileCacheDuration).Err()

	return tileData, meta["format"], nil
}
