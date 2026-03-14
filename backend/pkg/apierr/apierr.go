// Package apierr provides API error handling utilities.
package apierr

import (
	"log"
	"net/http"

	"markpost/internal/service"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// FieldError represents a validation field error.
type FieldError struct {
	Field   string `json:"field,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Errors  []FieldError `json:"errors,omitempty"`
}

// ServiceErrorMapping maps service error codes to HTTP status and i18n messages.
type serviceErrorMapping struct {
	Status  int
	Message *i18n.Message
}

var serviceErrorMappings = map[service.ErrCode]serviceErrorMapping{
	service.ErrInvalidCredentials: {
		Status: http.StatusUnauthorized,
		Message: &i18n.Message{
			ID:    "error.invalid_credentials",
			Other: "Invalid username or password",
		},
	},
	service.ErrInvalidPassword: {
		Status: http.StatusBadRequest,
		Message: &i18n.Message{
			ID:    "error.invalid_current_password",
			Other: "Current password is incorrect",
		},
	},
	service.ErrNotFound: {
		Status: http.StatusNotFound,
		Message: &i18n.Message{
			ID:    "error.not_found",
			Other: "Not Found",
		},
	},
	service.ErrUnauthorized: {
		Status: http.StatusUnauthorized,
		Message: &i18n.Message{
			ID:    "error.unauthorized",
			Other: "Unauthorized",
		},
	},
	service.ErrFailedGetUser: {
		Status: http.StatusInternalServerError,
		Message: &i18n.Message{
			ID:    "error.failed_get_user",
			Other: "Failed to get user information",
		},
	},
	service.ErrInternal: {
		Status: http.StatusInternalServerError,
		Message: &i18n.Message{
			ID:    "error.internal",
			Other: "Internal server error",
		},
	},
	service.ErrValidation: {
		Status: http.StatusBadRequest,
		Message: &i18n.Message{
			ID:    "error.validation_failed",
			Other: "Request validation failed",
		},
	},
	service.ErrMissingStateParam: {
		Status: http.StatusBadRequest,
		Message: &i18n.Message{
			ID:    "error.missing_state_param",
			Other: "Missing state query parameter",
		},
	},
	service.ErrMissingCode: {
		Status: http.StatusBadRequest,
		Message: &i18n.Message{
			ID:    "error.missing_code",
			Other: "Missing code field",
		},
	},
	service.ErrInvalidRequest: {
		Status: http.StatusBadRequest,
		Message: &i18n.Message{
			ID:    "error.invalid_request",
			Other: "Invalid request format",
		},
	},
	service.ErrMissingAuthorizationHeader: {
		Status: http.StatusUnauthorized,
		Message: &i18n.Message{
			ID:    "error.missing_authorization_header",
			Other: "Missing Authorization header",
		},
	},
	service.ErrInvalidToken: {
		Status: http.StatusUnauthorized,
		Message: &i18n.Message{
			ID:    "error.invalid_token",
			Other: "Invalid token",
		},
	},
	service.ErrInvalidPostKey: {
		Status: http.StatusForbidden,
		Message: &i18n.Message{
			ID:    "error.invalid_post_key",
			Other: "Invalid post key",
		},
	},
}

var validationFieldMessages = map[service.ErrCode]*i18n.Message{
	service.ErrRequired: {
		ID:    "error.validation_required",
		Other: "This field is required",
	},
	service.ErrMinLength: {
		ID:    "error.validation_min_length",
		Other: "Value does not meet minimum length",
	},
	service.ErrFieldViolation: {
		ID:    "error.invalid_request",
		Other: "Invalid request format",
	},
}

// RespondError writes an error response to the gin context.
func RespondError(c *gin.Context, err error) {
	se, ok := service.AsServiceError(err)
	if !ok {
		log.Printf("unexpected error: %v", err)
		writeInternalError(c)
		return
	}

	mapping, ok := serviceErrorMappings[se.Code]
	if !ok {
		log.Printf("unknown service error code: %s detail=%s err=%v", se.Code, se.Description, se.Err)
		writeInternalError(c)
		return
	}

	message := ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
		DefaultMessage: mapping.Message,
	})

	var fieldErrors []FieldError
	if se.Code == service.ErrValidation {
		fieldErrors = buildFieldErrors(c, se.Details)
	}

	c.JSON(mapping.Status, ErrorResponse{
		Code:    string(se.Code),
		Message: message,
		Errors:  fieldErrors,
	})
}

func buildFieldErrors(c *gin.Context, causes []service.Error) []FieldError {
	if len(causes) == 0 {
		return nil
	}

	fieldErrors := make([]FieldError, 0, len(causes))
	for _, cause := range causes {
		messageTemplate, ok := validationFieldMessages[cause.Code]
		if !ok {
			messageTemplate = validationFieldMessages[service.ErrFieldViolation]
		}

		message := ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
			DefaultMessage: messageTemplate,
		})

		fieldErrors = append(fieldErrors, FieldError{
			Field:   cause.Description,
			Code:    string(cause.Code),
			Message: message,
		})
	}

	return fieldErrors
}

func writeInternalError(c *gin.Context) {
	message := ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
		DefaultMessage: serviceErrorMappings[service.ErrInternal].Message,
	})

	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    string(service.ErrInternal),
		Message: message,
	})
}
