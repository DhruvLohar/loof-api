package main

import (
	"fmt"
	"log"

	"loof/internal/config"
	"loof/internal/database"
	"loof/internal/features/admins"
	"loof/internal/features/deploy"
	"loof/internal/features/users"
	"loof/internal/shared"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/session"
)

func healthCheck(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "healthy"})
}

func main() {
	database.ConnectDatabase()
	log.Println("DB Connected")
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://127.0.0.1:3000",
			"http://localhost:3000",
			"https://letsloof.com",
			"https://*.letsloof.com",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	app.Use(session.New())

	app.Get("/health", healthCheck)
	shared.RegisterRoutes(app,
		users.RegisterUserRoutes,
		admins.RegisterAdminRoutes,
		deploy.RegisterDeployRoutes,
	)

	port := config.GetEnv("PORT")
	if port == "" {
		port = "8000"
	}

	err := app.Listen(":" + port)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}
