package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	adminModel "github.com/AthulKrishna2501/zyra-admin-service/internals/core/models"
	auth "github.com/AthulKrishna2501/zyra-auth-service/internals/core/models"
	clientModel "github.com/AthulKrishna2501/zyra-client-service/internals/core/models"
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
	AddToVendorWallet(ctx context.Context, vendorId string, amount int64) error
	CategoryExists(ctx context.Context, categoryId string) (bool, error)
	CreateAdminWalletTransaction(ctx context.Context, newAdminWalletTransaction *adminModel.AdminWalletTransaction) error
	CreateService(service *models.Service) error
	CreateTransaction(ctx context.Context, newTransaction *clientModel.Transaction) error
	FindVendorProfile(ctx context.Context, VendorID uuid.UUID) (*responses.VendorProfileResponse, error)
	GetBookingById(ctx context.Context, bookingId string) (*adminModel.Booking, error)
	GetServicesByVendor(ctx context.Context, vendorID string) ([]models.Service, error)
	GetVendorBookings(ctx context.Context, vendorID string) ([]responses.BookingInfo, error)
	GetVendorByID(vendorID string) (*auth.User, error)
	GetVendorDashboard(ctx context.Context, vendorID string) (*requests.VendorDashboard, error)
	GetVendorStatus(vendorID string) (*auth.User, error)
	GetWalletBalance(ctx context.Context, vendorID string) (int64, error)
	HasRequestedCategory(ctx context.Context, vendorID string) (bool, error)
	ListCategories(ctx context.Context) ([]models.Category, error)
	RefundAmount(ctx context.Context, adminEmail string, clientID string, amount int) error
	RequestCategory(ctx context.Context, vendorID, categoryId string) error
	UpdateBookingStatus(ctx context.Context, bookingId string, status string) error
	ReleasePaymentToVendor(ctx context.Context, vendorID string, price float64) error
	UpdateCategoryRequestStatus(ctx context.Context, vendorID, categoryID, status string) error
	UpdateService(serviceID uuid.UUID, updatedService models.Service) error
	UpdateVendorPassword(vendorID string, newPassword string) error
	UpdateVendorProfile(ctx context.Context, vendorID uuid.UUID, updateData map[string]interface{}) error
	UpdateVendorApproval(ctx context.Context, bookingID string, status bool) error
	MarkBookingAsConfirmedAndReleased(ctx context.Context, bookingID string) error
	GetVendorWallet(ctx context.Context, vendorID string) (*models.Wallet, error)
	GetVendorTransactions(ctx context.Context, vendorID string) ([]clientModel.Transaction, error)
	GetMonthlyRevenue(ctx context.Context, vendorId string) ([]*responses.Result, error)
	GetTopServices(ctx context.Context, vendorId string) ([]*responses.ServiceStat, error)
	IsInCategory(vendorID string) (bool, error)
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
	service.AdditionalHourPrice = updatedService.AdditionalHourPrice

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

	err := s.DB.WithContext(ctx).
		Model(&adminModel.Booking{}).
		Where("vendor_id = ?", vendorID).
		Select("COUNT(DISTINCT client_id) as total_clients_served, COUNT(*) as total_bookings, COALESCE(SUM(price), 0) as total_revenue").
		Scan(&dashboard).Error
	if err != nil {
		return nil, err
	}

	now := time.Now()
	firstOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	firstOfPreviousMonth := firstOfCurrentMonth.AddDate(0, -1, 0)
	endOfPreviousMonth := firstOfCurrentMonth.Add(-time.Nanosecond)

	err = s.DB.WithContext(ctx).
		Model(&adminModel.Booking{}).
		Where("vendor_id = ? AND date >= ?", vendorID, firstOfCurrentMonth).
		Count(&dashboard.CurrentMonthBookings).Error
	if err != nil {
		return nil, err
	}

	err = s.DB.WithContext(ctx).
		Model(&adminModel.Booking{}).
		Where("vendor_id = ? AND date BETWEEN ? AND ?", vendorID, firstOfPreviousMonth, endOfPreviousMonth).
		Count(&dashboard.PreviousMonthBookings).Error
	if err != nil {
		return nil, err
	}

	err = s.DB.WithContext(ctx).
		Model(&adminModel.Booking{}).
		Where("vendor_id = ? AND date >= ?", vendorID, firstOfCurrentMonth).
		Distinct("client_id").
		Count(&dashboard.CurrentMonthClients).Error
	if err != nil {
		return nil, err
	}

	err = s.DB.WithContext(ctx).
		Model(&adminModel.Booking{}).
		Where("vendor_id = ? AND date BETWEEN ? AND ?", vendorID, firstOfPreviousMonth, endOfPreviousMonth).
		Distinct("client_id").
		Count(&dashboard.PreviousMonthClients).Error
	if err != nil {
		return nil, err
	}

	err = s.DB.WithContext(ctx).
		Model(&adminModel.Booking{}).
		Where("vendor_id = ? AND status = ?", vendorID, "pending").
		Select("COALESCE(SUM(price), 0)").
		Scan(&dashboard.PendingPayments).Error
	if err != nil {
		return nil, err
	}

	err = s.DB.WithContext(ctx).
		Model(&clientModel.Review{}).
		Where("vendor_id = ?", vendorID).
		Select("AVG(rating) as average_rating, COUNT(*) as total_reviews").
		Scan(&dashboard).Error
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

