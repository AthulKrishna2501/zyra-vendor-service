package responses

type VendorProfileResponse struct {
	UserID        string `json:"vendor_id" gorm:"column:user_id"`
	FirstName     string `json:"first_name" gorm:"column:first_name"`
	LastName      string `json:"last_name" gorm:"column:last_name"`
	Email         string `json:"email" gorm:"column:email"`
	ProfileImage  string `json:"profile_image" gorm:"column:profile_image"`
	PhoneNumber   string `json:"phone_number" gorm:"column:phone"`
	Category      string `json:"category"`
	RequestStatus string `json:"status" gorm:"column:status"`
}
