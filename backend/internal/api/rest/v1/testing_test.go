package v1

import (
	"fmt"

	"markpost/internal/domain/user"
	"markpost/internal/testutil"

	"github.com/gin-gonic/gin"
)

func newTestEngine() *gin.Engine {
	return testutil.NewTestEngine(testutil.TestEngineConfig{
		LocalesPath: "../../../../locales",
	})
}

func withUser(userID int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user", &user.User{
			ID:       userID,
			Email:    fmt.Sprintf("user%d@example.com", userID),
			Username: fmt.Sprintf("user%d", userID),
		})
		c.Next()
	}
}
