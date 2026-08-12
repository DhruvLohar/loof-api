package shared

import (
	"errors"

	"loof/internal/config"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	"github.com/golang-jwt/jwt/v5"
)

func GetAuthenticatedUserID(c fiber.Ctx) (uint, error) {
	sess := session.FromContext(c)
	token := sess.Get("__loof_session")
	if token == nil {
		return 0, errors.New("unauthorized")
	}

	secretKey := config.GetEnv("SECRET_KEY")
	if secretKey == "" {
		return 0, errors.New("server misconfiguration")
	}

	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token.(string), claims, func(token *jwt.Token) (interface{}, error) {
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
