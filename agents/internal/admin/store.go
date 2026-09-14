package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Store struct {
	pool *pgxpool.Pool
}

type User struct {
	ID          int64
	Name        string
	Phone       string
	Email       string
	Role        string
	Active      bool
	HasPassword bool
	Caps        []string
	CreatedAt   time.Time
}

type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
	User      User
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 2
	cfg.MinConns = 0
	cfg.MaxConnLifetime = time.Hour
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  phone TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL DEFAULT '',
  password_hash TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT 'viewer',
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS users_phone_uq ON users (phone) WHERE phone <> '';
CREATE TABLE IF NOT EXISTS capabilities (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  cap TEXT NOT NULL,
  PRIMARY KEY (user_id, cap)
);
CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS sessions (
  token TEXT PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL
);
`)
	if err != nil {
		return err
	}
	return s.migrateProjects(ctx)
}

func (s *Store) Seed(ctx context.Context, name, phone, email, password string) error {
	phone = normalizePhone(phone)
	n, err := s.countUsers(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		u, err := s.CreateUser(ctx, name, phone, email, password, "owner", []string{"web_admin", "whatsapp_accounts", "zoho_write"})
		if err != nil {
			return err
		}
		_ = u
	} else if phone != "" && password != "" {
		if err := s.ensureOwnerPassword(ctx, phone, password); err != nil {
			return err
		}
	}
	for k, v := range defaultSettings() {
		if err := s.seedSetting(ctx, k, v); err != nil {
			return err
		}
	}
	for _, p := range defaultProjects() {
		if err := s.seedProject(ctx, p); err != nil {
			return err
		}
	}
	return s.purgeExpired(ctx)
}

func (s *Store) countUsers(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) seedSetting(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO settings (key, value) VALUES ($1, $2::jsonb)
ON CONFLICT (key) DO NOTHING
`, key, raw)
	return err
}

