package services

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/yourusername/go-api-starter/core"
	db "github.com/yourusername/go-api-starter/db/sqlc"
	models "github.com/yourusername/go-api-starter/services/models"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

type AuthService struct {
	db *db.DB
}

func NewAuthService(database *db.DB) *AuthService {
	return &AuthService{db: database}
}

// Register creates a new user and returns auth tokens.
//
// @Summary      Register a new user
// @Description  Creates a user account and returns access + refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.RegisterRequest  true  "Register payload"
// @Success      201      {object}  models.AuthResponse
// @Failure      400      {object}  map[string]string
// @Failure      409      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/register [post]
func (s *AuthService) Register(c *fiber.Ctx) error {
	req := c.Locals("body").(*models.RegisterRequest)

	// Check email uniqueness
	if _, err := s.db.Queries.GetUserByEmail(c.Context(), req.Email); err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Email already registered"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return core.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to process password", err)
	}

	name := pgtype.Text{Valid: false}
	if req.Name != "" {
		name = pgtype.Text{String: req.Name, Valid: true}
	}

	user, err := s.db.Queries.RegisterUser(c.Context(), db.RegisterUserParams{
		Email:          req.Email,
		Name:           name,
		HashedPassword: string(hash),
	})
	if err != nil {
		return core.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to create user", err)
	}

	resp, err := s.issueTokens(c, user.ID)
	if err != nil {
		return core.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to issue tokens", err)
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// Login authenticates a user and returns auth tokens.
//
// @Summary      Login
// @Description  Authenticates with email + password and returns access + refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.LoginRequest  true  "Login payload"
// @Success      200      {object}  models.AuthResponse
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/login [post]
func (s *AuthService) Login(c *fiber.Ctx) error {
	req := c.Locals("body").(*models.LoginRequest)

	user, err := s.db.Queries.GetUserByEmail(c.Context(), req.Email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email or password"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email or password"})
	}

	resp, err := s.issueTokens(c, user.ID)
	if err != nil {
		return core.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to issue tokens", err)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// Refresh exchanges a refresh token for a new access token.
//
// @Summary      Refresh token
// @Description  Uses a valid refresh token to obtain a new access + refresh token pair
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.RefreshRequest  true  "Refresh payload"
// @Success      200      {object}  models.AuthResponse
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/refresh [post]
func (s *AuthService) Refresh(c *fiber.Ctx) error {
	req := c.Locals("body").(*models.RefreshRequest)

	tokenRecord, err := s.db.Queries.GetUserTokenByRefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired refresh token"})
	}

	if tokenRecord.RefreshTokenExpiresAt.Valid && time.Now().After(tokenRecord.RefreshTokenExpiresAt.Time) {
		_ = s.db.Queries.DeactivateToken(c.Context(), tokenRecord.AccessToken)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Refresh token has expired"})
	}

	// Rotate: deactivate old token, issue a new pair
	_ = s.db.Queries.DeactivateToken(c.Context(), tokenRecord.AccessToken)

	resp, err := s.issueTokens(c, tokenRecord.UserID)
	if err != nil {
		return core.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to issue tokens", err)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// Logout deactivates all tokens for the current user.
//
// @Summary      Logout
// @Description  Invalidates the current session token
// @Tags         auth
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Router       /auth/logout [post]
func (s *AuthService) Logout(c *fiber.Ctx) error {
	userID := c.Locals("userID").(int32)
	_ = s.db.Queries.DeactivateAllUserTokens(c.Context(), userID)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Logged out successfully"})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func (s *AuthService) issueTokens(c *fiber.Ctx, userID int32) (*models.AuthResponse, error) {
	accessToken, err := generateToken()
	if err != nil {
		return nil, err
	}
	refreshToken, err := generateToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_, err = s.db.Queries.CreateUserToken(c.Context(), db.CreateUserTokenParams{
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccessTokenExpiresAt: pgtype.Timestamptz{
			Time:  now.Add(accessTokenTTL),
			Valid: true,
		},
		RefreshTokenExpiresAt: pgtype.Timestamptz{
			Time:  now.Add(refreshTokenTTL),
			Valid: true,
		},
	})
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil
}

// generateToken creates a cryptographically random base64-URL token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
