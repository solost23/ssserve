package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
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
	UpdatedAt  string
	ExpiresAt  *string
	Active     bool
	Suspended  bool
}

type Admin struct {
	Username  string `json:"username"`
	IsOwner   bool   `json:"is_owner"`
	CreatedAt string `json:"created_at"`
}

var ErrNotFound = errors.New("not found")

type DB struct {
	db *sql.DB
}

func initDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tokens (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			token       TEXT UNIQUE NOT NULL,
			password    TEXT NOT NULL,
			server_port INTEGER NOT NULL DEFAULT 0,
			quota_gb    REAL,
			used_bytes  INTEGER NOT NULL DEFAULT 0,
			suspended   INTEGER NOT NULL DEFAULT 0,
			created_at  TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
			deleted_at  TEXT,
			expires_at  TEXT,
			active      INTEGER NOT NULL DEFAULT 1
		)
	`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS admins (
			username      TEXT PRIMARY KEY,
			password_hash TEXT NOT NULL,
			is_owner      INTEGER NOT NULL DEFAULT 0,
			created_at    TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at    TEXT NOT NULL DEFAULT (datetime('now')),
			deleted_at    TEXT
		)
	`)
	if err != nil {
		return nil, err
	}
	for _, m := range []struct{ table, col, def string }{
		{"tokens", "server_port", "INTEGER NOT NULL DEFAULT 0"},
		{"tokens", "suspended", "INTEGER NOT NULL DEFAULT 0"},
		{"tokens", "updated_at", "TEXT NOT NULL DEFAULT ''"},
		{"tokens", "deleted_at", "TEXT"},
		{"admins", "updated_at", "TEXT NOT NULL DEFAULT ''"},
		{"admins", "deleted_at", "TEXT"},
	} {
		if err := addColumnIfMissing(db, m.table, m.col, m.def); err != nil {
			return nil, fmt.Errorf("migrate %s.%s: %w", m.table, m.col, err)
		}
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func addColumnIfMissing(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, colType string
		var dflt any
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	_, err = db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	return err
}

func now() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}

// --- admin methods ---

func (d *DB) AdminCount() (int, error) {
	var n int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM admins WHERE deleted_at IS NULL`).Scan(&n)
	return n, err
}

func (d *DB) CreateAdmin(username, password string, isOwner bool) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	owner := 0
	if isOwner {
		owner = 1
	}
	t := now()
	_, err = d.db.Exec(
		`INSERT INTO admins (username, password_hash, is_owner, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		username, string(hash), owner, t, t,
	)
	return err
}

func (d *DB) VerifyAdmin(username, password string) (*Admin, error) {
	var a Admin
	var hash string
	var isOwner int
	err := d.db.QueryRow(
		`SELECT username, password_hash, is_owner, created_at FROM admins WHERE username = ? AND deleted_at IS NULL`,
		username,
	).Scan(&a.Username, &hash, &isOwner, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, ErrNotFound
	}
	a.IsOwner = isOwner == 1
	return &a, nil
}

