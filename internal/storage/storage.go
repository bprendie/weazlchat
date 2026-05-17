package storage

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db       *sql.DB
	key      []byte
	unlocked bool
}

func Open(path string) (*Store, error) {
	if err := mkdirFor(path); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
