package main

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

func getUser(c *gin.Context) (*User, bool) {
	u, ok := c.Get("user")
	if !ok {
		return nil, false
	}
	return u.(*User), true
}

func bindRequest(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": ErrValidation, "message": GetMessage(c, "error.invalid_request")})
		return false
	}
	return true
}

func handleServiceError(c *gin.Context, err error) {
	if se, ok := err.(*ServiceError); ok {
		switch se.Code {
		case ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"code": ErrNotFound, "message": GetMessage(c, "error.not_found")})
		case ErrUnauthorized:
			c.JSON(http.StatusUnauthorized, gin.H{"code": ErrUnauthorized, "message": GetMessage(c, "error.unauthorized")})
		case ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"code": ErrInvalidCredentials, "message": GetMessage(c, "error.invalid_credentials")})
		case ErrInvalidCurrentPassword:
			c.JSON(http.StatusBadRequest, gin.H{"code": ErrInvalidCurrentPassword, "message": GetMessage(c, "error.invalid_current_password")})
		case ErrSamePassword, ErrValidation:
			c.JSON(http.StatusBadRequest, gin.H{"code": se.Code, "message": GetMessage(c, "error.new_password_same")})
		case ErrConflict:
			c.JSON(http.StatusConflict, gin.H{"code": ErrConflict, "message": GetMessage(c, "error.conflict")})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": GetMessage(c, "error.internal")})
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"code": ErrInternal, "message": GetMessage(c, "error.internal")})
	}
}

func sendAuthResponse(c *gin.Context, user *User, tokens *JWTTokenPair) {
	c.JSON(http.StatusOK, gin.H{
		"user":          gin.H{"id": user.ID, "username": user.Username},
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

func GenerateGitHubOAuthURLHandler(c *gin.Context) {
	url, err := authSvc.GenerateGitHubAuthURL(c.Request.Context())
	if err != nil {
		handleServiceError(c, err)
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
		handleServiceError(c, err)
		return
	}
	sendAuthResponse(c, user, tokens)
}

type PostRequest struct {
	Title string `json:"title" binding:"required,titlesize"`
	Body  string `json:"body" binding:"required,bodysize"`
}

func CreatePostHandler(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": GetMessage(c, "error.failed_get_user")})
		return
	}

	var req PostRequest
	if !bindRequest(c, &req) {
		return
	}

	id, err := postSvc.CreatePost(c.Request.Context(), req.Title, req.Body, user.ID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func RenderPostHandler(c *gin.Context) {
	id := c.Param("id")
	title, html, err := postSvc.RenderPostHTML(c.Request.Context(), id)
	if err != nil {
		if se, ok := err.(*ServiceError); ok && se.Code == ErrNotFound {
			c.String(http.StatusNotFound, GetMessage(c, "error.not_found"))
		} else {
			c.String(http.StatusInternalServerError, GetMessage(c, "error.failed_render_post"))
		}
		return
	}
	c.HTML(http.StatusOK, "post.html", gin.H{"Title": title, "Body": template.HTML(html)})
}

func QueryPostKeyHandler(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": GetMessage(c, "error.failed_get_user")})
		return
	}

	postKey, createdAt, err := authSvc.QueryPostKey(c.Request.Context(), user.ID)
	if err != nil {
		handleServiceError(c, err)
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
	if !bindRequest(c, &req) {
		return
	}

	user, tokens, err := authSvc.LoginWithPassword(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	sendAuthResponse(c, user, tokens)
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func RefreshTokenHandler(c *gin.Context) {
	var req RefreshTokenRequest
	if !bindRequest(c, &req) {
		return
	}

	user, tokens, err := authSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	sendAuthResponse(c, user, tokens)
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

func ChangePasswordHandler(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": GetMessage(c, "error.failed_get_user")})
		return
	}

	var req PasswordChangeRequest
	if !bindRequest(c, &req) {
		return
	}

	if err := authSvc.ChangePassword(c.Request.Context(), user.ID, req.CurrentPassword, req.NewPassword); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": GetMessage(c, "error.password_changed_success")})
}

func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": GetMessage(c, "health.running")})
}

func PostsListHandler(c *gin.Context) {
	user, ok := getUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": GetMessage(c, "error.failed_get_user")})
		return
	}

	type queryParams struct {
		Page  int `form:"page" binding:"omitempty,min=1"`
		Limit int `form:"limit" binding:"omitempty,min=1"`
	}
	var query queryParams
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": ErrValidation, "message": GetMessage(c, "error.invalid_request")})
		return
	}

	query.Page = defaultInt(query.Page, 1)
	query.Limit = defaultInt(query.Limit, 20)
	if query.Limit > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"code": ErrValidation, "message": GetMessage(c, "error.invalid_request")})
		return
	}

	posts, total, err := postSvc.GetUserPostsPaginated(c.Request.Context(), user.ID, query.Page, query.Limit)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	items := make([]gin.H, 0, len(posts))
	for _, p := range posts {
		items = append(items, gin.H{
			"id":         p.ID,
			"title":      p.Title,
			"created_at": p.CreatedAt,
		})
	}
	totalPages := (total + int64(query.Limit) - 1) / int64(query.Limit)

	c.JSON(http.StatusOK, gin.H{
		"posts": items,
		"pagination": gin.H{
			"page":        query.Page,
			"limit":       query.Limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func defaultInt(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}
