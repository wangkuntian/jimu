//go:build jimu_lean

package middleware

import "github.com/gin-gonic/gin"

// GlobalRateLimit in a lean build is a no-op: the token bucket pulls in
// golang.org/x/time/rate, which a zero-capability service does not need.
func GlobalRateLimit(int, int) gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}
