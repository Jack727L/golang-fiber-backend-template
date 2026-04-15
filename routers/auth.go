package routers

import (
	db "github.com/yourusername/go-api-starter/db/sqlc"
	"github.com/yourusername/go-api-starter/middlewares"
	services "github.com/yourusername/go-api-starter/services"
	models "github.com/yourusername/go-api-starter/services/models"
	"github.com/gofiber/fiber/v2"
)

func SetupAuthRoutes(app *fiber.App, database *db.DB) {
	authService := services.NewAuthService(database)

	auth := app.Group("/auth")
	auth.Post("/register",
		middlewares.ValidateBody(&models.RegisterRequest{}),
		authService.Register,
	)
	auth.Post("/login",
		middlewares.ValidateBody(&models.LoginRequest{}),
		authService.Login,
	)
	auth.Post("/refresh",
		middlewares.ValidateBody(&models.RefreshRequest{}),
		authService.Refresh,
	)
	auth.Post("/logout",
		middlewares.UserAuthMiddleware(database),
		authService.Logout,
	)
}
