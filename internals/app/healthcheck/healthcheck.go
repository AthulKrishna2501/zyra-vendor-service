package healthcheck

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var rdb *redis.Client
var db *gorm.DB

type HealthCheckResponse struct {
	Status   string   `json:"status"`
	Services []string `json:"services"`
}

func checkPostgres() string {
	if err := db.Raw("SELECT 1").Error; err != nil {
		return "Postgres is not healthy"
	}
	return "Postgres is healthy"
}

func checkRedis() string {
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return "Redis is not healthy"
	}
	return "Redis is healthy"
}

func HealthCheckHandler(c *gin.Context) {
	services := []string{
		checkPostgres(),
		checkRedis(),
	}

	response := HealthCheckResponse{
		Status:   "Vendor Service is healthy!",
		Services: services,
	}

	c.JSON(http.StatusOK, response)
}
