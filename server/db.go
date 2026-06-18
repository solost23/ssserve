package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Token struct {
	ID         string
	Name       string
	Token      string
	Password   string
	ServerPort int
	QuotaGB    *float64
	UsedBytes  int64
	CreatedAt  string
	ExpiresAt  *string
	Active     bool
}

type DB struct {
	db *sql.DB
}

func initDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tokens (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			token       TEXT UNIQUE NOT NULL,
			password    TEXT NOT NULL,
			server_port INTEGER NOT NULL DEFAULT 0,
			quota_gb    REAL,
			used_bytes  INTEGER DEFAULT 0,
			created_at  TEXT DEFAULT (datetime('now')),
			expires_at  TEXT,
			active      INTEGER DEFAULT 1
		)
	`)
	if err != nil {
		return nil, err
	}
	// migrations for existing installs
	_, _ = db.Exec(`ALTER TABLE tokens ADD COLUMN server_port INTEGER NOT NULL DEFAULT 0`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) NextPort(basePort int) (int, error) {
	var maxPort int
	err := d.db.QueryRow(
		`SELECT COALESCE(MAX(server_port), ?) FROM tokens`,
		basePort-1,
	).Scan(&maxPort)
	if err != nil {
		return 0, err
	}
	return maxPort + 1, nil
}

func (d *DB) CreateToken(name, token, password string, serverPort int, expiryDays *int, quotaGB *float64) (*Token, error) {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	var expiresAt *string
	if expiryDays != nil {
		t := time.Now().AddDate(0, 0, *expiryDays).Format("2006-01-02 15:04:05")
		expiresAt = &t
	}
	_, err := d.db.Exec(
		`INSERT INTO tokens (id, name, token, password, server_port, quota_gb, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, name, token, password, serverPort, quotaGB, expiresAt,
	)
	if err != nil {
		return nil, err
	}
	return d.GetActiveToken(token)
}

func (d *DB) GetActiveToken(token string) (*Token, error) {
	row := d.db.QueryRow(`
		SELECT id, name, token, password, server_port, quota_gb, used_bytes, created_at, expires_at, active
		FROM tokens
		WHERE token = ?
		  AND active = 1
		  AND (expires_at IS NULL OR expires_at > datetime('now'))
	`, token)
	return scanToken(row)
}

func (d *DB) ListTokens() ([]Token, error) {
	rows, err := d.db.Query(`
		SELECT id, name, token, password, server_port, quota_gb, used_bytes, created_at, expires_at, active
		FROM tokens ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []Token
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, *t)
	}
	return tokens, rows.Err()
}

func (d *DB) RevokeToken(token string) error {
	_, err := d.db.Exec(`UPDATE tokens SET active = 0 WHERE token = ?`, token)
	return err
}

func (d *DB) ActiveTokens() ([]Token, error) {
	rows, err := d.db.Query(`
		SELECT id, name, token, password, server_port, quota_gb, used_bytes, created_at, expires_at, active
		FROM tokens
		WHERE active = 1 AND (expires_at IS NULL OR expires_at > datetime('now'))
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []Token
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, *t)
	}
	return tokens, rows.Err()
}

func (d *DB) UpdateQuota(token string, quotaGB *float64) error {
	_, err := d.db.Exec(`UPDATE tokens SET quota_gb = ? WHERE token = ?`, quotaGB, token)
	return err
}

func (d *DB) AddStats(increments map[int]int64) error {
	for port, delta := range increments {
		if delta <= 0 {
			continue
		}
		if _, err := d.db.Exec(
			`UPDATE tokens SET used_bytes = used_bytes + ? WHERE server_port = ? AND active = 1`,
			delta, port,
		); err != nil {
			return err
		}
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanToken(s scanner) (*Token, error) {
	var t Token
	var active int
	err := s.Scan(&t.ID, &t.Name, &t.Token, &t.Password, &t.ServerPort,
		&t.QuotaGB, &t.UsedBytes, &t.CreatedAt, &t.ExpiresAt, &active)
	if err != nil {
		return nil, err
	}
	t.Active = active == 1
	return &t, nil
}
