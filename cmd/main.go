package main

import (
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/app/config"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/app/grpc"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/database"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/core/repository"
	"github.com/AthulKrishna2501/zyra-vendor-service/internals/logger"
	"github.com/gin-gonic/gin"
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

	err = grpc.StartgRPCServer(VendorRepo, log, configEnv)

	if err != nil {
		log.Error("Faile to start gRPC server", err)
		return
	}

	router := gin.Default()
	log.Info("HTTP Server started on port 3004")
	router.Run(":3004")

}
