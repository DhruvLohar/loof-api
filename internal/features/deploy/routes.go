package deploy

import (
	"github.com/gofiber/fiber/v3"
)

func RegisterDeployRoutes(router fiber.Router, _ fiber.Handler) {
	deployGroup := router.Group("/deploy")
	deployer := NewDeployer(LoadConfig())

	// Authenticated by the GitHub HMAC signature, not by the session middleware.
	deployGroup.Post("/github", HandleGithubWebhook(deployer))
	deployGroup.Get("/status", HandleStatus(deployer))
}
