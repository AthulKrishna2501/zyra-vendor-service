package grpc

import (
	"net"

	"github.com/AthulKrishna2501/proto-repo/vendor"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/app/config"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/repository"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/services"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/logger"
	"google.golang.org/grpc"
)

func StartgRPCServer(VendorRepo repository.VendorRepository, log logger.Logger, cfg config.Config) error {
	go func() {
		lis, err := net.Listen("tcp", ":5004")
		if err != nil {
			log.Error("Failed to listen on port 5004: %v", err)
			return
		}

		grpcServer := grpc.NewServer(
			grpc.MaxRecvMsgSize(1024*1024*100),
			grpc.MaxSendMsgSize(1024*1024*100),
		)
		vendorService := services.NewVendorService(VendorRepo, log, cfg)
		vendor.RegisterVendorSeviceServer(grpcServer, vendorService)

		log.Info("gRPC Server started on port 5004")
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("Failed to serve gRPC: %v", err)
		}
	}()

	return nil

}
