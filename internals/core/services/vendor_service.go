package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	pb "github.com/AthulKrishna2501/proto-repo/vendor"
	adminModel "github.com/AthulKrishna2501/zyra-admin-service/internals/core/models"
	clientModel "github.com/AthulKrishna2501/zyra-client-service/internals/core/models"

	"github.com/AthulKrishna2501/zyra-vendor-service/internals/app/config"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/models"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/repository"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/logger"
	"github.com/AthulKrishna2501/zyra-vendor-service/utils"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type VendorService struct {
	pb.UnimplementedVendorSeviceServer
	vendorRepo  repository.VendorRepository
	redisClient *redis.Client
	log         logger.Logger
	cfg         config.Config
}

func NewVendorService(vendorRepo repository.VendorRepository, logger logger.Logger, cfg config.Config) *VendorService {
	return &VendorService{vendorRepo: vendorRepo, redisClient: config.RedisClient, log: logger, cfg: cfg}
}

func (s *VendorService) RequestCategory(ctx context.Context, req *pb.RequestCategoryRequest) (*pb.RequestCategoryResponse, error) {
	s.log.Info("Category ID in Request Category:", req.GetVendorId())
	if req.CategoryName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Category name cannot be empty")
	}
	categoryExists, err := s.vendorRepo.CategoryExists(ctx, req.CategoryName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error checking category: %v", err)
	}

	if !categoryExists {
		return nil, status.Errorf(codes.NotFound, "Category does not exist")
	}

	alreadyRequested, err := s.vendorRepo.HasRequestedCategory(ctx, req.VendorId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to check existing request: %v", err)
	}
	if alreadyRequested {
		return nil, status.Errorf(codes.AlreadyExists, "Vendor has already requested for category")
	}

	if err := s.vendorRepo.RequestCategory(ctx, req.GetVendorId(), req.GetCategoryName()); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create request: %v", err)
	}

	return &pb.RequestCategoryResponse{
		Status:  http.StatusOK,
		Message: "Category request submitted successfully",
	}, nil

}

func (s *VendorService) ListCategory(ctx context.Context, req *pb.ListCategoryRequest) (*pb.ListCategoryResponse, error) {
	categories, err := s.vendorRepo.ListCategories(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to fetch categories: %v", err)
	}

	var categoryResponses []*pb.Category
	for _, cat := range categories {
		categoryResponses = append(categoryResponses, &pb.Category{
			CategoryId:   cat.CategoryID.String(),
			CategoryName: cat.CategoryName,
		})

	}

	return &pb.ListCategoryResponse{
		Categories: categoryResponses,
	}, nil
}

func (s *VendorService) ApproveRejectCategory(ctx context.Context, req *pb.ApproveRejectCategoryRequest) (*pb.ApproveRejectCategoryResponse, error) {
	s.log.Info("Received gRPC request: VendorID=%s, CategoryID=%s, Status=%s", req.VendorId, req.CategoryId, req.Status)

	if req.Status != "approved" && req.Status != "rejected" {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid status. Allowed values: 'approved', 'rejected'")
	}

	err := s.vendorRepo.UpdateCategoryRequestStatus(ctx, req.VendorId, req.CategoryId, req.Status)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Category request not found")
	}

	return &pb.ApproveRejectCategoryResponse{
		Message: fmt.Sprintf("Category request has been %s", req.Status),
	}, nil
}

