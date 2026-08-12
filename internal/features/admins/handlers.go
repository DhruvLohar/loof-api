package admins

import (
	"errors"
	"log"
	"loof/internal/database"
	"loof/internal/features/users"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AdminListItem struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateUserRequest struct {
	Name           *string `json:"name"`
	Username       *string `json:"username"`
	Gender         *string `json:"gender"`
	DOB            *string `json:"dob"`
	Country        *string `json:"country"`
	IsActive       *bool   `json:"is_active"`
	ProfilePicture *string `json:"profile_picture"`
}

func Login(c fiber.Ctx) error {
	var req AdminLoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "invalid request body",
		})
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	password := strings.TrimSpace(req.Password)
	if email == "" || password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "email and password are required",
		})
	}

	var admin AdminUser
	err := database.DB.Db.Where("email = ?", email).First(&admin).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "invalid email or password",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to login",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "invalid email or password",
		})
	}

	accessToken, err := GenerateAdminAccessToken(c, admin)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to generate admin access token",
		})
	}

	if err := database.DB.Db.Model(&admin).Where("id = ?", admin.ID).Update("access_token", accessToken).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to complete admin login",
		})
	}

	sess := session.FromContext(c)
	sess.Set("lsess", accessToken)
	log.Printf("[session] set: session_id=%s lsess=%s admin_id=%d", sess.ID(), accessToken, admin.ID)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "admin logged in successfully",
		"data": fiber.Map{
			"id":    admin.ID,
			"name":  admin.Name,
			"email": admin.Email,
			"roles": admin.Roles,
		},
	})
}

func GetAllAdmins(c fiber.Ctx) error {
	var admins []AdminUser
	if err := database.DB.Db.Order("created_at DESC").Find(&admins).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to fetch admins",
		})
	}

	result := make([]AdminListItem, 0, len(admins))
	for _, admin := range admins {
		result = append(result, AdminListItem{
			ID:        admin.ID,
			Name:      admin.Name,
			Email:     admin.Email,
			Roles:     admin.Roles,
			CreatedAt: admin.CreatedAt,
			UpdatedAt: admin.UpdatedAt,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

func GetAllUsers(c fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.Query("page_size", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	search := strings.TrimSpace(c.Query("search"))

	appUsers, total, err := users.ListUsers(users.ListUsersParams{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
		Search:   search,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to fetch users",
		})
	}

	results := make([]fiber.Map, 0, len(appUsers))
	for i := range appUsers {
		results = append(results, users.SerializeUserListItem(&appUsers[i]))
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"results":      results,
			"page":         page,
			"page_size":    pageSize,
			"total":        total,
			"total_pages":  totalPages,
			"has_next":     page < totalPages,
			"has_previous": page > 1,
		},
	})
}

// GetDashboardStats returns aggregate counts shown on the admin dashboard
func GetDashboardStats(c fiber.Ctx) error {
	var totalUsers int64
	if err := database.DB.Db.Model(&users.User{}).Count(&totalUsers).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to fetch dashboard stats",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"total_users": totalUsers,
		},
	})
}

// GetUserByID returns full profile details for a single user, for the admin user-detail view
func GetUserByID(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "invalid user id",
		})
	}

	user, err := users.GetUser(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "user not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    users.SerializeUserProfile(user),
	})
}

// CheckUsername lets an admin check whether a username is available before
// saving edits. exclude_id should be set to the user currently being edited
// so their own existing username doesn't count as taken.
func CheckUsername(c fiber.Ctx) error {
	username := strings.TrimSpace(c.Query("username"))
	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "username is required",
		})
	}

	excludeID, err := strconv.Atoi(c.Query("exclude_id", "0"))
	if err != nil || excludeID < 0 {
		excludeID = 0
	}

	var count int64
	if err := database.DB.Db.Model(&users.User{}).
		Where("username = ? AND id <> ?", username, excludeID).
		Count(&count).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to validate username",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"available": count == 0,
		},
	})
}

// UpdateUser lets an admin edit a user's basic profile details
func UpdateUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "invalid user id",
		})
	}

	if _, err := users.GetUser(uint(id)); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "user not found",
		})
	}

	var req UpdateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "invalid request body",
		})
	}

	updates := map[string]any{}

	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}

	if req.Username != nil {
		username := strings.TrimSpace(*req.Username)
		if username != "" {
			var count int64
			if err := database.DB.Db.Model(&users.User{}).
				Where("username = ? AND id <> ?", username, id).
				Count(&count).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": "failed to validate username",
				})
			}
			if count > 0 {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"success": false,
					"message": "username already exists",
				})
			}
		}
		updates["username"] = username
	}

	if req.Gender != nil {
		updates["gender"] = *req.Gender
	}

	if req.Country != nil {
		updates["country"] = *req.Country
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if req.ProfilePicture != nil {
		updates["profile_picture"] = *req.ProfilePicture
	}

	if req.DOB != nil {
		if strings.TrimSpace(*req.DOB) == "" {
			updates["dob"] = nil
		} else {
			parsedDOB, err := time.Parse("2006-01-02", *req.DOB)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"success": false,
					"message": "invalid dob format, expected YYYY-MM-DD",
				})
			}
			updates["dob"] = parsedDOB
		}
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "no valid fields to update",
		})
	}

	updatedUser, err := users.UpdateBasicDetails(uint(id), updates)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to update user",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "user updated successfully",
		"data":    users.SerializeUserProfile(updatedUser),
	})
}
