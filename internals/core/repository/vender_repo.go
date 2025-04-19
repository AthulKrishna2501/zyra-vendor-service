package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	adminModel "github.com/AthulKrishna2501/zyra-admin-service/internals/core/models"
	auth "github.com/AthulKrishna2501/zyra-auth-service/internals/core/models"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/models"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/models/requests"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/models/responses"

	"github.com/google/uuid"
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
	FindVendorProfile(ctx context.Context, VendorID uuid.UUID) (*responses.VendorProfileResponse, error)
	UpdateVendorProfile(ctx context.Context, vendorID uuid.UUID, updateData map[string]interface{}) error
	CreateService(service *models.Service) error
	UpdateService(serviceID uuid.UUID, updatedService models.Service) error
	GetVendorByID(vendorID string) (*auth.User, error)
	UpdateVendorPassword(vendorID string, newPassword string) error
	GetVendorStatus(vendorID string) (*auth.User, error)
	GetVendorDashboard(ctx context.Context, vendorID string) (*requests.VendorDashboard, error)
	GetServicesByVendor(ctx context.Context, vendorID string) ([]models.Service, error)
	GetBookingsByVendor(ctx context.Context, vendorID string, bookings *[]responses.BookingInfo) error
	GetBookingById(ctx context.Context, bookingId string) (*adminModel.Booking, error)
	UpdateBookingStatus(ctx context.Context, bookingId string, status string) error
	AddToVendorWallet(ctx context.Context, vendorId string, amount int64) error
	GetWalletBalance(ctx context.Context, vendorID string) (int64, error)
}

func NewVendorRepository(db *gorm.DB) VendorRepository {
	return &VendorStorage{
		DB: db,
	}
}

func (s *VendorStorage) RequestCategory(ctx context.Context, vendorID, categoryName string) error {
	vendorUUID, err := uuid.Parse(vendorID)
	if err != nil {
		return fmt.Errorf("invalid vendor ID format: %v", err)
	}

	var category models.Category
	err = s.DB.WithContext(ctx).
		Where("category_name = ?", categoryName).
		First(&category).Error
	if err != nil {
		return fmt.Errorf("category with name '%s' not found: %v", categoryName, err)
	}
	categoryRequest := models.CategoryRequest{
		VendorID:   vendorUUID,
		CategoryID: category.CategoryID,
	}

	result := s.DB.WithContext(ctx).Create(&categoryRequest)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *VendorStorage) CategoryExists(ctx context.Context, categoryName string) (bool, error) {
	var count int64
	err := s.DB.WithContext(ctx).Model(&models.Category{}).Where("category_name = ?", categoryName).Count(&count).Error
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

func (r *VendorStorage) FindVendorProfile(ctx context.Context, vendorID uuid.UUID) (*responses.VendorProfileResponse, error) {
	var vendorProfile responses.VendorProfileResponse
	var category string

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
		Category:      category,
	}

	return response, nil
}

