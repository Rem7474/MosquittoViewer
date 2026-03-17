//go:build sqlite

package store

import (
	"database/sql"
	"time"

	"github.com/example/mosquitto-viewer/internal/logwatcher"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY,
			timestamp TEXT NOT NULL,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			client_id TEXT,
			topic TEXT,
			plugin TEXT,
			raw TEXT NOT NULL
		)
	`); err != nil {
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Save(entry logwatcher.LogEntry) error {
	_, err := s.db.Exec(`INSERT INTO logs(id,timestamp,level,message,client_id,topic,plugin,raw) VALUES(?,?,?,?,?,?,?,?)`,
		entry.ID,
		entry.Timestamp.Format(time.RFC3339Nano),
		entry.Level,
		entry.Message,
		entry.ClientID,
		entry.Topic,
		entry.Plugin,
		entry.Raw,
	)
	return err
}
