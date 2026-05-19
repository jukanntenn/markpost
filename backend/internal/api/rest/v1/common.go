// Package v1 provides REST API v1 handlers.
package v1

import (
	"context"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"markpost/internal/domain/user"
	"markpost/internal/middleware"
	"markpost/internal/service"
	"markpost/pkg/apierr"
	"markpost/pkg/utils"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func withUser(c *gin.Context, fn func(*user.User)) {
	u, ok := requireUser(c)
	if !ok {
		return
	}
	fn(u)
}

func withUserAndID(c *gin.Context, fn func(*user.User, int)) {
	u, ok := requireUser(c)
	if !ok {
		return
	}
	id, ok := parsePathID(c)
	if !ok {
		return
	}
	fn(u, id)
}

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

func bindAndValidatePagination(c *gin.Context, q *PaginationQuery) bool {
	if err := c.ShouldBindQuery(q); err != nil {
		writeBindingError(c, q, err)
		return false
	}
	return validatePaginationQuery(c, q)
}

func validatePaginationQuery(c *gin.Context, q *PaginationQuery) bool {
	if err := q.Validate(); err != nil {
		apierr.RespondError(c, err)
		return false
	}
	return true
}

func resolveFieldName(t reflect.Type, fieldName string) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
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

func ParseBindingErrors(err error, req interface{}) []service.FieldDetail {
	var causes []service.FieldDetail
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return []service.FieldDetail{{
			Code:        service.ErrFieldViolation,
			Description: "",
		}}
	}

	t := reflect.TypeOf(req)
	if t.Kind() == reflect.Pointer {
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
		causes = append(causes, service.FieldDetail{
			Code:        code,
			Description: jsonField,
		})
	}
	return causes
}

func writeBindingError(c *gin.Context, req interface{}, err error) {
	causes := ParseBindingErrors(err, req)
	apierr.RespondError(c, service.NewBindingError(causes))
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

func parsePathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		apierr.RespondError(c, service.NewServiceError(service.ErrValidation, "invalid ID"))
		return 0, false
	}
	return id, true
}

func getI18nMessage(c *gin.Context, defaultMsg string, msgID ...string) string {
	id := defaultMsg
	if len(msgID) > 0 {
		id = msgID[0]
	}
	return ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    id,
			Other: defaultMsg,
		},
	})
}

func bindPaginationQuery(c *gin.Context) (PaginationQuery, bool) {
	var q PaginationQuery
	if !bindAndValidatePagination(c, &q) {
		return PaginationQuery{}, false
	}
	return q, true
}

func bindAdminPostsQuery(c *gin.Context) (string, PaginationQuery, bool) {
	var q AdminPostsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		writeBindingError(c, &q, err)
		return "", PaginationQuery{}, false
	}
	if !validatePaginationQuery(c, &q.PaginationQuery) {
		return "", PaginationQuery{}, false
	}
	return q.Search, q.PaginationQuery, true
}

func withUserPaginatedQuery[T any, R any](
	c *gin.Context,
	fetch func(ctx context.Context, userID, offset, limit int) ([]T, int64, error),
	mapper func(T) R,
	wrap func([]R, Pagination) any,
) {
	u, ok := requireUser(c)
	if !ok {
		return
	}
	query, ok := bindPaginationQuery(c)
	if !ok {
		return
	}
	items, total, err := fetch(c.Request.Context(), u.ID, query.Offset, query.Limit)
	if err != nil {
		apierr.RespondError(c, err)
		return
	}
	writePaginatedList(c, items, total, query, mapper, wrap)
}

func handleSearchPaginatedQuery[T any, R any](
	c *gin.Context,
	bind func(*gin.Context) (string, PaginationQuery, bool),
	fetch func(ctx context.Context, search string, offset, limit int) ([]T, int64, error),
	mapper func(T) R,
	wrap func([]R, Pagination) any,
) {
	search, query, ok := bind(c)
	if !ok {
		return
	}
	items, total, err := fetch(c.Request.Context(), search, query.Offset, query.Limit)
	if err != nil {
		apierr.RespondError(c, err)
		return
	}
	writePaginatedList(c, items, total, query, mapper, wrap)
}

func handlePaginatedQuery[T any, R any](
	c *gin.Context,
	bind func(*gin.Context) (PaginationQuery, bool),
	fetch func(ctx context.Context, offset, limit int) ([]T, int64, error),
	mapper func(T) R,
	wrap func([]R, Pagination) any,
) {
	query, ok := bind(c)
	if !ok {
		return
	}
	items, total, err := fetch(c.Request.Context(), query.Offset, query.Limit)
	if err != nil {
		apierr.RespondError(c, err)
		return
	}
	writePaginatedList(c, items, total, query, mapper, wrap)
}

func writeList[T any, R any](
	c *gin.Context,
	items []T,
	mapper func(T) R,
	wrapResponse func([]R) any,
) {
	mapped := utils.MapSlice(items, mapper)
	c.JSON(http.StatusOK, wrapResponse(mapped))
}

func NotFound() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, MessageResponse{Message: "Not Found"})
	}
}

func paginatedResponse(key string, items any, p Pagination) map[string]any {
	return map[string]any{key: items, "pagination": p}
}

func paginatedWrap[T any](key string) func([]T, Pagination) any {
	return func(items []T, p Pagination) any {
		return paginatedResponse(key, items, p)
	}
}

func writePaginatedList[T any, R any](
	c *gin.Context,
	items []T,
	total int64,
	query PaginationQuery,
	mapper func(T) R,
	wrapResponse func([]R, Pagination) any,
) {
	mapped := utils.MapSlice(items, mapper)
	c.JSON(http.StatusOK, wrapResponse(mapped, query.ToPagination(total)))
}
