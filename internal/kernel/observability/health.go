package observability

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type Checker interface {
	Ping(context.Context) error
}

type Readiness struct {
	timeout  time.Duration
	checkers []Checker
}

func NewReadiness(timeout time.Duration, checkers ...Checker) *Readiness {
	if timeout <= 0 {
		timeout = time.Second
	}
	return &Readiness{timeout: timeout, checkers: checkers}
}

func (r *Readiness) Ready(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	for _, checker := range r.checkers {
		if err := checker.Ping(ctx); err != nil {
			return false
		}
	}
	return true
}

type sqlChecker struct{ db *sql.DB }

func NewSQLChecker(db *sql.DB) Checker { return sqlChecker{db: db} }

func (c sqlChecker) Ping(ctx context.Context) error { return c.db.PingContext(ctx) }

type redisChecker struct{ client redis.Cmdable }

func NewRedisChecker(client redis.Cmdable) Checker { return redisChecker{client: client} }

func (c redisChecker) Ping(ctx context.Context) error { return c.client.Ping(ctx).Err() }

func RegisterHealth(mux *http.ServeMux, readiness *Readiness) {
	mux.HandleFunc("/livez", func(w http.ResponseWriter, _ *http.Request) {
		writeHealth(w, http.StatusOK, map[string]string{"status": "up"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !readiness.Ready(r.Context()) {
			writeHealth(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeHealth(w, http.StatusOK, map[string]string{"status": "ready"})
	})
}

func writeHealth(w http.ResponseWriter, status int, body map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
