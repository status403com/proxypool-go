package sqlitestore

import (
	"database/sql"
	"fmt"

	proxypool "github.com/status403com/proxypool-go"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func New(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlitestore: open: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS proxies (
		id TEXT PRIMARY KEY,
		host TEXT NOT NULL,
		port TEXT NOT NULL,
		username TEXT DEFAULT '',
		password TEXT DEFAULT '',
		protocol TEXT DEFAULT 'http'
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlitestore: migrate: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Load() ([]proxypool.Proxy, error) {
	rows, err := s.db.Query("SELECT id, host, port, username, password, protocol FROM proxies")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []proxypool.Proxy
	for rows.Next() {
		var p proxypool.Proxy
		var proto string
		if err := rows.Scan(&p.ID, &p.Host, &p.Port, &p.Username, &p.Password, &proto); err != nil {
			return nil, err
		}
		p.Protocol = proxypool.Protocol(proto)
		proxies = append(proxies, p)
	}
	return proxies, rows.Err()
}

func (s *SQLiteStore) Save(proxies []proxypool.Proxy) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM proxies"); err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO proxies (id, host, port, username, password, protocol) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range proxies {
		if _, err := stmt.Exec(p.ID, p.Host, p.Port, p.Username, p.Password, string(p.Protocol)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