func (r *VendorStorage) GetVendorBookings(ctx context.Context, vendorID string) ([]responses.BookingInfo, error) {
	var bookings []responses.BookingInfo

	err := r.DB.WithContext(ctx).
		Table("bookings b").
		Select(`
			b.booking_id AS booking_id,
			ud.first_name || ' ' || ud.last_name AS client_name,
			b.service,
			b.date,
			b.price,
			b.status,
			b.created_at
		`).
		Joins("JOIN user_details ud ON b.client_id = ud.user_id").
		Where("b.vendor_id = ?", vendorID).
		Scan(&bookings).Error

	if err != nil {
		return nil, err
	}

	return bookings, nil
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

func (r *VendorStorage) RefundAmount(ctx context.Context, adminEmail string, clientID string, amount int) error {
	var wallet models.Wallet
	var adminWallet adminModel.AdminWallet

	err := r.DB.WithContext(ctx).Model(&adminWallet).Where("email = ?", adminEmail).Update("balance", gorm.Expr("balance - ?", amount)).Error
	if err != nil {
		return err
	}

	err = r.DB.WithContext(ctx).Model(&adminWallet).Where("email = ?", adminEmail).Update("total_withdrawals", gorm.Expr("total_withdrawals + ?", amount)).Error

	if err != nil {
		return nil
	}

	err = r.DB.WithContext(ctx).Model(&wallet).Where("client_id = ?", clientID).First(&wallet).Error

	clientUUID, _ := uuid.Parse(clientID)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		newWallet := models.Wallet{
			ClientID:      clientUUID,
			WalletBalance: int64(amount),
		}

		return r.DB.WithContext(ctx).Create(&newWallet).Error
	} else if err != nil {
		return err
	}

	err = r.DB.WithContext(ctx).Model(wallet).Where("client_id = ?", clientID).Update("wallet_balance", gorm.Expr("wallet_balance + ?", amount)).Error

	if err != nil {
		return err
	}

	return r.DB.WithContext(ctx).Model(wallet).Where("client_id = ?", clientID).Update("total_deposits", gorm.Expr("total_deposits + ?", amount)).Error

}

func (r *VendorStorage) CreateTransaction(ctx context.Context, newTransaction *clientModel.Transaction) error {
	return r.DB.WithContext(ctx).Create(newTransaction).Error

}

func (r *VendorStorage) CreateAdminWalletTransaction(ctx context.Context, newAdminWalletTransaction *adminModel.AdminWalletTransaction) error {
	return r.DB.WithContext(ctx).Create(newAdminWalletTransaction).Error
}

func (r *VendorStorage) UpdateVendorApproval(ctx context.Context, bookingID string, status bool) error {
	return r.DB.WithContext(ctx).Model(&adminModel.Booking{}).Where("booking_id = ?", bookingID).Update("is_vendor_approved", status).Error
}

