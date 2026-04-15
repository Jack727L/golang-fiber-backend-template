package services

import (
	"github.com/yourusername/go-api-starter/core"
	"github.com/yourusername/go-api-starter/core/jobs"
	db "github.com/yourusername/go-api-starter/db/sqlc"
	"github.com/yourusername/go-api-starter/middlewares"
	models "github.com/yourusername/go-api-starter/services/models"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserService struct {
	db            *db.DB
	smartExecutor *jobs.SmartExecutor
}

func NewUserService(database *db.DB) *UserService {
	return &UserService{
		db:            database,
		smartExecutor: jobs.NewSmartExecutor(database),
	}
}

// GetMe returns the authenticated user's profile.
//
// @Summary      Get current user
// @Description  Returns the profile of the authenticated user
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.UserResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /users/me [get]
func (s *UserService) GetMe(c *fiber.Ctx) error {
	userID := middlewares.GetUserID(c)

	// Background: update last_active_at (sync in tests, async in prod)
	_ = s.smartExecutor.UpdateUserLastActive(c.Context(), userID)

	user, err := s.db.Queries.GetUserByID(c.Context(), userID)
	if err != nil {
		return core.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch user", err)
	}

	return c.Status(fiber.StatusOK).JSON(buildUserResponse(user))
}

// UpdateMe updates the authenticated user's profile.
//
// @Summary      Update current user
// @Description  Updates the name of the authenticated user
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      models.UpdateUserRequest  true  "Update payload"
// @Success      200      {object}  models.UserResponse
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /users/me [put]
func (s *UserService) UpdateMe(c *fiber.Ctx) error {
	userID := middlewares.GetUserID(c)
	req := c.Locals("body").(*models.UpdateUserRequest)

	name := pgtype.Text{Valid: false}
	if req.Name != nil && *req.Name != "" {
		name = pgtype.Text{String: *req.Name, Valid: true}
	}

	user, err := s.db.Queries.UpdateUser(c.Context(), db.UpdateUserParams{
		ID:   userID,
		Name: name,
	})
	if err != nil {
		return core.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to update user", err)
	}

	return c.Status(fiber.StatusOK).JSON(buildUserResponse(user))
}

// DeleteMe removes the authenticated user account.
//
// @Summary      Delete current user
// @Description  Permanently deletes the authenticated user account
// @Tags         users
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /users/me [delete]
func (s *UserService) DeleteMe(c *fiber.Ctx) error {
	userID := middlewares.GetUserID(c)

	if err := s.db.Queries.DeleteUser(c.Context(), userID); err != nil {
		return core.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to delete user", err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Account deleted"})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func buildUserResponse(u db.User) models.UserResponse {
	resp := models.UserResponse{
		ID:    u.ID,
		Email: u.Email,
	}
	if u.Name.Valid {
		resp.Name = &u.Name.String
	}
	return resp
}
