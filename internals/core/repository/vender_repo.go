package repository

import (
	"context"
	"errors"
	"log"

	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/models"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/models/responses"
	"gorm.io/gorm"
)

type VendorStorage struct {
	DB *gorm.DB
}

type VendorRepository interface {
	RequestCategory(ctx context.Context, vendorID, categoryId string) error
	CategoryExists(ctx context.Context, categoryId string) (bool, error)
	HasRequestedCategory(ctx context.Context, vendorID string) (bool, error)
	ListCategories(ctx context.Context) ([]models.Category, error)
	UpdateCategoryRequestStatus(ctx context.Context, vendorID, categoryID, status string) error
	FindVendorProfile(ctx context.Context, VendorID string) (*responses.VendorProfileResponse, error)
	UpdateVendorProfile(ctx context.Context, vendorID string, updateData map[string]interface{}) error
}

func NewVendorRepository(db *gorm.DB) VendorRepository {
	return &VendorStorage{
		DB: db,
	}
}

func (s *VendorStorage) RequestCategory(ctx context.Context, vendorID, categoryId string) error {
	categoryRequest := models.CategoryRequest{
		VendorID:   vendorID,
		CategoryID: categoryId,
		Status:     "pending",
	}
	result := s.DB.WithContext(ctx).Create(&categoryRequest)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *VendorStorage) CategoryExists(ctx context.Context, categoryID string) (bool, error) {
	var count int64
	err := s.DB.WithContext(ctx).Model(&models.Category{}).Where("category_id = ?", categoryID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *VendorStorage) HasRequestedCategory(ctx context.Context, vendorID string) (bool, error) {
	var count int64

	err := r.DB.WithContext(ctx).
		Model(&models.CategoryRequest{}).
		Where("vendor_id = ?", vendorID).
		Count(&count).Error

	if err != nil {
		return false, err
	}
	return count > 0, nil
}
func (r *VendorStorage) ListCategories(ctx context.Context) ([]models.Category, error) {
	var categories []models.Category

	err := r.DB.Statement.DB.WithContext(ctx).
		Find(&categories).Error

	if err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *VendorStorage) UpdateCategoryRequestStatus(ctx context.Context, vendorID, categoryID, status string) error {
	result := r.DB.WithContext(ctx).
		Model(&models.CategoryRequest{}).
		Where("vendor_id = ? AND category_id = ?", vendorID, categoryID).
		Update("status", status)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("category request not found")
	}
	return nil
}

func (r *VendorStorage) FindVendorProfile(ctx context.Context, vendorID string) (*responses.VendorProfileResponse, error) {
	var vendorProfile responses.VendorProfileResponse
	err := r.DB.
		Table("users").
		Select("users.user_id, user_details.first_name, user_details.last_name, users.email, user_details.profile_image, user_details.phone, users.status").
		Joins("JOIN user_details ON user_details.user_id = users.user_id").
		Where("users.user_id = ? AND users.role = ?", vendorID, "vendor").
		First(&vendorProfile).Error

	if err != nil {
		return nil, err
	}

	log.Print("Vendor Profile details:", vendorProfile)

	response := &responses.VendorProfileResponse{
		UserID:        vendorProfile.UserID,
		FirstName:     vendorProfile.FirstName,
		LastName:      vendorProfile.LastName,
		Email:         vendorProfile.Email,
		ProfileImage:  vendorProfile.ProfileImage,
		PhoneNumber:   vendorProfile.PhoneNumber,
		RequestStatus: vendorProfile.RequestStatus,

		// Categories:   vendorProfile.Categories,
		// Bio:          vendorProfile.Bio,
	}

	return response, nil
}


func (r *VendorStorage) UpdateVendorProfile(ctx context.Context, vendorID string, updateData map[string]interface{}) error {
	err := r.DB.WithContext(ctx).
		Table("user_details").
		Where("user_id = ?", vendorID).
		Updates(updateData).Error

	if err != nil {
		log.Printf("Error updating vendor profile: %v", err)
		return err
	}

	return nil
}
