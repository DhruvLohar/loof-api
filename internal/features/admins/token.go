package admins

import (
	"time"

	"loof/internal/config"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func GenerateAdminAccessToken(c fiber.Ctx, admin AdminUser) (string, error) {
	secretKey := config.GetEnv("SECRET_KEY")
	if secretKey == "" {
		return "", fiber.NewError(fiber.StatusInternalServerError, "failed to generate admin access token")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"admin_id": admin.ID,
		"sub":      admin.ID,
		"roles":    admin.Roles,
		"exp":      now.Add(24 * time.Hour).Unix(),
		"iat":      now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return accessToken, nil
}
