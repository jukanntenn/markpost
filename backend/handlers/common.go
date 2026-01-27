package handlers

import (
	"net/http"
	"reflect"
	"strings"
	"time"

	apperrors "markpost/errors"
	"markpost/models"
	"markpost/services"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func ExtractUser(c *gin.Context) (*models.User, bool) {
	u, ok := c.Get("user")
	if !ok {
		return nil, false
	}
	return u.(*models.User), true
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
	var causes []services.ServiceError
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
			var code services.ErrCode
			switch tag {
			case "required":
				code = services.ErrRequired
			case "min":
				code = services.ErrMinLength
			default:
				code = services.ErrFieldViolation
			}
			causes = append(causes, services.ServiceError{
				Code:        code,
				Description: jsonField,
			})
		}
	}
	if len(causes) == 0 {
		causes = append(causes, services.ServiceError{
			Code:        services.ErrFieldViolation,
			Description: "",
		})
	}

	errResp := &services.ServiceError{
		Code:        services.ErrValidation,
		Description: "request validation failed",
		Details:     causes,
	}
	apperrors.RespondError(c, errResp)
	return false
}

func writeAuthResponse(c *gin.Context, user *models.User, tokens *services.JWTTokenPair) {
	c.JSON(http.StatusOK, gin.H{
		"user":          gin.H{"id": user.ID, "username": user.Username, "role": string(user.Role)},
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

func defaultInt(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}

type AuthResponse struct {
	User         UserInfo `json:"user"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
}

type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type PostKeyResponse struct {
	PostKey   string    `json:"post_key"`
	CreatedAt time.Time `json:"created_at"`
}

type CreatePostResponse struct {
	ID int `json:"id"`
}

type PostsListResponse struct {
	Posts      []models.Post `json:"posts"`
	Pagination Pagination    `json:"pagination"`
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
