package admins

import (
	"errors"
	"log"

	"loof/internal/config"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	"github.com/golang-jwt/jwt/v5"
)

func GetAuthenticatedAdminID(c fiber.Ctx) (uint, error) {
	sess := session.FromContext(c)
	token := sess.Get("lsess")
	log.Printf("[session] lookup: session_id=%s lsess=%v", sess.ID(), token)
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
		log.Printf("[session] session_id=%s lsess parse failed: %v", sess.ID(), err)
		return 0, errors.New("invalid token")
	}

	adminID, ok := claims["admin_id"].(float64)
	if !ok {
		log.Printf("[session] session_id=%s lsess missing admin_id claim: %v", sess.ID(), claims)
		return 0, errors.New("invalid token claims")
	}

	log.Printf("[session] session_id=%s resolved admin_id=%d", sess.ID(), uint(adminID))
	return uint(adminID), nil
}

func ProtectedRoutes() fiber.Handler {
	return func(c fiber.Ctx) error {
		_, err := GetAuthenticatedAdminID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "unauthorized",
			})
		}
		return c.Next()
	}
}
