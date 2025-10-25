package main

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GenerateGitHubOAuthURLHandler(c *gin.Context) {
	url, err := authSvc.GenerateGitHubAuthURL(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func LoginGitHubHandler(c *gin.Context) {
	var header struct {
		XOAuthState string `header:"X-Oauth-State" binding:"required"`
	}
	if err := c.ShouldBindHeader(&header); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": ErrValidation, "message": "Missing X-OAuth-State header"})
		return
	}
	var query struct {
		State string `form:"state" binding:"required"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": ErrValidation, "message": "Missing state query parameter"})
		return
	}
	var body struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": ErrValidation, "message": "Missing code field"})
		return
	}
	if header.XOAuthState != query.State {
		c.JSON(http.StatusBadRequest, gin.H{"code": "mismatch", "message": "State mismatch"})
		return
	}

	user, tokens, err := authSvc.LoginWithGitHub(c.Request.Context(), body.Code)
	if err != nil {
		if se, ok := err.(*ServiceError); ok && se.Code == ErrUnauthorized {
			c.JSON(http.StatusUnauthorized, gin.H{"code": ErrUnauthorized, "message": "Unauthorized"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":          gin.H{"id": user.ID, "username": user.Username},
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

type PostRequest struct {
	Title string `json:"title" binding:"required,titlesize"`
	Body  string `json:"body" binding:"required,bodysize"`
}

func CreatePostHandler(c *gin.Context) {
	u, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user information"})
		return
	}
	user := u.(*User)

	var req PostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	id, err := postSvc.CreatePost(c.Request.Context(), req.Title, req.Body, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func RenderPostHandler(c *gin.Context) {
	id := c.Param("id")
	title, html, err := postSvc.RenderPostHTML(c.Request.Context(), id)
	if err != nil {
		if se, ok := err.(*ServiceError); ok {
			switch se.Code {
			case ErrNotFound:
				c.String(http.StatusNotFound, "Not Found")
			default:
				c.String(http.StatusInternalServerError, "Internal Server Error")
			}
		} else {
			c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}
	c.HTML(http.StatusOK, "post.html", gin.H{"Title": title, "Body": template.HTML(html)})
}

func QueryPostKeyHandler(c *gin.Context) {
	u, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user information"})
		return
	}
	user := u.(*User)

	postKey, createdAt, err := authSvc.QueryPostKey(c.Request.Context(), user.ID)
	if err != nil {
		if se, ok := err.(*ServiceError); ok && se.Code == ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"post_key": postKey, "created_at": createdAt})
}

type PasswordLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func LoginWithPasswordHandler(c *gin.Context) {
	var req PasswordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "message": "Invalid request format"})
		return
	}
	user, tokens, err := authSvc.LoginWithPassword(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if se, ok := err.(*ServiceError); ok && se.Code == ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"code": ErrInvalidCredentials, "message": "Invalid username or password"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": "Internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user":          gin.H{"id": user.ID, "username": user.Username},
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

func ChangePasswordHandler(c *gin.Context) {
	u, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user information"})
		return
	}
	user := u.(*User)

	var req PasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "message": "Invalid request format"})
		return
	}

	if err := authSvc.ChangePassword(c.Request.Context(), user.ID, req.CurrentPassword, req.NewPassword); err != nil {
		if se, ok := err.(*ServiceError); ok {
			switch se.Code {
			case ErrInvalidCurrentPassword:
				c.JSON(http.StatusUnauthorized, gin.H{"code": ErrInvalidCurrentPassword, "message": "Current password is incorrect"})
			case ErrSamePassword, ErrValidation:
				c.JSON(http.StatusBadRequest, gin.H{"code": se.Code, "message": "New password cannot be the same as current password"})
			case ErrNotFound:
				c.JSON(http.StatusNotFound, gin.H{"code": ErrNotFound, "message": "Not Found"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": "Internal server error"})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "markpost is running"})
}