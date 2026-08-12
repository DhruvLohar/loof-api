package config

import (
	"os"

	"github.com/joho/godotenv"
)

func GetEnv(key string) string {
	// Try to load the .env file.
	// We ignore the error using "_" because in Docker or Production,
	// the .env file won't exist, and variables are injected directly into the OS.
	_ = godotenv.Load()

	// initialization and condition
	// same as if exists {} if value, exists := os.LookupEnv(key) defined before
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return ""
}
