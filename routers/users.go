package routers

import (
	db "github.com/yourusername/go-api-starter/db/sqlc"
	"github.com/yourusername/go-api-starter/middlewares"
	services "github.com/yourusername/go-api-starter/services"
	models "github.com/yourusername/go-api-starter/services/models"
	"github.com/gofiber/fiber/v2"
)

func SetupUsersRoutes(app *fiber.App, database *db.DB) {
	userService := services.NewUserService(database)

	users := app.Group("/users", middlewares.UserAuthMiddleware(database))

	users.Get("/me", userService.GetMe)
	users.Put("/me",
		middlewares.ValidateBody(&models.UpdateUserRequest{}),
		userService.UpdateMe,
	)
	users.Delete("/me", userService.DeleteMe)
}
