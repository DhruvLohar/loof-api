package users

import "github.com/gofiber/fiber/v3"

func RegisterUserRoutes(router fiber.Router, protectedMiddleware fiber.Handler) {
	usersGroup := router.Group("/users")

	// Auth Routes
	usersGroup.Post("/signup-signin", SignUpSignIn)
	usersGroup.Post("/send-otp", SendOTP)
	usersGroup.Post("/verify-otp", VerifyOTP)

	// Profile Routes
	usersGroup.Post("/validate-username", protectedMiddleware, ValidateUsername)
	usersGroup.Post("/preferences", protectedMiddleware, UpdatePreferences)
	usersGroup.Get("/profile", protectedMiddleware, GetProfile)
	usersGroup.Post("/profile", protectedMiddleware, UpdateProfile)
}
