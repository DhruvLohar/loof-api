package users

import "github.com/gofiber/fiber/v3"

var profileResponseFields = []string{
	"id",
	"name",
	"username",
	"country_code",
	"phone_number",
	"gender",
	"dob",
	"country",
	"profile_picture",
	"interests",
	"cover_images",
	"preferences",
	"is_active",
	"created_at",
	"updated_at",
	"last_login_at",
}

func SerializeUserProfile(user *User) fiber.Map {
	data := fiber.Map{}

	for _, field := range profileResponseFields {
		switch field {
		case "id":
			data["id"] = user.ID
		case "name":
			data["name"] = user.Name
		case "username":
			data["username"] = user.Username
		case "country_code":
			data["country_code"] = user.CountryCode
		case "phone_number":
			data["phone_number"] = user.PhoneNumber
		case "gender":
			data["gender"] = user.Gender
		case "dob":
			data["dob"] = user.DOB
		case "country":
			data["country"] = user.Country
		case "profile_picture":
			data["profile_picture"] = user.ProfilePicture
		case "interests":
			data["interests"] = user.Interests
		case "cover_images":
			data["cover_images"] = user.CoverImages
		case "preferences":
			data["preferences"] = user.Preferences
		case "is_active":
			data["is_active"] = user.IsActive
		case "created_at":
			data["created_at"] = user.CreatedAt
		case "updated_at":
			data["updated_at"] = user.UpdatedAt
		case "last_login_at":
			data["last_login_at"] = user.LastLoginAt
		}
	}

	return data
}

// SerializeUserListItem returns the lightweight row shape used in admin user listings
func SerializeUserListItem(user *User) fiber.Map {
	return fiber.Map{
		"id":            user.ID,
		"name":          user.Name,
		"username":      user.Username,
		"country_code":  user.CountryCode,
		"phone_number":  user.PhoneNumber,
		"gender":        user.Gender,
		"is_active":     user.IsActive,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"last_login_at": user.LastLoginAt,
	}
}
