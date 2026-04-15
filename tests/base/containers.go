package base

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	db "github.com/yourusername/go-api-starter/db/sqlc"
)

// TestContainers holds references to live containers for a test run.
type TestContainers struct {
	Postgres     testcontainers.Container
	PostgresPool *pgxpool.Pool
	DB           *db.DB
	PostgresHost string
	PostgresPort string
}

// SetupPostgres starts a PostgreSQL testcontainer, runs sqitch migrations,
// and returns a TestContainers handle. The container is torn down via t.Cleanup.
func SetupPostgres(t *testing.T) *TestContainers {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpassword"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second),
				wait.ForListeningPort("5432/tcp").
					WithStartupTimeout(60*time.Second),
			).WithDeadline(90*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable", "timezone=UTC")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	var pool *pgxpool.Pool
	for attempt := 1; attempt <= 5; attempt++ {
		pool, err = pgxpool.New(ctx, connStr)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				break
			}
			pool.Close()
		}
		if attempt < 5 {
			t.Logf("DB connection attempt %d failed, retrying …", attempt)
			time.Sleep(500 * time.Millisecond)
		}
	}
	if err != nil {
		t.Fatalf("Failed to create connection pool after retries: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	if err := runMigrations(ctx, pool); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}
	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	return &TestContainers{
		Postgres:     pgContainer,
		PostgresPool: pool,
		DB:           &db.DB{Pool: pool, Queries: db.New(pool)},
		PostgresHost: host,
		PostgresPort: port.Port(),
	}
}

// SetTestEnv writes the container's connection details into env vars so
// SetupApp() uses the test container instead of a real database.
func (tc *TestContainers) SetTestEnv() {
	_ = os.Setenv("BACKEND_ENV", "test")
	_ = os.Setenv("POSTGRES_HOST", tc.PostgresHost)
	_ = os.Setenv("POSTGRES_PORT", tc.PostgresPort)
	_ = os.Setenv("POSTGRES_USER", "testuser")
	_ = os.Setenv("POSTGRES_PASSWORD", "testpassword")
	_ = os.Setenv("POSTGRES_DATABASE", "testdb")
	_ = os.Setenv("ASYNC_MODE", "false")
}

// ─── migration helpers ────────────────────────────────────────────────────────

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	root, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}

	planFile := filepath.Join(root, "db", "sqitch", "sqitch.plan")
	migrations, err := parseSqitchPlan(planFile)
	if err != nil {
		return fmt.Errorf("parse sqitch.plan: %w", err)
	}

	deployDir := filepath.Join(root, "db", "sqitch", "deploy")
	for _, m := range migrations {
		sql, err := os.ReadFile(filepath.Join(deployDir, m+".sql"))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("execute migration %s: %w", m, err)
		}
	}
	return nil
}

func parseSqitchPlan(planFile string) ([]string, error) {
	f, err := os.Open(planFile)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var names []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") {
			continue
		}
		name := line
		if idx := strings.IndexAny(line, " ["); idx != -1 {
			name = line[:idx]
		}
		if name != "" {
			names = append(names, name)
		}
	}
	return names, scanner.Err()
}

func findProjectRoot() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}
	dir := filepath.Dir(currentFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found in any parent directory")
		}
		dir = parent
	}
}
