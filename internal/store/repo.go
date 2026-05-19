package store

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type Repo struct {
	DB *sql.DB
}

func NewRepo(db *sql.DB) *Repo { return &Repo{DB: db} }

type Session struct {
	Name      string `json:"name"`
	JID       string `json:"jid"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func (r *Repo) UpsertSession(ctx context.Context, name string) (*Session, error) {
	now := time.Now().Unix()
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO app_sessions(name,status,created_at,updated_at) VALUES(?,?,?,?)
		 ON CONFLICT(name) DO UPDATE SET updated_at=excluded.updated_at`,
		name, "STOPPED", now, now,
	)
	if err != nil {
		return nil, err
	}
	return r.GetSession(ctx, name)
}

func (r *Repo) GetSession(ctx context.Context, name string) (*Session, error) {
	row := r.DB.QueryRowContext(ctx,
		`SELECT name, COALESCE(jid,''), status, created_at, updated_at FROM app_sessions WHERE name = ?`, name)
	var s Session
	if err := row.Scan(&s.Name, &s.JID, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) ListSessions(ctx context.Context) ([]Session, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT name, COALESCE(jid,''), status, created_at, updated_at FROM app_sessions ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.Name, &s.JID, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repo) UpdateSessionStatus(ctx context.Context, name, status, jid string) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE app_sessions SET status=?, jid=COALESCE(NULLIF(?,''), jid), updated_at=? WHERE name=?`,
		status, jid, time.Now().Unix(), name)
	return err
}

func (r *Repo) DeleteSession(ctx context.Context, name string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM app_sessions WHERE name=?`, name)
	return err
}

type Webhook struct {
	ID          int64    `json:"id"`
	SessionName string   `json:"session_name"`
	URL         string   `json:"url"`
	Secret      string   `json:"secret"`
	Events      []string `json:"events"`
	Enabled     bool     `json:"enabled"`
	CreatedAt   int64    `json:"created_at"`
}

func (r *Repo) AddWebhook(ctx context.Context, w *Webhook) error {
	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO webhooks(session_name,url,secret,events,enabled,created_at) VALUES(?,?,?,?,?,?)`,
		w.SessionName, w.URL, w.Secret, strings.Join(w.Events, ","), boolToInt(w.Enabled), time.Now().Unix(),
	)
	if err != nil {
		return err
	}
	w.ID, _ = res.LastInsertId()
	return nil
}

func (r *Repo) ListWebhooksBySession(ctx context.Context, session string) ([]Webhook, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, session_name, url, secret, events, enabled, created_at FROM webhooks WHERE session_name=?`,
		session)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Webhook
	for rows.Next() {
		var w Webhook
		var events string
		var enabled int
		if err := rows.Scan(&w.ID, &w.SessionName, &w.URL, &w.Secret, &events, &enabled, &w.CreatedAt); err != nil {
			return nil, err
		}
		if events != "" {
			w.Events = strings.Split(events, ",")
		}
		w.Enabled = enabled == 1
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *Repo) DeleteWebhook(ctx context.Context, id int64) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM webhooks WHERE id=?`, id)
	return err
}

type APIKey struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	KeyHash   string `json:"-"`
	CreatedAt int64  `json:"created_at"`
	LastUsed  *int64 `json:"last_used,omitempty"`
}

func (r *Repo) AddAPIKey(ctx context.Context, name, hash string) (*APIKey, error) {
	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO api_keys(name,key_hash,created_at) VALUES(?,?,?)`,
		name, hash, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &APIKey{ID: id, Name: name, KeyHash: hash, CreatedAt: time.Now().Unix()}, nil
}

func (r *Repo) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, name, key_hash, created_at, last_used FROM api_keys ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKey
	for rows.Next() {
		var k APIKey
		var lu sql.NullInt64
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyHash, &k.CreatedAt, &lu); err != nil {
			return nil, err
		}
		if lu.Valid {
			k.LastUsed = &lu.Int64
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *Repo) DeleteAPIKey(ctx context.Context, id int64) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM api_keys WHERE id=?`, id)
	return err
}

func (r *Repo) DeleteAPIKeysByName(ctx context.Context, name string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM api_keys WHERE name=?`, name)
	return err
}

func (r *Repo) TouchAPIKey(ctx context.Context, hash string) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE api_keys SET last_used=? WHERE key_hash=?`, time.Now().Unix(), hash)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
