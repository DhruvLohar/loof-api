package users

import "github.com/gofiber/fiber/v3"

func RegisterUserRoutes(router fiber.Router, protectedMiddleware fiber.Handler) {
	usersGroup := router.Group("/users")

	// Auth Routes
	usersGroup.Post("/signup-signin", SignUpSignIn)
	usersGroup.Post("/resend-otp", SendOTP)
	usersGroup.Post("/verify-otp", VerifyOTP)

	// Profile Routes
	usersGroup.Post("/validate-username", protectedMiddleware, RejectDeletedUser, ValidateUsername)
	usersGroup.Post("/preferences", protectedMiddleware, RejectDeletedUser, UpdatePreferences)
	usersGroup.Get("/profile", protectedMiddleware, RejectDeletedUser, GetProfile)
	usersGroup.Post("/profile", protectedMiddleware, RejectDeletedUser, UpdateProfile)
	usersGroup.Post("/delete", protectedMiddleware, RejectDeletedUser, DeleteAccount)
}
