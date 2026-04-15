package base

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/suite"

	db "github.com/yourusername/go-api-starter/db/sqlc"
	"github.com/yourusername/go-api-starter/setup"
)

func init() {
	_ = os.Setenv("TZ", "UTC")
	time.Local = time.UTC
}

// BaseTestSuite provides a self-contained HTTP + PostgreSQL environment per test suite.
// Each suite gets its own isolated Postgres testcontainer and a running Fiber server.
type BaseTestSuite struct {
	suite.Suite
	App            *fiber.App
	Port           string
	DbConn         *db.DB
	testContainers *TestContainers
}

func (suite *BaseTestSuite) SetupSuite() {
	useDockerCompose := os.Getenv("USE_DOCKER_COMPOSE") == "true"

	if useDockerCompose {
		// Async/docker-compose mode: env vars are set by the test script.
		suite.testContainers = nil
	} else {
		// Default (sync) mode: spin up an isolated testcontainer.
		suite.testContainers = SetupPostgres(suite.T())
		suite.testContainers.SetTestEnv()
	}

	suite.App, suite.DbConn = setup.SetupApp()

	port, err := availablePort()
	suite.Require().NoError(err)
	suite.Port = port

	go func() {
		defer suite.DbConn.Close()
		if err := suite.App.Listen(":" + suite.Port); err != nil {
			fmt.Printf("Test server stopped: %v\n", err)
		}
	}()

	suite.waitForReady()
}

func (suite *BaseTestSuite) TearDownSuite() {
	_ = suite.App.Shutdown()
	if suite.DbConn != nil {
		suite.DbConn.Close()
	}
}

// TearDownTest truncates all mutable tables after every test for full isolation.
func (suite *BaseTestSuite) TearDownTest() {
	suite.truncateTables()
}

// RunTestSuite is a convenience wrapper around suite.Run.
func RunTestSuite(t *testing.T, s suite.TestingSuite) {
	suite.Run(t, s)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func (suite *BaseTestSuite) waitForReady() {
	url := fmt.Sprintf("http://localhost:%s/healthz", suite.Port)
	client := &http.Client{Timeout: 500 * time.Millisecond}

	delay := 50 * time.Millisecond
	for attempt := 1; attempt <= 30; attempt++ {
		if resp, err := client.Get(url); err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		if delay < 500*time.Millisecond {
			delay *= 2
		}
		time.Sleep(delay)
	}
	fmt.Printf("Warning: app health check timed out — proceeding anyway\n")
}

// mutableTables is the ordered list of tables to truncate (children before parents).
// Extend this list when you add new tables.
var mutableTables = []string{
	"user_tokens",
	"users",
}

func (suite *BaseTestSuite) truncateTables() {
	ctx := suite.T().Context()
	quoted := make([]string, len(mutableTables))
	for i, t := range mutableTables {
		quoted[i] = fmt.Sprintf(`"%s"`, t)
	}
	sql := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE",
		joinStrings(quoted, ", "))
	if _, err := suite.DbConn.Pool.Exec(ctx, sql); err != nil {
		suite.T().Logf("Warning: truncate failed: %v", err)
	}
}

func availablePort() (string, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}
	defer func() { _ = l.Close() }()
	return fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port), nil
}

func joinStrings(ss []string, sep string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}