func (r *VendorStorage) UpdateVendorProfile(ctx context.Context, vendorID uuid.UUID, updateData map[string]interface{}) error {
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

func (r *VendorStorage) CreateService(service *models.Service) error {
	return r.DB.Create(service).Error
}

func (r *VendorStorage) UpdateService(serviceID uuid.UUID, updatedService models.Service) error {
	var service models.Service

	if err := r.DB.First(&service, "id = ?", serviceID).Error; err != nil {
		return err
	}

	if !updatedService.AvailableDate.IsZero() {
		service.AvailableDate = updatedService.AvailableDate
	}

	service.ServiceTitle = updatedService.ServiceTitle
	service.YearOfExperience = updatedService.YearOfExperience
	service.ServiceDescription = updatedService.ServiceDescription
	service.CancellationPolicy = updatedService.CancellationPolicy
	service.TermsAndConditions = updatedService.TermsAndConditions
	service.ServiceDuration = updatedService.ServiceDuration
	service.ServicePrice = updatedService.ServicePrice

	if updatedService.AdditionalHourPrice != nil {
		service.AdditionalHourPrice = updatedService.AdditionalHourPrice
	}

	if err := r.DB.Save(&service).Error; err != nil {
		return err
	}

	return nil
}

func (r *VendorStorage) GetVendorByID(vendorID string) (*auth.User, error) {
	var vendor auth.User
	err := r.DB.Where("user_id = ?", vendorID).First(&vendor).Error
	if err != nil {
		return nil, errors.New("vendor not found")
	}
	return &vendor, nil
}

func (r *VendorStorage) UpdateVendorPassword(vendorID string, newPassword string) error {
	return r.DB.Model(&auth.User{}).Where("user_id = ?", vendorID).Update("password", newPassword).Error
}

func (r *VendorStorage) GetVendorStatus(vendorID string) (*auth.User, error) {
	var vendor auth.User
	err := r.DB.Where("user_id = ?", vendorID).First(&vendor).Error
	if err != nil {
		return nil, errors.New("vendor not found")
	}
	return &vendor, nil
}

func (s *VendorStorage) GetVendorDashboard(ctx context.Context, vendorID string) (*requests.VendorDashboard, error) {
	var dashboard requests.VendorDashboard

	err := s.DB.Raw(`
        SELECT 
            COUNT(DISTINCT client_id) AS total_clients_served,
            COUNT(*) AS total_bookings,
            COALESCE(SUM(price), 0) AS total_revenue
        FROM bookings
        WHERE vendor_id = ?`, vendorID).Scan(&dashboard).Error

	if err != nil {
		return nil, err
	}

	return &dashboard, nil
}

func (r *VendorStorage) GetServicesByVendor(ctx context.Context, vendorID string) ([]models.Service, error) {
	var services []models.Service
	err := r.DB.WithContext(ctx).Where("vendor_id = ?", vendorID).Find(&services).Error
	if err != nil {
		return nil, err
	}
	return services, nil
}

func (r *VendorStorage) GetBookingsByVendor(ctx context.Context, vendorID string, bookings *[]responses.BookingInfo) error {
	query := `
    SELECT 
        b.booking_id AS booking_id,
        ud.first_name || ' ' || ud.last_name AS client_name,
        b.service AS service,
        b.date AS date,
        b.price AS price,
        b.status AS status
    FROM bookings b
    JOIN user_details ud ON b.client_id = ud.user_id
    WHERE b.vendor_id = ?
`

	return r.DB.WithContext(ctx).Raw(query, vendorID).Scan(bookings).Error
}

func (r *VendorStorage) GetBookingById(ctx context.Context, bookingId string) (*adminModel.Booking, error) {
	var booking adminModel.Booking
	err := r.DB.WithContext(ctx).Where("booking_id = ?", bookingId).First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *VendorStorage) UpdateBookingStatus(ctx context.Context, bookingId string, status string) error {
	return r.DB.WithContext(ctx).Model(&adminModel.Booking{}).Where("booking_id = ?", bookingId).Update("status", status).Error
}

func (r *VendorStorage) AddToVendorWallet(ctx context.Context, vendorId string, amount int64) error {
	var wallet models.Wallet

	err := r.DB.WithContext(ctx).Where("vendor_id = ?", vendorId).First(&wallet).Error

	vendorUUID, _ := uuid.Parse(vendorId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		newWallet := models.Wallet{
			VendorID:      vendorUUID,
			WalletBalance: amount,
		}
		return r.DB.WithContext(ctx).Create(&newWallet).Error
	} else if err != nil {
		return err
	}

	return r.DB.WithContext(ctx).
		Model(&wallet).
		Update("wallet_balance", gorm.Expr("wallet_balance + ?", amount)).Error

}

func (r *VendorStorage) GetWalletBalance(ctx context.Context, vendorID string) (int64, error) {
	var wallet models.Wallet
	err := r.DB.WithContext(ctx).Where("vendor_id = ?", vendorID).First(&wallet).Error
	if err != nil {
		return 0, err
	}
	return wallet.WalletBalance, nil
}
