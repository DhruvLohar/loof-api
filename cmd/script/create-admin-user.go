package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"loof/internal/database"
	"loof/internal/features/admins"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	prod := flag.Bool("prod", false, "create the admin user on the production database (loads .env.prod)")
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)

	if *prod {
		// database.ConnectDatabase() reads vars via config.GetEnv, which loads
		// ".env" but never overrides vars already set in the environment — so
		// loading .env.prod here first makes it win over the local .env.
		if err := godotenv.Load(".env.prod"); err != nil {
			log.Fatalf("Error loading .env.prod: %v", err)
		}

		fmt.Println("WARNING: this will create an admin user on PRODUCTION.")
		fmt.Print("Type 'yes' to continue: ")
		confirm, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading confirmation: %v", err)
		}
		if strings.TrimSpace(confirm) != "yes" {
			log.Fatal("Aborted.")
		}
	}

	// Initialize Database
	database.ConnectDatabase()
	log.Println("Database connection established successfully.")

	// Read Name
	fmt.Print("Enter Admin Name: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading name: %v", err)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		log.Fatal("Name cannot be empty")
	}

	// Read Email
	fmt.Print("Enter Admin Email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading email: %v", err)
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		log.Fatal("Email cannot be empty")
	}

	// Read Password
	fmt.Print("Enter Admin Password: ")
	password, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading password: %v", err)
	}
	password = strings.TrimSpace(password)
	if password == "" {
		log.Fatal("Password cannot be empty")
	}

	// Read Roles
	fmt.Printf("Enter Admin Roles (comma-separated, allowed: admin, user) [default: admin]: ")
	rolesInput, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Error reading roles: %v", err)
	}
	rolesInput = strings.TrimSpace(rolesInput)

	var roles []string
	if rolesInput == "" {
		roles = []string{"admin"}
	} else {
		parts := strings.Split(rolesInput, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				roles = append(roles, trimmed)
			}
		}
	}

	// Validate roles before proceeding
	if err := admins.ValidateRoles(roles); err != nil {
		log.Fatalf("Invalid roles specified: %v", err)
	}

	// Check if admin already exists
	var count int64
	err = database.DB.Db.Model(&admins.AdminUser{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		log.Fatalf("Error checking existing admin user: %v", err)
	}
	if count > 0 {
		log.Fatalf("Admin with email %s already exists", email)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Error hashing password: %v", err)
	}

	// Create Admin User
	admin := admins.AdminUser{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
		Roles:    pq.StringArray(roles),
	}

	err = database.DB.Db.Create(&admin).Error
	if err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	fmt.Printf("\nSuccess! Admin User created successfully:\n")
	fmt.Printf("ID:    %d\n", admin.ID)
	fmt.Printf("Name:  %s\n", admin.Name)
	fmt.Printf("Email: %s\n", admin.Email)
	fmt.Printf("Roles: %s\n", strings.Join(admin.Roles, ", "))
}