func (d *DB) GetAdmin(username string) (*Admin, error) {
	var a Admin
	var isOwner int
	err := d.db.QueryRow(
		`SELECT username, is_owner, created_at FROM admins WHERE username = ? AND deleted_at IS NULL`,
		username,
	).Scan(&a.Username, &isOwner, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.IsOwner = isOwner == 1
	return &a, nil
}

func (d *DB) ListAdmins() ([]Admin, error) {
	rows, err := d.db.Query(
		`SELECT username, is_owner, created_at FROM admins WHERE deleted_at IS NULL ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	admins := []Admin{}
	for rows.Next() {
		var a Admin
		var isOwner int
		if err := rows.Scan(&a.Username, &isOwner, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.IsOwner = isOwner == 1
		admins = append(admins, a)
	}
	return admins, rows.Err()
}

func (d *DB) DeleteAdmin(username string) error {
	res, err := d.db.Exec(
		`UPDATE admins SET deleted_at = ?, updated_at = ? WHERE username = ? AND is_owner = 0 AND deleted_at IS NULL`,
		now(), now(), username,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- token methods ---

func (d *DB) CreateToken(name, token, password string, serverPort int, expiryDays *int, quotaGB *float64) (*Token, error) {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	var expiresAt *string
	if expiryDays != nil {
		t := time.Now().AddDate(0, 0, *expiryDays).Format("2006-01-02 15:04:05")
		expiresAt = &t
	}
	t := now()
	_, err := d.db.Exec(
		`INSERT INTO tokens (id, name, token, password, server_port, quota_gb, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, name, token, password, serverPort, quotaGB, expiresAt, t, t,
	)
	if err != nil {
		return nil, err
	}
	return d.GetActiveToken(token)
}

func (d *DB) RenameToken(token, name string) error {
	res, err := d.db.Exec(
		`UPDATE tokens SET name = ?, updated_at = ? WHERE token = ? AND deleted_at IS NULL`,
		name, now(), token,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetActiveToken returns a token only if it is usable: not deleted, active, not suspended, not expired.
func (d *DB) GetActiveToken(token string) (*Token, error) {
	row := d.db.QueryRow(`
		SELECT id, name, token, password, server_port, quota_gb, used_bytes,
		       created_at, updated_at, expires_at, active, suspended
		FROM tokens
		WHERE token = ?
		  AND deleted_at IS NULL
		  AND active = 1
		  AND suspended = 0
		  AND (expires_at IS NULL OR expires_at > datetime('now'))
	`, token)
	return scanToken(row)
}

// GetToken returns any non-deleted token regardless of active/suspended/expiry.
func (d *DB) GetToken(token string) (*Token, error) {
	row := d.db.QueryRow(`
		SELECT id, name, token, password, server_port, quota_gb, used_bytes,
		       created_at, updated_at, expires_at, active, suspended
		FROM tokens WHERE token = ? AND deleted_at IS NULL
	`, token)
	return scanToken(row)
}

func (d *DB) ListTokens() ([]Token, error) {
	rows, err := d.db.Query(`
		SELECT id, name, token, password, server_port, quota_gb, used_bytes,
		       created_at, updated_at, expires_at, active, suspended
		FROM tokens WHERE deleted_at IS NULL ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tokens := []Token{}
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, *t)
	}
	return tokens, rows.Err()
}

// ActiveTokens returns tokens that should be registered with Xray.
func (d *DB) ActiveTokens() ([]Token, error) {
	rows, err := d.db.Query(`
		SELECT id, name, token, password, server_port, quota_gb, used_bytes,
		       created_at, updated_at, expires_at, active, suspended
		FROM tokens
		WHERE deleted_at IS NULL
		  AND active = 1
		  AND suspended = 0
		  AND (expires_at IS NULL OR expires_at > datetime('now'))
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

// ExpiredActiveTokens returns tokens that are past their expiry but still registered in Xray.
func (d *DB) ExpiredActiveTokens() ([]Token, error) {
	rows, err := d.db.Query(`
		SELECT id, name, token, password, server_port, quota_gb, used_bytes,
		       created_at, updated_at, expires_at, active, suspended
		FROM tokens
		WHERE deleted_at IS NULL
		  AND active = 1
		  AND suspended = 0
		  AND expires_at IS NOT NULL
		  AND expires_at <= datetime('now')
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

// SuspendToken marks a token as quota-suspended. Does not change active.
func (d *DB) SuspendToken(token string) error {
	_, err := d.db.Exec(
		`UPDATE tokens SET suspended = 1, updated_at = ? WHERE token = ? AND deleted_at IS NULL`,
		now(), token,
	)
	return err
}

// UpdateQuota sets a new quota and lifts suspension if the new quota is sufficient.
func (d *DB) UpdateQuota(token string, quotaGB *float64) error {
	res, err := d.db.Exec(`
		UPDATE tokens
		SET quota_gb   = ?,
		    suspended  = CASE
		                   WHEN ? IS NULL THEN 0
		                   WHEN used_bytes < CAST(? * 1000000000 AS INTEGER) THEN 0
		                   ELSE suspended
		                 END,
		    updated_at = ?
		WHERE token = ? AND deleted_at IS NULL`,
		quotaGB, quotaGB, quotaGB, now(), token,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ChangePassword updates an admin's password after verifying the current one.
func (d *DB) ChangePassword(username, currentPassword, newPassword string) error {
	var hash string
	err := d.db.QueryRow(
		`SELECT password_hash FROM admins WHERE username = ? AND deleted_at IS NULL`, username,
	).Scan(&hash)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)); err != nil {
		return errors.New("current password incorrect")
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(
		`UPDATE admins SET password_hash = ?, updated_at = ? WHERE username = ?`,
		string(newHash), now(), username,
	)
	return err
}

// ExtendExpiry adds days to a token's expiry (from now if no expiry set).
func (d *DB) ExtendExpiry(token string, days int) error {
	res, err := d.db.Exec(`
		UPDATE tokens
		SET expires_at = datetime(
		        CASE
		          WHEN expires_at IS NULL OR expires_at <= datetime('now') THEN datetime('now')
		          ELSE expires_at
		        END,
		        '+' || ? || ' days'
		    ),
		    updated_at = ?
		WHERE token = ? AND deleted_at IS NULL`,
		days, now(), token,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetSuspended manually suspends or resumes a token.
func (d *DB) SetSuspended(token string, suspend bool) error {
	v := 0
	if suspend {
		v = 1
	}
	res, err := d.db.Exec(
		`UPDATE tokens SET suspended = ?, updated_at = ? WHERE token = ? AND deleted_at IS NULL AND active = 1`,
		v, now(), token,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ResetUsage zeroes used_bytes without changing suspension state.
func (d *DB) ResetUsage(token string) error {
	res, err := d.db.Exec(
		`UPDATE tokens SET used_bytes = 0, updated_at = ? WHERE token = ? AND deleted_at IS NULL`,
		now(), token,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteToken soft-deletes a token.
func (d *DB) DeleteToken(token string) error {
	res, err := d.db.Exec(
		`UPDATE tokens SET deleted_at = ?, updated_at = ?, active = 0 WHERE token = ? AND deleted_at IS NULL`,
		now(), now(), token,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) AddStats(increments map[string]int64) error {
	for token, delta := range increments {
		if delta <= 0 {
			continue
		}
		if _, err := d.db.Exec(
			`UPDATE tokens SET used_bytes = used_bytes + ?, updated_at = ?
			 WHERE token = ? AND deleted_at IS NULL AND active = 1`,
			delta, now(), token,
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
	var active, suspended int
	err := s.Scan(
		&t.ID, &t.Name, &t.Token, &t.Password, &t.ServerPort,
		&t.QuotaGB, &t.UsedBytes, &t.CreatedAt, &t.UpdatedAt, &t.ExpiresAt,
		&active, &suspended,
	)
	if err != nil {
		return nil, err
	}
	t.Active = active == 1
	t.Suspended = suspended == 1
	return &t, nil
}