func (s *VendorService) VendorProfile(ctx context.Context, req *pb.VendorProfileRequest) (*pb.VendorProfileResponse, error) {
	vendorUUID, err := uuid.Parse(req.VendorId)
	if err != nil {
		return nil, fmt.Errorf("invalid vendor ID format: %v", err)
	}

	vendor, err := s.vendorRepo.FindVendorProfile(ctx, vendorUUID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.VendorProfileResponse{
		UserId:        vendor.UserID,
		FirstName:     vendor.FirstName,
		LastName:      vendor.LastName,
		Email:         vendor.Email,
		PhoneNumber:   vendor.PhoneNumber,
		ProfileImage:  vendor.ProfileImage,
		RequestStatus: vendor.RequestStatus,
		Category:      vendor.CategoryName,
	}, nil
}

func (s *VendorService) UpdateProfile(ctx context.Context, req *pb.UpdateVendorProfileRequest) (*pb.UpdateVendorProfileResponse, error) {
	updateData := map[string]interface{}{}

	if *req.FirstName != "" {
		updateData["first_name"] = req.FirstName
	}
	if *req.LastAme != "" {
		updateData["last_name"] = req.LastAme
	}
	if *req.ProfileImage != "" {
		updateData["profile_image"] = req.ProfileImage
	}
	if *req.PhoneNumber != "" {
		if len(*req.PhoneNumber) != 10 {
			return nil, fmt.Errorf("phone number should be 10 digits of length")
		}
		updateData["phone"] = req.PhoneNumber
	}
	if *req.Bio != "" {
		updateData["bio"] = req.Bio
	}

	if len(updateData) == 0 {
		return nil, status.Error(codes.InvalidArgument, "No fields provided for update")
	}

	vendorUUID, err := uuid.Parse(req.VendorId)
	if err != nil {
		return nil, fmt.Errorf("invalid vendor ID format: %v", err)
	}

	err = s.vendorRepo.UpdateVendorProfile(ctx, vendorUUID, updateData)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpdateVendorProfileResponse{
		Message: "Vendor profile updated successfully",
	}, nil
}

func (s *VendorService) CreateService(ctx context.Context, req *pb.CreateServiceRequest) (*pb.CreateServiceResponse, error) {
	serviceID := uuid.New()

	vendor, err := s.vendorRepo.GetVendorStatus(req.VendorId)
	if err != nil {
		return nil, errors.New("vendor not found")
	}

	if vendor.Status != "approved" {
		return nil, errors.New("vendor is not approved to add a service")
	}

	isInCategory, err := s.vendorRepo.IsInCategory(req.VendorId)

	if err != nil {
		return nil, errors.New("failed to check vendor in category")
	}

	if !isInCategory {
		return &pb.CreateServiceResponse{
			Message: "Vendor does not belong to any category. Please assign a category before creating the service.",
		}, nil
	}

	var availableDate time.Time
	if len(req.AvailableDates) > 0 && req.AvailableDates[0] != nil {
		availableDate = req.AvailableDates[0].AsTime()
	}

	service := models.Service{
		ID:                  serviceID,
		VendorID:            uuid.MustParse(req.VendorId),
		ServiceTitle:        req.ServiceTitle,
		AvailableDate:       availableDate,
		YearOfExperience:    int(req.YearOfExperience),
		ServiceDescription:  req.ServiceDescription,
		CancellationPolicy:  req.CancellationPolicy,
		TermsAndConditions:  req.TermsAndConditions,
		ServiceDuration:     int(req.ServiceDuration),
		ServicePrice:        int(req.ServicePrice),
		AdditionalHourPrice: int(req.AdditionalHourPrice),
	}

	if err := s.vendorRepo.CreateService(&service); err != nil {
		return nil, err
	}

	return &pb.CreateServiceResponse{
		Message: "Service Created successfully",
	}, nil

}

func (s *VendorService) UpdateService(ctx context.Context, req *pb.UpdateServiceRequest) (*pb.UpdateServiceResponse, error) {

	serviceUUID, err := uuid.Parse(req.ServiceId)
	if err != nil {
		return nil, fmt.Errorf("invalid service ID format: %v", err)
	}

	var availableDate time.Time
	if len(req.AvailableDates) > 0 && req.AvailableDates[0] != nil {
		availableDate = req.AvailableDates[0].AsTime()
	}

	updatedService := models.Service{
		ID:                  serviceUUID,
		ServiceTitle:        req.ServiceTitle,
		YearOfExperience:    int(req.YearOfExperience),
		ServiceDescription:  req.ServiceDescription,
		AvailableDate:       availableDate,
		CancellationPolicy:  req.CancellationPolicy,
		TermsAndConditions:  req.TermsAndConditions,
		ServiceDuration:     int(req.ServiceDuration),
		ServicePrice:        int(req.ServicePrice),
		AdditionalHourPrice: int(*req.AdditionalHourPrice),
	}

	err = s.vendorRepo.UpdateService(serviceUUID, updatedService)
	if err != nil {
		return nil, fmt.Errorf("failed to update service: %v", err)
	}

	return &pb.UpdateServiceResponse{
		Message: "Service Updated Successfully",
	}, nil

}

func (s *VendorService) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	vendor, err := s.vendorRepo.GetVendorByID(req.VendorId)
	if err != nil {
		return nil, errors.New("vendor not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(vendor.Password), []byte(req.CurrentPassword))
	if err != nil {
		return nil, errors.New("incorrect current password")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash new password")
	}

	err = s.vendorRepo.UpdateVendorPassword(req.VendorId, string(hashedPassword))
	if err != nil {
		return nil, errors.New("failed to update password")
	}

	return &pb.ChangePasswordResponse{
		Message: "Password changed successfully",
	}, nil
}

func (s *VendorService) GetVendorDashboard(ctx context.Context, req *pb.GetVendorDashboardRequest) (*pb.GetVendorDashboardResponse, error) {
	dash, err := s.vendorRepo.GetVendorDashboard(ctx, req.VendorId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch dashboard data: %v", err)
	}

	monthlyRevenue, err := s.vendorRepo.GetMonthlyRevenue(ctx, req.VendorId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch monthly revenue trend: %v", err)
	}

	var revenues []*pb.MonthRevenue
	for _, res := range monthlyRevenue {
		revenues = append(revenues, &pb.MonthRevenue{
			Month:   res.Month,
			Revenue: res.Revenue,
		})
	}

	topServices, err := s.vendorRepo.GetTopServices(ctx, req.VendorId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch top services: %v", err)
	}

	averageRevenuePerBooking := float64(0)
	if dash.TotalBookings > 0 {
		averageRevenuePerBooking = float64(dash.TotalRevenue) / float64(dash.TotalBookings)
	}

	bookingGrowthRate := utils.CalculateGrowthRate(int32(dash.CurrentMonthBookings), int32(dash.PreviousMonthBookings))
	clientGrowthRate := utils.CalculateGrowthRate(int32(dash.CurrentMonthClients), int32(dash.PreviousMonthClients))

	var topServicesResp []*pb.ServiceStat
	for _, svc := range topServices {
		topServicesResp = append(topServicesResp, &pb.ServiceStat{
			ServiceName: svc.Service,
			Bookings:    svc.TotalBookings,
		})
	}

	resp := &pb.GetVendorDashboardResponse{
		TotalClientsServed:       dash.TotalClientsServed,
		TotalBookings:            dash.TotalBookings,
		TotalRevenue:             dash.TotalRevenue,
		MonthlyRevenueTrend:      revenues,
		TopServices:              topServicesResp,
		AverageRevenuePerBooking: averageRevenuePerBooking,
		BookingGrowthRate:        bookingGrowthRate,
		ClientGrowthRate:         clientGrowthRate,
		PendingPayments:          dash.PendingPayments,
		AverageRating:            dash.AverageRating,
		TotalReviews:             dash.TotalReviews,
	}

	return resp, nil
}

func (s *VendorService) GetVendorServices(ctx context.Context, req *pb.GetVendorServicesRequest) (*pb.GetVendorServicesResponse, error) {
	if s.vendorRepo == nil {
		return nil, status.Errorf(codes.Internal, "vendorRepo is not initialized")
	}

	if req.VendorId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "vendor_id is required")
	}

	services, err := s.vendorRepo.GetServicesByVendor(ctx, req.VendorId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch services: %v", err)
	}

	if len(services) == 0 {
		return &pb.GetVendorServicesResponse{Services: []*pb.Service{}}, nil
	}

	var serviceList []*pb.Service
	for _, service := range services {
		serviceList = append(serviceList, &pb.Service{
			Id:                  service.ID.String(),
			ServiceTitle:        service.ServiceTitle,
			YearOfExperience:    int64(service.YearOfExperience),
			AvailableDate:       service.AvailableDate.Format(time.RFC3339),
			ServiceDescription:  service.ServiceDescription,
			CancellationPolicy:  service.CancellationPolicy,
			TermsAndConditions:  service.TermsAndConditions,
			ServiceDuration:     int64(service.ServiceDuration),
			ServicePrice:        int64(service.ServicePrice),
			AdditionalHourPrice: int64(service.AdditionalHourPrice),
		})
	}

	return &pb.GetVendorServicesResponse{
		Services: serviceList,
	}, nil
}