func (s *Store) ensureOwnerPassword(ctx context.Context, phone, password string) error {
	var id int64
	var hash string
	err := s.pool.QueryRow(ctx, `SELECT id, password_hash FROM users WHERE phone = $1`, phone).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if hash != "" {
		return nil
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE users SET password_hash = $1, role = 'owner' WHERE id = $2`, string(h), id)
	return err
}

func (s *Store) CreateUser(ctx context.Context, name, phone, email, password, role string, caps []string) (User, error) {
	name = strings.TrimSpace(name)
	phone = normalizePhone(phone)
	email = strings.ToLower(strings.TrimSpace(email))
	role = strings.TrimSpace(role)
	if name == "" {
		return User{}, errors.New("name is required")
	}
	if role == "" {
		role = "viewer"
	}
	hash := ""
	if strings.TrimSpace(password) != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return User{}, err
		}
		hash = string(h)
	}
	var id int64
	err := s.pool.QueryRow(ctx, `
INSERT INTO users (name, phone, email, password_hash, role, active)
VALUES ($1, $2, $3, $4, $5, true)
RETURNING id
`, name, phone, email, hash, role).Scan(&id)
	if err != nil {
		return User{}, friendlyDBErr(err)
	}
	if err := s.replaceCaps(ctx, id, caps); err != nil {
		return User{}, err
	}
	return s.GetUser(ctx, id)
}

func (s *Store) UpdateUser(ctx context.Context, id int64, name, phone, email, password, role string, active bool, caps []string) error {
	name = strings.TrimSpace(name)
	phone = normalizePhone(phone)
	email = strings.ToLower(strings.TrimSpace(email))
	if name == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(password) != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = s.pool.Exec(ctx, `
UPDATE users SET name=$1, phone=$2, email=$3, role=$4, active=$5, password_hash=$6 WHERE id=$7
`, name, phone, email, role, active, string(h), id)
		if err != nil {
			return friendlyDBErr(err)
		}
	} else {
		_, err := s.pool.Exec(ctx, `
UPDATE users SET name=$1, phone=$2, email=$3, role=$4, active=$5 WHERE id=$6
`, name, phone, email, role, active, id)
		if err != nil {
			return friendlyDBErr(err)
		}
	}
	return s.replaceCaps(ctx, id, caps)
}

func (s *Store) replaceCaps(ctx context.Context, id int64, caps []string) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM capabilities WHERE user_id = $1`, id); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, c := range caps {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		if _, err := s.pool.Exec(ctx, `INSERT INTO capabilities (user_id, cap) VALUES ($1, $2)`, id, c); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetUser(ctx context.Context, id int64) (User, error) {
	u := User{}
	var hash string
	err := s.pool.QueryRow(ctx, `
SELECT id, name, phone, email, password_hash, role, active, created_at FROM users WHERE id = $1
`, id).Scan(&u.ID, &u.Name, &u.Phone, &u.Email, &hash, &u.Role, &u.Active, &u.CreatedAt)
	if err != nil {
		return User{}, err
	}
	u.HasPassword = hash != ""
	u.Caps, err = s.capsFor(ctx, id)
	return u, err
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, name, phone, email, password_hash, role, active, created_at
FROM users ORDER BY id
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		var hash string
		if err := rows.Scan(&u.ID, &u.Name, &u.Phone, &u.Email, &hash, &u.Role, &u.Active, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.HasPassword = hash != ""
		u.Caps, err = s.capsFor(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) capsFor(ctx context.Context, id int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT cap FROM capabilities WHERE user_id = $1 ORDER BY cap`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var caps []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		caps = append(caps, c)
	}
	return caps, rows.Err()
}

func (s *Store) Authenticate(ctx context.Context, login, password string) (User, error) {
	login = strings.TrimSpace(login)
	phone := normalizePhone(login)
	email := strings.ToLower(login)
	var id int64
	var hash string
	err := s.pool.QueryRow(ctx, `
SELECT id, password_hash FROM users
WHERE active = true AND password_hash <> ''
  AND (phone = $1 OR email = $2)
LIMIT 1
`, phone, email).Scan(&id, &hash)
	if err != nil {
		return User{}, errors.New("invalid login")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return User{}, errors.New("invalid login")
	}
	return s.GetUser(ctx, id)
}

func (s *Store) CreateSession(ctx context.Context, userID int64, ttl time.Duration) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	exp := time.Now().Add(ttl)
	_, err := s.pool.Exec(ctx, `INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`, token, userID, exp)
	return token, err
}

func (s *Store) Session(ctx context.Context, token string) (Session, error) {
	var sess Session
	err := s.pool.QueryRow(ctx, `
SELECT token, user_id, expires_at FROM sessions WHERE token = $1 AND expires_at > now()
`, token).Scan(&sess.Token, &sess.UserID, &sess.ExpiresAt)
	if err != nil {
		return Session{}, err
	}
	sess.User, err = s.GetUser(ctx, sess.UserID)
	return sess, err
}

func (s *Store) DeleteSession(ctx context.Context, token string) {
	_, _ = s.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
}

func (s *Store) purgeExpired(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
	return err
}

func (s *Store) WhatsAppAccounts(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
SELECT DISTINCT phone FROM users
WHERE active = true AND phone <> ''
  AND id IN (SELECT user_id FROM capabilities WHERE cap = 'whatsapp_accounts')
ORDER BY phone
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		p = normalizePhone(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out, rows.Err()
}

func (s *Store) Settings(ctx context.Context) (map[string]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT key, value FROM settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k string
		var raw []byte
		if err := rows.Scan(&k, &raw); err != nil {
			return nil, err
		}
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			out[k] = string(raw)
			continue
		}
		switch t := v.(type) {
		case string:
			out[k] = t
		default:
			b, _ := json.Marshal(t)
			out[k] = string(b)
		}
	}
	return out, rows.Err()
}

func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	key = strings.TrimSpace(key)
	raw, err := json.Marshal(strings.TrimSpace(value))
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO settings (key, value, updated_at) VALUES ($1, $2::jsonb, now())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
`, key, raw)
	return err
}

func (s *Store) Ready(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func HasCap(u User, cap string) bool {
	if u.Role == "owner" {
		return true
	}
	return hasStoredCap(u, cap)
}

func (s *Store) CountOwners(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'owner' AND active = true`).Scan(&n)
	return n, err
}

func (s *Store) DeleteUser(ctx context.Context, actor *User, id int64) error {
	target, err := s.GetUser(ctx, id)
	if err != nil {
		return err
	}
	owners, err := s.CountOwners(ctx)
	if err != nil {
		return err
	}
	if err := removalBlocked(actor, target, owners); err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func removalBlocked(actor *User, target User, owners int) error {
	if actor != nil && actor.ID == target.ID {
		return errors.New("you cannot remove yourself")
	}
	if target.Role == "owner" && owners <= 1 {
		return errors.New("the last owner cannot be removed")
	}
	return nil
}

func lastOwnerLocked(target User, newRole string, active bool, owners int) error {
	if target.Role != "owner" || owners > 1 {
		return nil
	}
	if newRole != "owner" {
		return errors.New("the last owner cannot be demoted")
	}
	if !active {
		return errors.New("the last owner cannot be turned off")
	}
	return nil
}

func friendlyDBErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "users_phone_uq") {
		return errors.New("that phone is already on the roster")
	}
	if strings.Contains(msg, "projects_slug") {
		return errors.New("that project slug is already in use")
	}
	return err
}
