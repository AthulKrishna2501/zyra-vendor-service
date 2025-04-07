package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	pb "github.com/AthulKrishna2501/proto-repo/vendor"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/app/config"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/models"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/repository"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/logger"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type VendorService struct {
	pb.UnimplementedVendorSeviceServer
	vendorRepo  repository.VendorRepository
	redisClient *redis.Client
	log         logger.Logger
}

func NewVendorService(vendorRepo repository.VendorRepository, logger logger.Logger) *VendorService {
	return &VendorService{vendorRepo: vendorRepo, redisClient: config.RedisClient, log: logger}
}

func (s *VendorService) RequestCategory(ctx context.Context, req *pb.RequestCategoryRequest) (*pb.RequestCategoryResponse, error) {
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

	if err := s.vendorRepo.RequestCategory(ctx, req.VendorId, req.CategoryName); err != nil {
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
		Categories:    vendor.Category,
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

	var availableDate time.Time
	if len(req.AvailableDates) > 0 && req.AvailableDates[0] != nil {
		availableDate = req.AvailableDates[0].AsTime()
	}

	service := models.Service{
		ID:                 serviceID,
		VendorID:           uuid.MustParse(req.VendorId),
		ServiceTitle:       req.ServiceTitle,
		AvailableDate:      availableDate,
		YearOfExperience:   int(req.YearOfExperience),
		ServiceDescription: req.ServiceDescription,
		CancellationPolicy: req.CancellationPolicy,
		TermsAndConditions: req.TermsAndConditions,
		ServiceDuration:    int(req.ServiceDuration),
		ServicePrice:       int(req.ServicePrice),
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
		ID:                 serviceUUID,
		ServiceTitle:       req.ServiceTitle,
		YearOfExperience:   int(req.YearOfExperience),
		ServiceDescription: req.ServiceDescription,
		AvailableDate:      availableDate,
		CancellationPolicy: req.CancellationPolicy,
		TermsAndConditions: req.TermsAndConditions,
		ServiceDuration:    int(req.ServiceDuration),
		ServicePrice:       int(req.ServicePrice),
	}

	if req.AdditionalHourPrice != nil {
		price := int(*req.AdditionalHourPrice)
		updatedService.AdditionalHourPrice = &price
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

	return &pb.GetVendorDashboardResponse{
		TotalClientsServed: dash.TotalClientsServed,
		TotalBookings:      dash.TotalBookings,
		TotalRevenue:       dash.TotalRevenue,
	}, nil
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
		var additionalHourPrice int64
		if service.AdditionalHourPrice != nil {
			additionalHourPrice = int64(*service.AdditionalHourPrice)
		}

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
			AdditionalHourPrice: additionalHourPrice,
		})
	}

	return &pb.GetVendorServicesResponse{
		Services: serviceList,
	}, nil
}
