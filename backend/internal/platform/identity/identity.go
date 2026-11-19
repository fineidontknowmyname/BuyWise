package identity

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"selfemi/internal/platform/database"
	"selfemi/internal/platform/httpjson"
)

const sessionLifetime = 8 * time.Hour
const passwordRounds = 600000

type Handler struct {
	pool         *pgxpool.Pool
	passwordWork chan struct{}
	limitMu      sync.Mutex
	remaining    float64
	refillAt     time.Time
}

func New(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool, passwordWork: make(chan struct{}, 4), remaining: 12, refillAt: time.Now()}
}

func (h *Handler) allowPasswordRequest() bool {
	h.limitMu.Lock()
	defer h.limitMu.Unlock()
	now := time.Now()
	h.remaining = min(12, h.remaining+now.Sub(h.refillAt).Seconds())
	h.refillAt = now
	if h.remaining < 1 {
		return false
	}
	h.remaining--
	return true
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func derive(password string, salt []byte) []byte {
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordRounds, 32)
	if err != nil {
		panic(err)
	}
	return key
}

func hashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(derive(password, salt))
}

func passwordMatches(password, stored string) bool {
	left, right, _ := strings.Cut(stored, ":")
	salt, _ := hex.DecodeString(left)
	key, _ := hex.DecodeString(right)
	valid := len(salt) == 16 && len(key) == 32
	if !valid {
		salt, key = make([]byte, 16), make([]byte, 32)
	}
	return subtle.ConstantTimeCompare(derive(password, salt), key) == 1 && valid
}

func tokenHash(r *http.Request) []byte {
	scheme, value, _ := strings.Cut(r.Header.Get("Authorization"), " ")
	if scheme != "Bearer" || len(value) != 43 {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(raw) != 32 {
		return nil
	}
	sum := sha256.Sum256(raw)
	return sum[:]
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.authenticate(true))
	mux.HandleFunc("POST /auth/login", h.authenticate(false))
	mux.HandleFunc("GET /auth/session", h.session)
	mux.HandleFunc("POST /auth/logout", h.logout)
	mux.HandleFunc("POST /auth/refresh", h.refresh)
}

func (h *Handler) authenticate(register bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var input credentials
		if err := httpjson.Decode(w, r, &input); err != nil {
			httpjson.WriteError(w, 400, "invalid_credentials", "Provide an email and password.")
			return
		}
		email := strings.ToLower(strings.TrimSpace(input.Email))
		address, err := mail.ParseAddress(email)
		if err != nil || address.Address != email || len(email) > 254 || len(input.Password) > 256 || len(input.Password) == 0 || (register && len([]rune(input.Password)) < 15) {
			httpjson.WriteError(w, 400, "invalid_credentials", "Provide a valid email and a password of at least 15 characters when registering.")
			return
		}
		select {
		case h.passwordWork <- struct{}{}:
			defer func() { <-h.passwordWork }()
		default:
			httpjson.WriteError(w, 429, "try_later", "Try again shortly.")
			return
		}

		ctx := r.Context()
		if !h.allowPasswordRequest() {
			w.Header().Set("Retry-After", "1")
			httpjson.WriteError(w, 429, "try_later", "Try again shortly.")
			return
		}
		var id string
		if register {
			err = h.register(ctx, email, hashPassword(input.Password), &id)
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				httpjson.WriteError(w, 409, "registration_failed", "Unable to register this account.")
				return
			}
		} else {
			var stored string
			err = h.pool.QueryRow(ctx, "SELECT id::text, password_hash FROM identity.users WHERE email = $1", email).Scan(&id, &stored)
			if err == nil || errors.Is(err, pgx.ErrNoRows) {
				if !passwordMatches(input.Password, stored) {
					httpjson.WriteError(w, 401, "invalid_credentials", "Email or password is incorrect.")
					return
				}
			}
		}
		if err != nil {
			unavailable(w)
			return
		}
		raw := make([]byte, 32)
		rand.Read(raw)
		sum := sha256.Sum256(raw)
		if _, err := h.pool.Exec(ctx, "WITH created AS (INSERT INTO identity.sessions VALUES ($1, $2::uuid, $3) RETURNING user_id) INSERT INTO identity.session_events (user_id, event) SELECT user_id, 'signed_in' FROM created", sum[:], id, time.Now().Add(sessionLifetime)); err != nil {
			unavailable(w)
			return
		}
		httpjson.Write(w, 200, map[string]any{"token": base64.RawURLEncoding.EncodeToString(raw), "expiresIn": int(sessionLifetime.Seconds())})
	}
}

