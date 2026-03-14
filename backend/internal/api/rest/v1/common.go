// Package v1 provides REST API v1 handlers.
package v1

import (
	"reflect"
	"strings"
	"time"

	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ExtractUser extracts the authenticated user from the gin context.
func ExtractUser(c *gin.Context) (*user.User, bool) {
	u, ok := c.Get("user")
	if !ok {
		return nil, false
	}
	return u.(*user.User), true
}

func bindJSON(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		return writeBindingError(c, req, err)
	}
	return true
}

func bindQuery(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindQuery(req); err != nil {
		return writeBindingError(c, req, err)
	}
	return true
}

func writeBindingError(c *gin.Context, req interface{}, err error) bool {
	var causes []service.Error
	if ve, ok := err.(validator.ValidationErrors); ok {
		t := reflect.TypeOf(req)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		for _, fe := range ve {
			fieldName := fe.Field()
			jsonField := fieldName
			if t.Kind() == reflect.Struct {
				if sf, ok := t.FieldByName(fieldName); ok {
					tag := sf.Tag.Get("json")
					if tag == "" {
						tag = sf.Tag.Get("form")
					}
					if tag != "" {
						parts := strings.Split(tag, ",")
						if parts[0] != "" {
							jsonField = parts[0]
						}
					}
				}
			}
			tag := fe.Tag()
			var code service.ErrCode
			switch tag {
			case "required":
				code = service.ErrRequired
			case "min":
				code = service.ErrMinLength
			default:
				code = service.ErrFieldViolation
			}
			causes = append(causes, service.Error{
				Code:        code,
				Description: jsonField,
			})
		}
	}
	if len(causes) == 0 {
		causes = append(causes, service.Error{
			Code:        service.ErrFieldViolation,
			Description: "",
		})
	}

	errResp := &service.Error{
		Code:        service.ErrValidation,
		Description: "request validation failed",
		Details:     causes,
	}
	apierr.RespondError(c, errResp)
	return false
}

func defaultInt(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}

// AuthResponse represents an authentication response.
type AuthResponse struct {
	User         UserInfo `json:"user"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
}

// UserInfo represents user information in responses.
type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// MessageResponse represents a simple message response.
type MessageResponse struct {
	Message string `json:"message"`
}

// PostKeyResponse represents a post key response.
type PostKeyResponse struct {
	PostKey   string    `json:"post_key"`
	CreatedAt time.Time `json:"created_at"`
}

// CreatePostResponse represents a post creation response.
type CreatePostResponse struct {
	ID int `json:"id"`
}

// PostsListResponse represents a list of posts response.
type PostsListResponse struct {
	Posts      []post.Post `json:"posts"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination represents pagination information.
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
