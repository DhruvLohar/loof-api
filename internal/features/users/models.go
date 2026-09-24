package users

import (
	"crypto/rand"
	"errors"
	"fmt"
	"loof/internal/database"
	"math/big"
	"strings"
	"time"

	"gorm.io/gorm"
)

// GetUser fetches a user by primary key ID
func GetUser(id uint) (*User, error) {
	var user User
	if err := database.DB.Db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindOrCreateUser finds an existing user by phone or creates a new account
func FindOrCreateUser(countryCode, phoneNumber string) (*User, error) {
	var user User

	err := database.DB.Db.Where("country_code = ? AND phone_number = ?", countryCode, phoneNumber).First(&user).Error
	if err == nil {
		return &user, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Create new user if not found
	user = User{
		CountryCode: countryCode,
		PhoneNumber: phoneNumber,
		IsActive:    true,
	}

	if err := database.DB.Db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateOTP saves the generated OTP and timestamp
func UpdateOTP(userID uint, otp int) error {
	now := time.Now()
	return database.DB.Db.Model(&User{}).Where("id = ?", userID).Updates(map[string]any{
		"otp_generated":    otp,
		"otp_generated_at": now,
	}).Error
}

// ListUsers returns a paginated, filtered set of users along with the total matching count
func ListUsers(params ListUsersParams) ([]User, int64, error) {
	query := database.DB.Db.Model(&User{})

	switch params.Status {
	case "active":
		query = query.Where("is_active = ?", true)
	case "inactive":
		query = query.Where("is_active = ?", false)
	}

	if search := strings.TrimSpace(params.Search); search != "" {
		like := "%" + search + "%"
		query = query.Where(
			"CAST(id AS TEXT) ILIKE ? OR name ILIKE ? OR username ILIKE ? OR phone_number ILIKE ?",
			like, like, like, like,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var usersList []User
	offset := (params.Page - 1) * params.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(params.PageSize).Find(&usersList).Error; err != nil {
		return nil, 0, err
	}

	return usersList, total, nil
}

// updatableBasicFields whitelists the columns an admin is allowed to update directly
var updatableBasicFields = map[string]bool{
	"name":            true,
	"username":        true,
	"gender":          true,
	"dob":             true,
	"country":         true,
	"is_active":       true,
	"profile_picture": true,
}

// UpdateBasicDetails applies a whitelisted set of field updates to a user and returns the refreshed record
func UpdateBasicDetails(id uint, updates map[string]any) (*User, error) {
	filtered := make(map[string]any, len(updates))
	for key, value := range updates {
		if updatableBasicFields[key] {
			filtered[key] = value
		}
	}

	if len(filtered) > 0 {
		if err := database.DB.Db.Model(&User{}).Where("id = ?", id).Updates(filtered).Error; err != nil {
			return nil, err
		}
	}

	return GetUser(id)
}

// withRandomSuffix appends _<random> to base, truncating base so the result fits maxLen
func withRandomSuffix(base string, maxLen int) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	suffix := fmt.Sprintf("_%06d", n.Int64())
	if len(base)+len(suffix) > maxLen {
		base = base[:maxLen-len(suffix)]
	}
	return base + suffix, nil
}

// SoftDeleteUser frees the unique username and phone number for reuse, deactivates the user and revokes their session
func SoftDeleteUser(id uint) error {
	user, err := GetUser(id)
	if err != nil {
		return err
	}

	phone, err := withRandomSuffix(user.PhoneNumber, 20)
	if err != nil {
		return err
	}

	updates := map[string]any{
		"phone_number":     phone,
		"is_active":        false,
		"access_token":     "",
		"otp_generated":    0,
		"otp_generated_at": nil,
		"deleted_at":       time.Now(),
	}
	if user.Username != nil {
		username, err := withRandomSuffix(*user.Username, 30)
		if err != nil {
			return err
		}
		updates["username"] = username
	}

	return database.DB.Db.Model(&User{}).Where("id = ?", id).Updates(updates).Error
}
