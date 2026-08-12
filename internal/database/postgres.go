package database

import (
	"fmt"
	"log"
	"loof/internal/config"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DBInstance struct {
	Db *gorm.DB
}

// global DBInstance
var DB DBInstance

func ConnectDatabase() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Kolkata",
		config.GetEnv("DB_HOST"),
		config.GetEnv("DB_USER"),
		config.GetEnv("DB_PASS"),
		config.GetEnv("DB_NAME"),
		config.GetEnv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Println("Error connecting to database:", err)
		os.Exit(2)
	}

	DB = DBInstance{Db: db}

	// Run migrations
}