func (h *Handler) register(ctx context.Context, email, hash string, id *string) error {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	if err := tx.QueryRow(ctx, "INSERT INTO identity.users (email, password_hash) VALUES ($1,$2) RETURNING id::text", email, hash).Scan(id); err != nil {
		return err
	}
	if err := setScope(ctx, tx, *id); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO finance.categories (id, name, kind, committed) VALUES
        (uuidv7(), 'Salary', 'income', false), (uuidv7(), 'Other income', 'income', false),
        (uuidv7(), 'Rent', 'expense', true), (uuidv7(), 'Utilities', 'expense', true),
        (uuidv7(), 'Insurance', 'expense', true), (uuidv7(), 'Loan payments', 'expense', true),
        (uuidv7(), 'Groceries', 'expense', false), (uuidv7(), 'Transport', 'expense', false),
        (uuidv7(), 'Dining out', 'expense', false), (uuidv7(), 'Shopping', 'expense', false),
        (uuidv7(), 'Entertainment', 'expense', false), (uuidv7(), 'Other expense', 'expense', false)`)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (h *Handler) user(r *http.Request) (id, email string, err error) {
	hash := tokenHash(r)
	if hash == nil {
		return "", "", pgx.ErrNoRows
	}
	err = h.pool.QueryRow(r.Context(), `SELECT u.id::text, u.email FROM identity.sessions s
        JOIN identity.users u ON u.id = s.user_id WHERE s.token_hash = $1 AND s.expires_at > now()`, hash).Scan(&id, &email)
	return
}

func (h *Handler) session(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	id, email, err := h.user(r)
	if err != nil {
		sessionError(w, err)
		return
	}
	httpjson.Write(w, 200, map[string]string{"id": id, "email": email})
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	previous := tokenHash(r)
	if previous == nil {
		sessionError(w, pgx.ErrNoRows)
		return
	}
	raw := make([]byte, 32)
	rand.Read(raw)
	replacement := sha256.Sum256(raw)
	var userID string
	err := h.pool.QueryRow(r.Context(), `WITH prior AS (
        DELETE FROM identity.sessions WHERE token_hash = $1 AND expires_at > now() RETURNING user_id
    )
    INSERT INTO identity.sessions (token_hash, user_id, expires_at)
    SELECT $2, user_id, $3 FROM prior RETURNING user_id::text`,
		previous, replacement[:], time.Now().Add(sessionLifetime)).Scan(&userID)
	if err != nil {
		sessionError(w, err)
		return
	}
	httpjson.Write(w, 200, map[string]any{"token": base64.RawURLEncoding.EncodeToString(raw), "expiresIn": int(sessionLifetime.Seconds())})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if hash := tokenHash(r); hash != nil {
		if _, err := h.pool.Exec(r.Context(), "WITH removed AS (DELETE FROM identity.sessions WHERE token_hash = $1 RETURNING user_id) INSERT INTO identity.session_events (user_id, event) SELECT user_id, 'signed_out' FROM removed", hash); err != nil {
			unavailable(w)
			return
		}
	}
	httpjson.Write(w, 200, map[string]bool{"signedOut": true})
}

func sessionError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		httpjson.WriteError(w, 401, "sign_in_required", "Sign in to continue.")
		return
	}
	unavailable(w)
}

func unavailable(w http.ResponseWriter) {
	httpjson.WriteError(w, 503, "service_unavailable", "BuyWise is temporarily unavailable.")
}

func setScope(ctx context.Context, tx pgx.Tx, id string) error {
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE buywise_tenant"); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, "SELECT set_config('buywise.user_id', $1, true)", id)
	return err
}

func (h *Handler) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _, err := h.user(r)
		if err != nil {
			sessionError(w, err)
			return
		}
		ctx := r.Context()
		tx, err := h.pool.Begin(ctx)
		if err != nil {
			unavailable(w)
			return
		}
		defer tx.Rollback(context.WithoutCancel(ctx))
		if err := setScope(ctx, tx, id); err != nil {
			unavailable(w)
			return
		}
		response := newResponse()
		next.ServeHTTP(response, r.WithContext(database.WithTenantTransaction(ctx, tx)))
		if response.status < 400 {
			if err := tx.Commit(ctx); err != nil {
				unavailable(w)
				return
			}
		}
		response.writeTo(w)
	})
}