func (s *VendorService) GetBookingRequests(ctx context.Context, req *pb.GetBookingRequestsRequest) (*pb.GetBookingRequestsResponse, error) {
	if req.VendorId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "vendor_id is required")
	}

	vendorBookings, err := s.vendorRepo.GetVendorBookings(ctx, req.VendorId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch bookings: %v", err)
	}

	var bookingList []*pb.BookingRequest
	for _, booking := range vendorBookings {
		bookingList = append(bookingList, &pb.BookingRequest{
			BookingId:  booking.BookingID,
			ClientName: booking.ClientName,
			Service:    booking.Service,
			Date:       timestamppb.New(booking.Date),
			Price:      int32(booking.Price),
			Status:     booking.Status,
			BookedAt:   booking.CreatedAt.String(),
		})
	}

	return &pb.GetBookingRequestsResponse{
		Bookings: bookingList,
	}, nil
}

func (s *VendorService) ApproveBooking(ctx context.Context, req *pb.ApproveBookingRequest) (*pb.ApproveBookingResponse, error) {
	if req.BookingId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "booking_id is required")
	}

	if req.VendorId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "vendor_id is required")
	}
	booking, err := s.vendorRepo.GetBookingById(ctx, req.GetBookingId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "booking not found: %v", err)
	}

	vendorUUID, _ := uuid.Parse(req.VendorId)

	if booking.VendorID != vendorUUID {
		return nil, status.Errorf(codes.PermissionDenied, "booking does not belong to the vendor")
	}

	err = s.vendorRepo.UpdateBookingStatus(ctx, req.BookingId, req.Status)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update booking status: %v", err)
	}

	if req.Status == "rejected" {

		newTransaction := &clientModel.Transaction{
			UserID:        booking.ClientID,
			Purpose:       "Vendor Booking",
			AmountPaid:    booking.Price,
			PaymentMethod: "wallet",
			DateOfPayment: time.Now(),
			PaymentStatus: "refunded",
		}

		newAdminWalletTransaction := &adminModel.AdminWalletTransaction{
			Date:   time.Now(),
			Type:   "Vendor Booking",
			Amount: float64(booking.Price),
			Status: "refunded",
		}

		clientIDStr := booking.ClientID.String()

		err = s.vendorRepo.RefundAmount(ctx, s.cfg.ADMIN_EMAIL, clientIDStr, booking.Price)

		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to refund amount %v", err)
		}

		err = s.vendorRepo.CreateTransaction(ctx, newTransaction)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create transaction: %v", err)
		}

		err = s.vendorRepo.CreateAdminWalletTransaction(ctx, newAdminWalletTransaction)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create admin wallet transaction")
		}

		return &pb.ApproveBookingResponse{
			Message: fmt.Sprintf("Booking %s successfully", req.Status),
		}, nil

	}

	err = s.vendorRepo.UpdateVendorApproval(ctx, req.BookingId, true)
	if err != nil {

		return nil, status.Errorf(codes.Internal, "failed to update vendor approval: %v", err)
	}
	if booking.IsVendorApproved && booking.IsClientApproved && !booking.IsFundReleased {
		err = s.vendorRepo.ReleasePaymentToVendor(ctx, booking.VendorID.String(), float64(booking.Price))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to release payment: %v", err)
		}

		err = s.vendorRepo.MarkBookingAsConfirmedAndReleased(ctx, req.BookingId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update booking status: %v", err)
		}

	}

	return &pb.ApproveBookingResponse{
		Message: fmt.Sprintf("Booking %s successfully", req.Status),
	}, nil
}

