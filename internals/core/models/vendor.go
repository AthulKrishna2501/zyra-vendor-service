package models

import (
	"time"

	"github.com/AthulKrishna2501/zyra-auth-service/internals/core/models"
	"github.com/google/uuid"
)

type Category struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CategoryID   string    `json:"category_id" gorm:"type:varchar(255);unique;not null"`
	CategoryName string    `json:"category_name" gorm:"type:varchar(255);"`
	Title        string    `json:"title" gorm:"type:varchar(255);not null"`
	Image        string    `json:"image" gorm:"type:varchar(255)"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type CategoryRequest struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VendorID   string    `json:"vendor_id" gorm:"type:varchar(255);not null"`
	CategoryID string    `json:"category_id" gorm:"type:varchar(255);not null"`
	Status     string    `json:"status" gorm:"type:varchar(20);default:'pending'"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type VendorCategory struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VendorID   uuid.UUID `gorm:"type:uuid;not null"`
	CategoryID string    `gorm:"type:varchar(255);not null"`

	User     models.User `gorm:"foreignKey:VendorID;references:UserID;constraint:OnDelete:CASCADE"`
	Category Category    `gorm:"foreignKey:CategoryID;references:ID;constraint:OnDelete:CASCADE"`
}
