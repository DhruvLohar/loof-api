package admins

import (
	"github.com/gofiber/fiber/v3"
)

func RegisterAdminRoutes(router fiber.Router, _ fiber.Handler) {
	adminGroup := router.Group("/admins")
	adminProtected := ProtectedRoutes()

	adminGroup.Post("/login", Login)
	adminGroup.Get("", adminProtected, GetAllAdmins)
	adminGroup.Get("/dashboard/stats", adminProtected, GetDashboardStats)
	adminGroup.Get("/users", adminProtected, GetAllUsers)
	adminGroup.Get("/users/check-username", adminProtected, CheckUsername)
	adminGroup.Get("/users/:id", adminProtected, GetUserByID)
	adminGroup.Patch("/users/:id", adminProtected, UpdateUser)
}
