package middleware

import (
	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// PostKey returns a middleware that validates post key.
func PostKey(users user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		postKey := c.Param("post_key")
		if postKey == "" {
			abortWithError(c, service.New(auth.ErrInvalidPostKey, "post key is required"))
			return
		}

		u, err := users.GetByPostKey(c.Request.Context(), postKey)
		if err != nil {
			abortWithError(c, service.Wrap(auth.ErrInvalidPostKey, "invalid post key", err))
			return
		}

		// C3.1 漏洞补正（关键）：封禁必须切断发帖——被封禁用户的 post_key
		// 即使不换也立即失效。封禁即时性闭环：登录挡住 + 发帖挡住 +
		// token version 即时失效已存 session。
		if !u.IsActive {
			abortWithError(c, service.New(auth.ErrUserDisabled, "user is disabled"))
			return
		}

		setUserFields(c, u)
		c.Next()
	}
}
