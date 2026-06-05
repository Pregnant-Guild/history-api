package mbtiles

import (
	"database/sql"
	"fmt"
	"history-api/pkg/config"
	"runtime"

	_ "github.com/glebarez/go-sqlite"
)

func NewMBTilesDB(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?mode=ro&_journal_mode=off&_synchronous=off", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	maxOpenConns := config.GetIntConfigWithDefault("MBTILES_MAX_OPEN_CONNS", runtime.NumCPU()*4)
	if maxOpenConns < 1 {
		maxOpenConns = 1
	}
	maxIdleConns := config.GetIntConfigWithDefault("MBTILES_MAX_IDLE_CONNS", maxOpenConns/2)
	if maxIdleConns < 1 {
		maxIdleConns = 1
	}
	if maxIdleConns > maxOpenConns {
		maxIdleConns = maxOpenConns
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)

	return db, nil
}
