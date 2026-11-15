package main

import (
	"database/sql"
	"fmt"
)

func NewSqliteDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)

	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db %w", err)
	}

	return db, db.Ping()
}
