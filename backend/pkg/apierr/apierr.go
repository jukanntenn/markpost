package apierr

import (
	"log"
	"net/http"

	"markpost/internal/service"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type FieldError struct {
	Field   string `json:"field,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Errors  []FieldError `json:"errors,omitempty"`
}

var errorCodeMessages = map[service.ErrCode]*i18n.Message{
	service.ErrInvalidCredentials: {
		ID:    "error.invalid_credentials",
		Other: "Invalid username or password",
	},
	service.ErrInvalidPassword: {
		ID:    "error.invalid_password",
		Other: "Current password is incorrect",
	},
	service.ErrNotFound: {
		ID:    "error.not_found",
		Other: "Not Found",
	},
	service.ErrUnauthorized: {
		ID:    "error.unauthorized",
		Other: "Unauthorized",
	},
	service.ErrFailedGetUser: {
		ID:    "error.failed_get_user",
		Other: "Failed to get user information",
	},
	service.ErrInternal: {
		ID:    "error.internal",
		Other: "Internal server error",
	},
	service.ErrValidation: {
		ID:    "error.validation_failed",
		Other: "Request validation failed",
	},
	service.ErrInvalidRequest: {
		ID:    "error.invalid_request",
		Other: "Invalid request format",
	},
	service.ErrMissingAuthorizationHeader: {
		ID:    "error.missing_authorization_header",
		Other: "Missing Authorization header",
	},
	service.ErrInvalidToken: {
		ID:    "error.invalid_token",
		Other: "Invalid token",
	},
	service.ErrInvalidPostKey: {
		ID:    "error.invalid_post_key",
		Other: "Invalid post key",
	},
	service.ErrForbidden: {
		ID:    "error.forbidden",
		Other: "Forbidden",
	},
	service.ErrRateLimited: {
		ID:    "error.rate_limited",
		Other: "Too many requests",
	},
	service.ErrUserDisabled: {
		ID:    "error.user_disabled",
		Other: "User account is disabled",
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

func RespondError(c *gin.Context, err error) {
	se, ok := service.AsServiceError(err)
	if !ok {
		log.Printf("unexpected error: %v", err)
		writeInternalError(c)
		return
	}

	message := resolveErrorMessage(c, se.Code)

	var fieldErrors []FieldError
	if se.Code == service.ErrValidation {
		fieldErrors = buildFieldErrors(c, se.Details)
	}

	c.JSON(se.HTTPStatus(), ErrorResponse{
		Code:    string(se.Code),
		Message: message,
		Errors:  fieldErrors,
	})
}

func resolveErrorMessage(c *gin.Context, code service.ErrCode) string {
	msg, ok := errorCodeMessages[code]
	if !ok {
		msg = errorCodeMessages[service.ErrInternal]
	}
	return ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
		DefaultMessage: msg,
	})
}

func buildFieldErrors(c *gin.Context, causes []service.FieldDetail) []FieldError {
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
	message := resolveErrorMessage(c, service.ErrInternal)
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    string(service.ErrInternal),
		Message: message,
	})
}
