package services

import (
	"context"
	"fmt"
	"log"
	"net/http"

	pb "github.com/AthulKrishna2501/proto-repo/vendor"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/app/config"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/repository"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type VendorService struct {
	pb.UnimplementedVendorSeviceServer
	vendorRepo  repository.VendorRepository
	redisClient *redis.Client
}

func NewVendorService(vendorRepo repository.VendorRepository) *VendorService {
	return &VendorService{vendorRepo: vendorRepo, redisClient: config.RedisClient}
}

func (s *VendorService) RequestCategory(ctx context.Context, req *pb.RequestCategoryRequest) (*pb.RequestCategoryResponse, error) {
	if req.CategoryId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Category ID cannot be empty")
	}

	log.Println(req.CategoryId)

	categoryExists, err := s.vendorRepo.CategoryExists(ctx, req.CategoryId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error checking category: %v", err)
	}

	if !categoryExists {
		return nil, status.Errorf(codes.NotFound, "Category does not exist")
	}

	if err := s.vendorRepo.RequestCategory(ctx, req.VendorId, req.CategoryId); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create request: %v", err)
	}

	alreadyRequested, err := s.vendorRepo.HasRequestedCategory(ctx, req.VendorId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to check existing request: %v", err)
	}
	if alreadyRequested {
		return nil, status.Errorf(codes.AlreadyExists, "Vendor has already requested for category")
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
			CategoryId:   cat.CategoryID,
			CategoryName: cat.CategoryName,
			Title:        cat.Title,
			Image:        cat.Image,
		})
	}

	return &pb.ListCategoryResponse{
		Categories: categoryResponses,
	}, nil
}

func (s *VendorService) ApproveRejectCategory(ctx context.Context, req *pb.ApproveRejectCategoryRequest) (*pb.ApproveRejectCategoryResponse, error) {
	log.Printf("Received gRPC request: VendorID=%s, CategoryID=%s, Status=%s", req.VendorId, req.CategoryId, req.Status)

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
