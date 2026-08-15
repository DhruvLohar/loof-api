package shared

import (
	"errors"
	"strings"

	"loof/internal/config"

	"github.com/gofiber/fiber/v3"

	"github.com/golang-jwt/jwt/v5"
)

func GetAuthenticatedUserID(c fiber.Ctx) (uint, error) {
	authHeader := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
	if authHeader == "" {
		return 0, errors.New("unauthorized")
	}

	scheme, tokenValue, found := strings.Cut(authHeader, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return 0, errors.New("unauthorized")
	}

	tokenValue = strings.TrimSpace(tokenValue)
	if tokenValue == "" {
		return 0, errors.New("unauthorized")
	}

	secretKey := config.GetEnv("SECRET_KEY")
	if secretKey == "" {
		return 0, errors.New("server misconfiguration")
	}

	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenValue, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil {
		return 0, errors.New("invalid token")
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("invalid token claims")
	}

	return uint(userID), nil
}

func ProtectedRoutes() fiber.Handler {
	return func(c fiber.Ctx) error {
		_, err := GetAuthenticatedUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "unauthorized",
			})
		}
		return c.Next()
	}
}
