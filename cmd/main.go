package main

import (
	"net"

	"github.com/AthulKrishna2501/proto-repo/vendor"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/app/config"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/database"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/repository"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/services"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/logger"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	log := logger.NewLogrusLogger()
	configEnv, err := config.LoadConfig()
	if err != nil {
		log.Error("Error in config .env: %v", err)
		return
	}
	config.InitRedis()
	db := database.ConnectDatabase(configEnv)
	if db == nil {
		log.Error("Failed to connect to database")
		return
	}

	VendorRepo := repository.NewVendorRepository(db)

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
		vendorService := services.NewVendorService(VendorRepo)
		vendor.RegisterVendorSeviceServer(grpcServer, vendorService)
		log.Info("gRPC Server started on port 5004")
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("Failed to serve gRPC: %v", err)
		}

	}()

	router := gin.Default()
	log.Info("HTTP Server started on port 5003")
	router.Run(":5003")

	select {}

}
