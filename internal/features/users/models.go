package users

import (
	"errors"
	"loof/internal/database"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `json:"name"`
	Username    *string `gorm:"uniqueIndex;type:varchar(30)" json:"username"`
	CountryCode string `gorm:"type:varchar(10);not null" json:"country_code"`
	PhoneNumber string `gorm:"uniqueIndex;type:varchar(20);not null" json:"phone_number"`
	Gender      string `gorm:"type:varchar(10)" json:"gender"`

	DOB            *time.Time `gorm:"type:date" json:"dob"`
	Country        string     `gorm:"type:varchar(100)" json:"country"`
	ProfilePicture string     `json:"profile_picture"`

	Interests   pq.StringArray `gorm:"type:text[]" json:"interests"`
	Preferences datatypes.JSON `gorm:"type:jsonb" json:"preferences"`

	AccessToken    string     `json:"access_token"`
	OTPGenerated   int        `json:"otp_generated"`
	OTPGeneratedAt *time.Time `json:"otp_generated_at"`

	IsActive    bool       `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

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

// ListUsersParams defines filtering and pagination options for admin user listing
type ListUsersParams struct {
	Page     int
	PageSize int
	Status   string // "active", "inactive", or "" for all
	Search   string // matches id, name, username or phone_number
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
