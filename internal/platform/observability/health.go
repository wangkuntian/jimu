package observability

import (
	"jimu/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func Register(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	r.GET("/health", func(c *gin.Context) {
		dbStatus := "ok"
		if sqlDB, err := db.DB(); err != nil || sqlDB.Ping() != nil {
			dbStatus = "error"
		}

		redisStatus := "ok"
		if rdb != nil {
			if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
				redisStatus = "error"
			}
		}

		response.OK(c, gin.H{
			"status": "up",
			"db":     dbStatus,
			"redis":  redisStatus,
		})
	})
}
