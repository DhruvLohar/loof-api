package users

import (
	"crypto/rand"
	"encoding/json"
	"loof/internal/database"
	"loof/internal/shared"
	"math/big"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

// Request Payload Structs
type SignUpSignInRequest struct {
	CountryCode string `json:"country_code"`
	PhoneNumber string `json:"phone_number"`
}

type SendOTPRequest struct {
	UID uint `json:"uid"`
}

type VerifyOTPRequest struct {
	UID uint `json:"uid"`
	OTP int  `json:"otp"`
}

type ValidateUsernameRequest struct {
	Username string `json:"username"`
}

// --- Auth Handlers ---

func SignUpSignIn(c fiber.Ctx) error {
	var req SignUpSignInRequest
	if err := c.Bind().Body(&req); err != nil || req.CountryCode == "" || req.PhoneNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "country_code and phone_number are required",
		})
	}

	user, err := FindOrCreateUser(req.CountryCode, req.PhoneNumber)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to process account",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":           user.ID,
			"country_code": user.CountryCode,
			"phone_number": user.PhoneNumber,
		},
	})
}

func SendOTP(c fiber.Ctx) error {
	var req SendOTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "invalid request body",
		})
	}

	if req.UID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "uid is required",
		})
	}

	// Generate secure 6-digit OTP (100000 - 999999)
	nBig, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to generate OTP",
		})
	}
	otp := int(nBig.Int64() + 100000)

	if err := UpdateOTP(req.UID, otp); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to save OTP",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "OTP sent successfully",
		"otp":     otp, // Remove in production!
	})
}

// VerifyOTP checks the provided OTP against the database record
func VerifyOTP(c fiber.Ctx) error {
	var req VerifyOTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "invalid request body",
		})
	}

	if req.UID == 0 || req.OTP == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "uid and otp are required",
		})
	}

	user, err := GetUser(req.UID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "user not found",
		})
	}

	// Verify OTP value
	if user.OTPGenerated == 0 || user.OTPGenerated != req.OTP {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid OTP",
		})
	}

	accessToken, err := GenerateUserAccessToken(c, user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to generate access token",
		})
	}

	now := time.Now()
	if err := database.DB.Db.Model(&User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"last_login_at": now,
		"access_token":  accessToken,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to complete login",
		})
	}

	sess := session.FromContext(c)
	sess.Set("__loof_session", accessToken)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "OTP verified successfully",
		"data": fiber.Map{
			"id":           user.ID,
			"phone_number": user.PhoneNumber,
			"country_code": user.CountryCode,
			"access_token": accessToken,
		},
	})
}

// --- Profile Handlers ---

func ValidateUsername(c fiber.Ctx) error {
	var req ValidateUsernameRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "invalid request body",
		})
	}

	username := strings.TrimSpace(req.Username)
	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "username is required",
		})
	}

	var count int64
	if err := database.DB.Db.Model(&User{}).Where("username = ?", username).Count(&count).Error; err != nil {
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

	return c.JSON(fiber.Map{
		"success": true,
		"message": "username is valid",
	})
}

func UpdatePreferences(c fiber.Ctx) error {
	authUserID, err := shared.GetAuthenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "unauthorized",
		})
	}

	var body map[string]any
	if err := c.Bind().Body(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "invalid request body",
		})
	}

	validPreferenceFields := []string{"send_notification"}
	validFieldSet := make(map[string]struct{}, len(validPreferenceFields))
	for _, field := range validPreferenceFields {
		validFieldSet[field] = struct{}{}
	}

	for key := range body {
		if _, ok := validFieldSet[key]; !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "invalid preference field: " + key,
			})
		}
	}

	user, err := GetUser(authUserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "user not found",
		})
	}

	mergedPreferences := map[string]any{}
	if len(user.Preferences) > 0 && string(user.Preferences) != "null" {
		if err := json.Unmarshal(user.Preferences, &mergedPreferences); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "failed to read existing preferences",
			})
		}
	}

	for key, value := range body {
		mergedPreferences[key] = value
	}

	updatedPreferences, err := json.Marshal(mergedPreferences)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to update preferences",
		})
	}

	if err := database.DB.Db.Model(&User{}).Where("id = ?", authUserID).Update("preferences", updatedPreferences).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to save preferences",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "preferences updated successfully",
		"data":    mergedPreferences,
	})
}

func GetProfile(c fiber.Ctx) error {
	authUserID, err := shared.GetAuthenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "unauthorized",
		})
	}

	user, err := GetUser(authUserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "user not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"user":    SerializeUserProfile(user),
	})
}
