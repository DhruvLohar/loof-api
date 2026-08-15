package users

import (
	"encoding/json"
	"fmt"
	"loof/internal/database"
	"loof/internal/shared"
	"loof/internal/storage"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/lib/pq"
)

// --- Auth Handlers ---

// staticOTP is the fixed OTP accepted during development.
// TODO: replace with a generated OTP once whatsapp integration is complete
const staticOTP = 123456

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

	if err := UpdateOTP(req.UID, staticOTP); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to save OTP",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "OTP sent successfully",
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
	if req.OTP != staticOTP {
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

// UpdateProfile applies a multipart/form-data profile edit.
// Text fields (name, username, gender, dob, country) are optional and only
// updated when present; interests is a text[] column sent as repeated fields
// or a JSON array and replaces the stored list; profile_picture (single file)
// and cover_images (one or more files) are uploaded to S3 and stored as URLs.
func UpdateProfile(c fiber.Ctx) error {
	authUserID, err := shared.GetAuthenticatedUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "unauthorized",
		})
	}

	updates := map[string]any{}

	if name := strings.TrimSpace(c.FormValue("name")); name != "" {
		updates["name"] = name
	}

	if username := strings.TrimSpace(c.FormValue("username")); username != "" {
		var count int64
		if err := database.DB.Db.Model(&User{}).
			Where("username = ? AND id <> ?", username, authUserID).
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
		updates["username"] = username
	}

	if gender := strings.TrimSpace(c.FormValue("gender")); gender != "" {
		updates["gender"] = gender
	}

	if country := strings.TrimSpace(c.FormValue("country")); country != "" {
		updates["country"] = country
	}

	if dob := strings.TrimSpace(c.FormValue("dob")); dob != "" {
		parsedDOB, err := time.Parse("2006-01-02", dob)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "invalid dob format, expected YYYY-MM-DD",
			})
		}
		updates["dob"] = parsedDOB
	}

	if fileHeader, err := c.FormFile("profile_picture"); err == nil && fileHeader != nil {
		url, err := storage.UploadImage(c.Context(), fileHeader, fmt.Sprintf("users/%d/profile-picture", authUserID))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})
		}
		updates["profile_picture"] = url
	}

	if form, err := c.MultipartForm(); err == nil && form != nil {
		if values, ok := shared.MultipartValues(form, "interests"); ok {
			interests, err := shared.ParseStringArrayField(values)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"success": false,
					"message": "invalid interests, expected repeated text fields or a JSON array of strings",
				})
			}
			updates["interests"] = interests
		}

		if files := form.File["cover_images"]; len(files) > 0 {
			coverImageURLs := make([]string, 0, len(files))
			for _, fileHeader := range files {
				url, err := storage.UploadImage(c.Context(), fileHeader, fmt.Sprintf("users/%d/cover-images", authUserID))
				if err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
						"success": false,
						"message": err.Error(),
					})
				}
				coverImageURLs = append(coverImageURLs, url)
			}
			updates["cover_images"] = pq.StringArray(coverImageURLs)
		}
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "no fields to update",
		})
	}

	if err := database.DB.Db.Model(&User{}).Where("id = ?", authUserID).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "failed to update profile",
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
		"message": "profile updated successfully",
		"data":    SerializeUserProfile(user),
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
