package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"` // UUID primary key
	CategoryID   string    `json:"category_id" gorm:"type:varchar;unique;not null"`          // Unique category ID (Fixed typo in `varchar`)
	CategoryName string    `json:"category_name" gorm:"type:varchar(255);"`                  // Optional category name
	Title        string    `json:"title" gorm:"type:varchar(255);not null"`                  // Required title
	Image        string    `json:"image" gorm:"type:varchar(255)"`                           // Image URL (optional)
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`                         // Auto timestamp
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`                         // Auto update timestamp
}

type CategoryRequest struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"` // UUID primary key
	VendorID   string `json:"vendor_id" gorm:"type:varchar(255);not null"`                      // UUID for vendor (Fixed)
	CategoryID string `json:"category_id" gorm:"type:varchar(255);not null"`                    // UUID for category (Fixed JSON tag & type)
	Status     string    `json:"status" gorm:"type:varchar(20);default:'pending'"`         // Status with default value
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`                         // Auto timestamp
}
