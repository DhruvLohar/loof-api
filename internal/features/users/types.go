package users

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
)

type User struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	Name        string  `json:"name"`
	Username    *string `gorm:"uniqueIndex;type:varchar(30)" json:"username"`
	CountryCode string  `gorm:"type:varchar(10);not null" json:"country_code"`
	PhoneNumber string  `gorm:"uniqueIndex;type:varchar(20);not null" json:"phone_number"`
	Gender      string  `gorm:"type:varchar(10)" json:"gender"`

	DOB            *time.Time `gorm:"type:date" json:"dob"`
	Country        string     `gorm:"type:varchar(100)" json:"country"`
	ProfilePicture string     `json:"profile_picture"`

	Interests   pq.StringArray `gorm:"type:text[]" json:"interests"`
	CoverImages pq.StringArray `gorm:"type:text[]" json:"cover_images"`
	Preferences datatypes.JSON `gorm:"type:jsonb" json:"preferences"`

	AccessToken    string     `json:"access_token"`
	OTPGenerated   int        `json:"otp_generated"`
	OTPGeneratedAt *time.Time `json:"otp_generated_at"`

	IsActive    bool       `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

// ListUsersParams defines filtering and pagination options for admin user listing
type ListUsersParams struct {
	Page     int
	PageSize int
	Status   string // "active", "inactive", or "" for all
	Search   string // matches id, name, username or phone_number
}

// Request Payload Structs

type SignUpSignInRequest struct {
	CountryCode string `json:"country_code"`
	PhoneNumber string `json:"phone_number"`
}

type SendOTPRequest struct {
	ID uint `json:"id"`
}

type VerifyOTPRequest struct {
	ID  uint `json:"id"`
	OTP int  `json:"otp"`
}

type ValidateUsernameRequest struct {
	Username string `json:"username"`
}
