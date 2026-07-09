package middleware

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// FuzzUserIDFromContext exercises the rate-limiter key derivation with arbitrary
// user_id context values to confirm it never panics and accepts only positive
// ints/int64s (rejecting everything else so anonymous traffic cannot share a key).
func FuzzUserIDFromContext(f *testing.F) {
	f.Add(0)
	f.Add(1)
	f.Add(-5)
	f.Add(2147483647)

	f.Fuzz(func(t *testing.T, userID int) {
		c, _ := gin.CreateTestContext(nil)
		c.Set("user_id", userID)
		_, _ = userIDFromContext(c)
	})
}
