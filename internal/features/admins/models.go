package admins

import (
	"errors"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Allowed roles in your application backend.
var AllowedRoles = map[string]bool{
	"admin": true,
	"user":  true,
}

type AdminUser struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"not null" json:"name"`
	Email       string         `gorm:"uniqueIndex;type:varchar(255);not null" json:"email"`
	Password    string         `gorm:"not null" json:"-"`
	AccessToken string         `json:"access_token"`
	Roles       pq.StringArray `gorm:"type:text[]" json:"roles"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

// ValidateRoles checks if all passed roles exist in AllowedRoles.
func ValidateRoles(roles []string) error {
	if len(roles) == 0 {
		return errors.New("at least one role is required")
	}

	for _, role := range roles {
		if !AllowedRoles[role] {
			return errors.New("invalid role: " + role)
		}
	}

	return nil
}

// BeforeSave validates roles before saving to the database.
func (a *AdminUser) BeforeSave(tx *gorm.DB) error {
	return ValidateRoles(a.Roles)
}
