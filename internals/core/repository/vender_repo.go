package repository

import (
	"context"
	"errors"

	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/models"
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
