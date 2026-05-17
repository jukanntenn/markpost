// Package v1 provides REST API v1 handlers.
package v1

import (
	"reflect"
	"strings"

	"markpost/internal/domain/user"
	"markpost/internal/middleware"
	"markpost/internal/service"
	"markpost/pkg/apierr"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func requireUser(c *gin.Context) (*user.User, bool) {
	u, ok := middleware.ExtractUser(c)
	if !ok {
		err := service.NewServiceError(service.ErrFailedGetUser, "failed to get user from context")
		apierr.RespondError(c, err)
		return nil, false
	}
	return u, true
}

func bindJSON(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		writeBindingError(c, req, err)
		return false
	}
	return true
}

func bindQuery(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindQuery(req); err != nil {
		writeBindingError(c, req, err)
		return false
	}
	return true
}

func bindPaginationQuery(c *gin.Context) (PaginationQuery, bool) {
	var query PaginationQuery
	if !bindQuery(c, &query) {
		return query, false
	}
	if !validatePaginationQuery(c, &query) {
		return query, false
	}
	return query, true
}

func validatePaginationQuery(c *gin.Context, q *PaginationQuery) bool {
	if err := q.Validate(); err != nil {
		apierr.RespondError(c, err)
		return false
	}
	return true
}

func resolveFieldName(t reflect.Type, fieldName string) string {
	if t.Kind() != reflect.Struct {
		return fieldName
	}
	sf, ok := t.FieldByName(fieldName)
	if !ok {
		return fieldName
	}
	tag := sf.Tag.Get("json")
	if tag == "" {
		tag = sf.Tag.Get("form")
	}
	if tag == "" {
		return fieldName
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" {
		return fieldName
	}
	return name
}

func ParseBindingErrors(err error, req interface{}) []service.Error {
	var causes []service.Error
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return []service.Error{{
			Code:        service.ErrFieldViolation,
			Description: "",
		}}
	}

	t := reflect.TypeOf(req)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for _, fe := range ve {
		jsonField := resolveFieldName(t, fe.Field())
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
	if len(causes) == 0 {
		causes = append(causes, service.Error{
			Code:        service.ErrFieldViolation,
			Description: "",
		})
	}
	return causes
}

func writeBindingError(c *gin.Context, req interface{}, err error) {
	causes := ParseBindingErrors(err, req)
	errResp := &service.Error{
		Code:        service.ErrValidation,
		Description: "request validation failed",
		Details:     causes,
	}
	apierr.RespondError(c, errResp)
}

func deref[T any](s *T) T {
	if s == nil {
		var zero T
		return zero
	}
	return *s
}

func mapSlice[T any, R any](src []T, fn func(T) R) []R {
	result := make([]R, 0, len(src))
	for _, item := range src {
		result = append(result, fn(item))
	}
	return result
}

// MessageResponse represents a simple message response.
type MessageResponse struct {
	Message string `json:"message"`
}

// Pagination represents pagination information.
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// PaginationQuery binds pagination query parameters from the request.
type PaginationQuery struct {
	Page   int `form:"page" binding:"omitempty,min=1"`
	Limit  int `form:"limit" binding:"omitempty,min=1"`
	Offset int `form:"-" json:"-"`
}

func (q *PaginationQuery) Validate() error {
	offset, page, limit, err := service.ValidatePagination(q.Page, q.Limit)
	if err != nil {
		return err
	}
	q.Page = page
	q.Limit = limit
	q.Offset = offset
	return nil
}

func (q *PaginationQuery) ToPagination(total int64) Pagination {
	return Pagination{
		Page:       q.Page,
		Limit:      q.Limit,
		Total:      int(total),
		TotalPages: service.CalcTotalPages(total, q.Limit),
	}
}

func getI18nMessage(c *gin.Context, defaultMsg string, msgID ...string) string {
	id := defaultMsg
	if len(msgID) > 0 {
		id = msgID[0]
	}
	if _, exists := c.Get("i18n"); exists {
		return ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    id,
				Other: defaultMsg,
			},
		})
	}
	return defaultMsg
}
