package errors

import (
	"log"
	"net/http"

	"markpost/services"

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

type serviceErrorMapping struct {
	Status  int
	Message *i18n.Message
}

var serviceErrorMappings = map[services.ErrCode]serviceErrorMapping{
	services.ErrInvalidCredentials: {
		Status: http.StatusUnauthorized,
		Message: &i18n.Message{
			ID:    "error.invalid_credentials",
			Other: "Invalid username or password",
		},
	},
	services.ErrInvalidPassword: {
		Status: http.StatusBadRequest,
		Message: &i18n.Message{
			ID:    "error.invalid_current_password",
			Other: "Current password is incorrect",
		},
	},
	services.ErrNotFound: {
		Status: http.StatusNotFound,
		Message: &i18n.Message{
			ID:    "error.not_found",
			Other: "Not Found",
		},
	},
	services.ErrUnauthorized: {
		Status: http.StatusUnauthorized,
		Message: &i18n.Message{
			ID:    "error.unauthorized",
			Other: "Unauthorized",
		},
	},
	services.ErrFailedGetUser: {
		Status: http.StatusInternalServerError,
		Message: &i18n.Message{
			ID:    "error.failed_get_user",
			Other: "Failed to get user information",
		},
	},
	services.ErrInternal: {
		Status: http.StatusInternalServerError,
		Message: &i18n.Message{
			ID:    "error.internal",
			Other: "Internal server error",
		},
	},
	services.ErrValidation: {
		Status: http.StatusBadRequest,
		Message: &i18n.Message{
			ID:    "error.validation_failed",
			Other: "Request validation failed",
		},
	},
	services.ErrMissingStateParam: {
		Status: http.StatusBadRequest,
		Message: &i18n.Message{
			ID:    "error.missing_state_param",
			Other: "Missing state query parameter",
		},
	},
	services.ErrMissingCode: {
		Status: http.StatusBadRequest,
		Message: &i18n.Message{
			ID:    "error.missing_code",
			Other: "Missing code field",
		},
	},
	services.ErrInvalidRequest: {
		Status: http.StatusBadRequest,
		Message: &i18n.Message{
			ID:    "error.invalid_request",
			Other: "Invalid request format",
		},
	},
	services.ErrMissingAuthorizationHeader: {
		Status: http.StatusUnauthorized,
		Message: &i18n.Message{
			ID:    "error.missing_authorization_header",
			Other: "Missing Authorization header",
		},
	},
	services.ErrInvalidToken: {
		Status: http.StatusUnauthorized,
		Message: &i18n.Message{
			ID:    "error.invalid_token",
			Other: "Invalid token",
		},
	},
	services.ErrInvalidPostKey: {
		Status: http.StatusForbidden,
		Message: &i18n.Message{
			ID:    "error.invalid_post_key",
			Other: "Invalid post key",
		},
	},
}

var validationFieldMessages = map[services.ErrCode]*i18n.Message{
	services.ErrRequired: {
		ID:    "error.validation_required",
		Other: "This field is required",
	},
	services.ErrMinLength: {
		ID:    "error.validation_min_length",
		Other: "Value does not meet minimum length",
	},
	services.ErrFieldViolation: {
		ID:    "error.invalid_request",
		Other: "Invalid request format",
	},
}

func RespondError(c *gin.Context, err error) {
	se, ok := services.AsServiceError(err)
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
	if se.Code == services.ErrValidation {
		fieldErrors = buildFieldErrors(c, se.Details)
	}

	c.JSON(mapping.Status, ErrorResponse{
		Code:    string(se.Code),
		Message: message,
		Errors:  fieldErrors,
	})
}

func buildFieldErrors(c *gin.Context, causes []services.ServiceError) []FieldError {
	if len(causes) == 0 {
		return nil
	}

	fieldErrors := make([]FieldError, 0, len(causes))
	for _, cause := range causes {
		messageTemplate, ok := validationFieldMessages[cause.Code]
		if !ok {
			messageTemplate = validationFieldMessages[services.ErrFieldViolation]
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
		DefaultMessage: serviceErrorMappings[services.ErrInternal].Message,
	})

	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    string(services.ErrInternal),
		Message: message,
	})
}
