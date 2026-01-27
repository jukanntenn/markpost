package handlers

import (
	"net/http"
	"strconv"

	"markpost/models"
	"markpost/services"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	authSvc services.AuthServiceInterface
}

func NewUserHandler(authSvc services.AuthServiceInterface) *UserHandler {
	return &UserHandler{authSvc: authSvc}
}

type ListUsersResponse struct {
	Users     []models.User `json:"users"`
	Total     int64         `json:"total"`
	Page      int           `json:"page"`
	PageSize  int           `json:"page_size"`
}

func (h *UserHandler) ListAllUsers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "invalid_page",
			"message": "invalid page parameter",
		})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "invalid_limit",
			"message": "invalid limit parameter",
		})
		return
	}

	users, total, err := h.authSvc.GetAllUsers(page, limit)
	if err != nil {
		services.AsServiceError(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "internal_error",
			"message": "failed to get users",
		})
		return
	}

	c.JSON(http.StatusOK, ListUsersResponse{
		Users:     users,
		Total:     total,
		Page:      page,
		PageSize:  limit,
	})
}
