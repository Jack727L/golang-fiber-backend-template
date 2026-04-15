package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/yourusername/go-api-starter/env"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// DB wraps a pgxpool and the generated SQLC Queries helper.
type DB struct {
	Pool    *pgxpool.Pool
	Queries *Queries
}

// PoolOptions lets callers override pool sizing. Zero values fall back to env vars.
type PoolOptions struct {
	MaxConns        int
	MinConns        int
	ConnMaxLifetime int // seconds
	ConnMaxIdleTime int // seconds
}

// NewDB creates a DB using env-var defaults.
func NewDB() (*DB, error) {
	return NewDBWithOptions(nil)
}

// NewDBWithOptions creates a DB with optional pool-size overrides.
func NewDBWithOptions(opts *PoolOptions) (*DB, error) {
	_ = godotenv.Load()

	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	database := os.Getenv("POSTGRES_DATABASE")
	sslMode := os.Getenv("POSTGRES_SSLMODE")

	maxConns := getEnvInt("DB_MAX_OPEN_CONNS", 10)
	minConns := getEnvInt("DB_MAX_IDLE_CONNS", 5)
	connMaxLifetime := getEnvInt("DB_CONN_MAX_LIFETIME", 300)
	connMaxIdleTime := 30

	if opts != nil {
		if opts.MaxConns > 0 {
			maxConns = opts.MaxConns
		}
		if opts.MinConns > 0 {
			minConns = opts.MinConns
		}
		if opts.ConnMaxLifetime > 0 {
			connMaxLifetime = opts.ConnMaxLifetime
		}
		if opts.ConnMaxIdleTime > 0 {
			connMaxIdleTime = opts.ConnMaxIdleTime
		}
	}

	// Test-mode defaults (testcontainers sets these via env vars)
	if env.IsTestMode() {
		if host == "" {
			host = "127.0.0.1"
		}
		if port == "" {
			port = "5432"
		}
		if user == "" {
			user = "testuser"
		}
		if password == "" {
			password = "testpassword"
		}
		if database == "" {
			database = "testdb"
		}
		sslMode = "disable"
	}

	if host == "" || port == "" || user == "" || password == "" || database == "" {
		log.Fatal("Missing required PostgreSQL env vars: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DATABASE")
	}

	if sslMode == "" {
		sslMode = "require"
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&statement_timeout=30000&idle_in_transaction_session_timeout=60000&lock_timeout=10000",
		user, password, host, port, database, sslMode,
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("error parsing DSN: %w", err)
	}

	poolCfg.MaxConns = int32(maxConns)
	poolCfg.MinConns = int32(minConns)
	poolCfg.MaxConnLifetime = time.Duration(connMaxLifetime) * time.Second
	poolCfg.MaxConnIdleTime = time.Duration(connMaxIdleTime) * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("error creating connection pool: %w", err)
	}

	if err = pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	if !env.IsTestMode() {
		fmt.Printf("Database connected: MaxConns=%d, MinConns=%d, lifetime=%ds, idle=%ds\n",
			maxConns, minConns, connMaxLifetime, connMaxIdleTime)
	}

	return &DB{Pool: pool, Queries: New(pool)}, nil
}

// Close closes the underlying connection pool.
func (d *DB) Close() {
	if d.Pool != nil {
		d.Pool.Close()
	}
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
