package setup

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/yourusername/go-api-starter/core"
	"github.com/yourusername/go-api-starter/core/jobs"
	db "github.com/yourusername/go-api-starter/db/sqlc"
	"github.com/yourusername/go-api-starter/env"
	"github.com/yourusername/go-api-starter/middlewares"
	"github.com/yourusername/go-api-starter/routers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func SetupApp() (*fiber.App, *db.DB) {
	_ = godotenv.Load()

	// ── Redis ──────────────────────────────────────────────────────────────
	jobs.InitRedis()

	// ── Fiber ──────────────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		DisableStartupMessage: env.IsTestMode(),
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          60 * time.Second,
		IdleTimeout:           120 * time.Second,
	})

	// ── CORS ───────────────────────────────────────────────────────────────
	// Adjust AllowOriginsFunc to match your frontend domain(s).
	allowedOriginRe := regexp.MustCompile(`^https?://(localhost.*|127\.0\.0\.1.*)$`)
	app.Use(cors.New(cors.Config{
		AllowHeaders: "Origin,Content-Type,Accept,Content-Length,Accept-Language,Authorization,Accept-Encoding,Connection,Access-Control-Allow-Origin",
		AllowOriginsFunc: func(origin string) bool {
			return origin == "" || allowedOriginRe.MatchString(origin)
		},
		AllowCredentials: true,
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
	}))

	// ── Logging ────────────────────────────────────────────────────────────
	if !env.IsTestMode() {
		app.Use(middlewares.LoggingMiddleware())
	}

	// ── Database ───────────────────────────────────────────────────────────
	dbConn, err := db.NewDB()
	if err != nil {
		core.LogError(nil, err)
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err := dbConn.Pool.Ping(context.Background()); err != nil {
		core.LogError(nil, err)
		log.Fatalf("Database ping failed: %v", err)
	}
	if !env.IsTestMode() {
		fmt.Println("Database connected successfully!")
	}

	// ── Health probes ──────────────────────────────────────────────────────
	app.Get("/healthz", func(c *fiber.Ctx) error { return c.SendString("OK") })
	app.Get("/healthcheck", func(c *fiber.Ctx) error { return c.SendString("OK") })
	app.Get("/readyz", func(c *fiber.Ctx) error {
		if err := dbConn.Pool.Ping(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "not ready", "database": "unhealthy",
			})
		}
		return c.JSON(fiber.Map{"status": "ready", "database": "healthy"})
	})

	// ── Routes ─────────────────────────────────────────────────────────────
	routers.SetupAuthRoutes(app, dbConn)
	routers.SetupUsersRoutes(app, dbConn)

	if !env.IsTestMode() {
		fmt.Println("App setup complete.")
	}
	return app, dbConn
}

func StartApp(app *fiber.App, dbConn *db.DB) {
	fmt.Println("Starting server on :3000 …")

	if env.IsTestMode() {
		go func() {
			defer dbConn.Close()
			if err := app.Listen(":3000"); err != nil {
				log.Fatalf("Server error: %v", err)
			}
		}()
		return
	}

	defer dbConn.Close()
	if err := app.Listen(":3000"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
