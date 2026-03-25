package mbtiles

import (
    "database/sql"
    "fmt"

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

    db.SetMaxOpenConns(10)
    db.SetMaxIdleConns(5)

    return db, nil
}