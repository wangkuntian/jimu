package v1

import (
	"net/http"

	"example.com/upgrade-basic-api/internal/shared/response"

	"github.com/gin-gonic/gin"
)

func Ping(c *gin.Context) {
	response.OK(c, http.StatusOK, gin.H{"message": "pong"})
}