func (r *VendorStorage) ReleasePaymentToVendor(ctx context.Context, vendorID string, price float64) error {
	tx := r.DB.WithContext(ctx).Begin() // Start transaction

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var adminWallet adminModel.AdminWallet
	if err := tx.Where("email = ?", "admin@gmail.com").First(&adminWallet).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to fetch admin wallet: %w", err)
	}

	if adminWallet.Balance < price {
		tx.Rollback()
		return fmt.Errorf("admin wallet has insufficient balance")
	}

	var vendorWallet models.Wallet
	vendorUUID, err := uuid.Parse(vendorID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("invalid vendor ID: %w", err)
	}
	if err := tx.Where("vendor_id = ?", vendorUUID).First(&vendorWallet).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to fetch vendor wallet: %w", err)
	}

	adminWallet.Balance -= price
	adminWallet.TotalWithdrawals += price
	if err := tx.Save(&adminWallet).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update admin wallet: %w", err)
	}

	vendorWallet.WalletBalance += int64(price)
	vendorWallet.TotalDeposits += int64(price)
	if err := tx.Save(&vendorWallet).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update vendor wallet: %w", err)
	}

	adminWalletTxn := &adminModel.AdminWalletTransaction{
		Date:   time.Now(),
		Type:   "Vendor Payout",
		Amount: price,
		Status: "succeded",
	}
	if err := tx.Create(adminWalletTxn).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create admin wallet transaction: %w", err)
	}

	vendorTxn := &clientModel.Transaction{
		UserID:        vendorWallet.VendorID,
		Purpose:       "Vendor Payout",
		AmountPaid:    int(price),
		PaymentMethod: "wallet",
		PaymentStatus: "Paid",
		DateOfPayment: time.Now(),
	}
	if err := tx.Create(vendorTxn).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create vendor transaction: %w", err)
	}

	return tx.Commit().Error
}

func (r *VendorStorage) MarkBookingAsConfirmedAndReleased(ctx context.Context, bookingID string) error {
	tx := r.DB.WithContext(ctx).Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var booking adminModel.Booking
	if err := tx.Where("booking_id = ?", bookingID).First(&booking).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find booking: %w", err)
	}

	booking.IsVendorApproved = true
	booking.IsClientApproved = true
	booking.IsFundReleased = true
	booking.UpdatedAt = time.Now()

	if err := tx.Save(&booking).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update booking: %w", err)
	}

	return tx.Commit().Error
}

func (r *VendorStorage) GetVendorWallet(ctx context.Context, vendorID string) (*models.Wallet, error) {
	var wallet models.Wallet

	err := r.DB.Where("vendor_id = ?", vendorID).First(&wallet).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		vendorUUID, err := uuid.Parse(vendorID)
		if err != nil {
			return nil, err
		}
		newWallet := &models.Wallet{
			VendorID:         vendorUUID,
			WalletBalance:    0,
			TotalDeposits:    0,
			TotalWithdrawals: 0,
		}

		err = r.DB.Create(newWallet).Error
		if err != nil {
			return nil, err
		}

	} else if err != nil {
		return nil, err
	}

	return &wallet, nil
}

func (r *VendorStorage) GetVendorTransactions(ctx context.Context, vendorID string) ([]clientModel.Transaction, error) {
	var transactions []clientModel.Transaction

	err := r.DB.WithContext(ctx).Where("user_id = ?", vendorID).Find(&transactions).Error

	if err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *VendorStorage) GetMonthlyRevenue(ctx context.Context, vendorId string) ([]*responses.Result, error) {
	var results []*responses.Result

	err := r.DB.WithContext(ctx).
		Model(&adminModel.Booking{}).
		Select("TO_CHAR(date, 'Mon') AS month, SUM(price) AS revenue").
		Where("vendor_id = ?", vendorId).
		Group("month").
		Order("MIN(DATE_PART('month',date))").
		Limit(6).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *VendorStorage) GetTopServices(ctx context.Context, vendorId string) ([]*responses.ServiceStat, error) {

	var services []*responses.ServiceStat

	err := r.DB.WithContext(ctx).
		Model(&adminModel.Booking{}).
		Select("service, COUNT(*) AS total_bookings").
		Where("vendor_id = ?", vendorId).
		Group("service").
		Order("total_bookings DESC").
		Limit(5).
		Scan(&services).Error

	if err != nil {
		return nil, err
	}

	return services, nil
}

func (r *VendorStorage) IsInCategory(vendorID string) (bool, error) {
	var count int64
	err := r.DB.
		Model(&models.VendorCategory{}).
		Where("vendor_id = ?", vendorID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
