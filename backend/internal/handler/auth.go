package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"comp-video-service/backend/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type adminAuthRepository interface {
	GetByUsername(ctx context.Context, username string) (*model.Admin, error)
}

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	adminRepo adminAuthRepository
	jwtSecret string
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(adminRepo adminAuthRepository, jwtSecret string) *AuthHandler {
	return &AuthHandler{adminRepo: adminRepo, jwtSecret: jwtSecret}
}

// Login godoc
// @Summary      Admin login
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      model.LoginRequest   true  "Credentials"
// @Success      200   {object}  model.LoginResponse
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Router       /admin/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	admin, err := h.adminRepo.GetByUsername(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	csrfToken, err := generateCSRFToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate csrf token"})
		return
	}

	token, err := generateJWT(admin.ID.String(), csrfToken, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, model.LoginResponse{
		Token:     token,
		CSRFToken: csrfToken,
		Admin:     admin,
	})
}

func generateJWT(adminID, csrfToken, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  adminID,
		"csrf": csrfToken,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func generateCSRFToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
