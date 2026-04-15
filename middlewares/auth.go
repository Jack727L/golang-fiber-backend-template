package middlewares

import (
	"strings"
	"time"

	"github.com/yourusername/go-api-starter/core"
	db "github.com/yourusername/go-api-starter/db/sqlc"
	"github.com/gofiber/fiber/v2"
)

// UserAuthMiddleware validates the Bearer token in the Authorization header
// against the user_tokens table. On success it stores userID on the context.
func UserAuthMiddleware(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header with Bearer token is required",
			})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Empty token"})
		}

		tokenRecord, err := database.Queries.GetUserTokenByAccessToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		// Check expiry
		if tokenRecord.AccessTokenExpiresAt.Valid && time.Now().After(tokenRecord.AccessTokenExpiresAt.Time) {
			// Deactivate the expired token asynchronously (fire and forget)
			go func() {
				_ = database.Queries.DeactivateToken(c.Context(), token)
			}()
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token has expired"})
		}

		c.Locals("userID", tokenRecord.UserID)
		c.Locals("token", token)
		c.Locals("tokenRecord", tokenRecord)
		return c.Next()
	}
}

// RequireAuth is a convenience alias for UserAuthMiddleware.
func RequireAuth(database *db.DB) fiber.Handler {
	return UserAuthMiddleware(database)
}

// GetUserID extracts the authenticated user ID from Fiber locals.
// Returns 0 if not set.
func GetUserID(c *fiber.Ctx) int32 {
	if id, ok := c.Locals("userID").(int32); ok {
		return id
	}
	core.LogDebug(c, "GetUserID called but userID not set in context")
	return 0
}
