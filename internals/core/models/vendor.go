package models

import (
	"time"

	"github.com/AthulKrishna2501/zyra-auth-service/internals/core/models"
	"github.com/google/uuid"
)

type Category struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CategoryID   uuid.UUID `json:"category_id" gorm:"type:uuid;default:gen_random_uuid()"`
	CategoryName string    `json:"category_name" gorm:"type:varchar(255);"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type CategoryRequest struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VendorID   uuid.UUID `json:"vendor_id" gorm:"type:uuid;not null"`
	CategoryID uuid.UUID `json:"category_id" gorm:"type:uuid;not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type Service struct {
	ID                  uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VendorID            uuid.UUID `json:"vendor_id" gorm:"type:uuid;not null;index"`
	ServiceTitle        string    `json:"service_title" gorm:"type:varchar(255);not null"`
	YearOfExperience    int       `json:"year_of_experience" gorm:"not null"`
	AvailableDate       time.Time `json:"available_date" gorm:"type:timestamptz;not null"`
	ServiceDescription  string    `json:"service_description" gorm:"type:text;not null"`
	CancellationPolicy  string    `json:"cancellation_policy" gorm:"type:text"`
	TermsAndConditions  string    `json:"terms_and_conditions" gorm:"type:text"`
	ServiceDuration     int       `json:"service_duration" gorm:"not null"`
	ServicePrice        int       `json:"service_price" gorm:"not null"`
	AdditionalHourPrice *int      `json:"additional_hour_price" gorm:""`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Vendor models.User `gorm:"foreignKey:VendorID;references:ID;constraint:OnDelete:CASCADE"`
}

type VendorCategory struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VendorID   uuid.UUID `json:"vendor_id" gorm:"type:uuid;not null"`
	CategoryID uuid.UUID `json:"category_id" gorm:"type:uuid;not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