func (s *VendorService) GetVendorWallet(ctx context.Context, req *pb.GetVendorWalletRequest) (*pb.GetVendorWalletResponse, error) {
	vendorID := req.GetVendorId()
	wallet, err := s.vendorRepo.GetVendorWallet(ctx, vendorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch client wallet: %v", err)
	}

	return &pb.GetVendorWalletResponse{
		Balance:          float32(wallet.WalletBalance),
		TotalDeposits:    float32(wallet.TotalDeposits),
		TotalWithdrawals: float32(wallet.TotalWithdrawals),
	}, nil
}

func (s *VendorService) GetVendorTransactions(ctx context.Context, req *pb.ViewVendorTransactionsRequest) (*pb.ViewVendorTransactionResponse, error) {
	vendorID := req.GetVendorId()
	walletTransactions, err := s.vendorRepo.GetVendorTransactions(ctx, vendorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve admin wallet transactions: %v", err.Error())
	}

	var protoTransactions []*pb.VendorTransaction
	for _, txn := range walletTransactions {
		protoTransactions = append(protoTransactions, &pb.VendorTransaction{
			TransactionId: txn.TransactionID.String(),
			Date:          txn.DateOfPayment.String(),
			Type:          txn.Purpose,
			Amount:        float32(txn.AmountPaid),
			Status:        txn.PaymentStatus,
		})
	}

	return &pb.ViewVendorTransactionResponse{
		Transactions: protoTransactions,
	}, nil
}
