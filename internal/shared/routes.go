package shared

import "github.com/gofiber/fiber/v3"

type RouteRegistrar func(fiber.Router, fiber.Handler)

func RegisterRoutes(
	app *fiber.App,
	registrars ...RouteRegistrar,
) {
	v1 := app.Group("/v1")
	protected := ProtectedRoutes()

	for _, registrar := range registrars {
		registrar(v1, protected)
	}
}
