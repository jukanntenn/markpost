package main

import (
    "html/template"
    "net/http"

    "github.com/gin-gonic/gin"
)

func GenerateGitHubOAuthURLHandler(c *gin.Context) {
	url, err := authSvc.GenerateGitHubAuthURL(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": GetMessage(c, "error.internal")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func LoginGitHubHandler(c *gin.Context) {
	var header struct {
		XOAuthState string `header:"X-Oauth-State" binding:"required"`
	}
	if err := c.ShouldBindHeader(&header); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": ErrValidation, "message": GetMessage(c, "error.missing_oauth_state")})
		return
	}
	var query struct {
		State string `form:"state" binding:"required"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": ErrValidation, "message": GetMessage(c, "error.missing_state_param")})
		return
	}
	var body struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": ErrValidation, "message": GetMessage(c, "error.missing_code")})
		return
	}
	if header.XOAuthState != query.State {
		c.JSON(http.StatusBadRequest, gin.H{"code": "mismatch", "message": GetMessage(c, "error.state_mismatch")})
		return
	}

	user, tokens, err := authSvc.LoginWithGitHub(c.Request.Context(), body.Code)
	if err != nil {
		if se, ok := err.(*ServiceError); ok && se.Code == ErrUnauthorized {
			c.JSON(http.StatusUnauthorized, gin.H{"code": ErrUnauthorized, "message": GetMessage(c, "error.unauthorized")})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": GetMessage(c, "error.internal")})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": GetMessage(c, "error.failed_get_user")})
		return
	}
	user := u.(*User)

	var req PostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": GetMessage(c, "error.invalid_request")})
		return
	}

	id, err := postSvc.CreatePost(c.Request.Context(), req.Title, req.Body, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": GetMessage(c, "error.failed_create_post")})
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
				c.String(http.StatusNotFound, GetMessage(c, "error.not_found"))
			default:
				c.String(http.StatusInternalServerError, GetMessage(c, "error.internal"))
			}
		} else {
			c.String(http.StatusInternalServerError, GetMessage(c, "error.failed_render_post"))
		}
		return
	}
	c.HTML(http.StatusOK, "post.html", gin.H{"Title": title, "Body": template.HTML(html)})
}

func QueryPostKeyHandler(c *gin.Context) {
	u, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": GetMessage(c, "error.failed_get_user")})
		return
	}
	user := u.(*User)

	postKey, createdAt, err := authSvc.QueryPostKey(c.Request.Context(), user.ID)
	if err != nil {
		if se, ok := err.(*ServiceError); ok && se.Code == ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": GetMessage(c, "error.not_found")})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": GetMessage(c, "error.internal")})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "message": GetMessage(c, "error.invalid_request")})
		return
	}
	user, tokens, err := authSvc.LoginWithPassword(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if se, ok := err.(*ServiceError); ok && se.Code == ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"code": ErrInvalidCredentials, "message": GetMessage(c, "error.invalid_credentials")})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": GetMessage(c, "error.internal")})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user":          gin.H{"id": user.ID, "username": user.Username},
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

type RefreshTokenRequest struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}

func RefreshTokenHandler(c *gin.Context) {
    var req RefreshTokenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "message": GetMessage(c, "error.invalid_request")})
        return
    }

    user, tokens, err := authSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
    if err != nil {
        if se, ok := err.(*ServiceError); ok {
            switch se.Code {
            case ErrUnauthorized:
                c.JSON(http.StatusUnauthorized, gin.H{"code": ErrUnauthorized, "message": GetMessage(c, "error.invalid_token")})
            case ErrNotFound:
                c.JSON(http.StatusNotFound, gin.H{"code": ErrNotFound, "message": GetMessage(c, "error.not_found")})
            default:
                c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": GetMessage(c, "error.internal")})
            }
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": GetMessage(c, "error.internal")})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": GetMessage(c, "error.failed_get_user")})
		return
	}
	user := u.(*User)

	var req PasswordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "message": GetMessage(c, "error.invalid_request")})
		return
	}

	if err := authSvc.ChangePassword(c.Request.Context(), user.ID, req.CurrentPassword, req.NewPassword); err != nil {
		if se, ok := err.(*ServiceError); ok {
			switch se.Code {
			case ErrInvalidCurrentPassword:
				c.JSON(http.StatusBadRequest, gin.H{"code": ErrInvalidCurrentPassword, "message": GetMessage(c, "error.invalid_current_password")})
			case ErrSamePassword, ErrValidation:
				c.JSON(http.StatusBadRequest, gin.H{"code": se.Code, "message": GetMessage(c, "error.new_password_same")})
			case ErrNotFound:
				c.JSON(http.StatusNotFound, gin.H{"code": ErrNotFound, "message": GetMessage(c, "error.not_found")})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": GetMessage(c, "error.internal")})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": GetMessage(c, "error.internal")})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": GetMessage(c, "error.password_changed_success")})
}

func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": GetMessage(c, "health.running")})
}
