package auth

import (
	"strconv"

	"markpost/internal/service"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Authentication-domain error codes. Defined here per error-handling.md's
// "域专属码分文件" principle; each is a *service.ErrCode singleton carrying its
// own HTTP status and i18n message so the auth domain is fully self-contained.

var (
	ErrInvalidCredentials = &service.ErrCode{
		Value:   "invalid_credentials",
		HTTP:    401,
		Message: &i18n.Message{ID: "error.invalid_credentials", Other: "Invalid username or password"},
	}
	ErrInvalidPassword = &service.ErrCode{
		Value:   "invalid_password",
		HTTP:    401,
		Message: &i18n.Message{ID: "error.invalid_password", Other: "Current password is incorrect"},
	}
	ErrUserDisabled = &service.ErrCode{
		Value:   "user_disabled",
		HTTP:    403,
		Message: &i18n.Message{ID: "error.user_disabled", Other: "User account is disabled"},
	}
	ErrInvalidToken = &service.ErrCode{
		Value:   "invalid_token",
		HTTP:    401,
		Message: &i18n.Message{ID: "error.invalid_token", Other: "Invalid or expired token"},
	}
	ErrInvalidPostKey = &service.ErrCode{
		Value:   "invalid_post_key",
		HTTP:    403,
		Message: &i18n.Message{ID: "error.invalid_post_key", Other: "Invalid post key"},
	}

	// OAuth (GitHub) error codes — auth.md §3.8.
	ErrMissingState = &service.ErrCode{
		Value:   "missing_state",
		HTTP:    400,
		Message: &i18n.Message{ID: "error.missing_state", Other: "OAuth state parameter is missing"},
	}
	ErrMissingCode = &service.ErrCode{
		Value:   "missing_code",
		HTTP:    400,
		Message: &i18n.Message{ID: "error.missing_code", Other: "OAuth code parameter is missing"},
	}
	ErrInvalidState = &service.ErrCode{
		Value:   "invalid_state",
		HTTP:    401,
		Message: &i18n.Message{ID: "error.invalid_state", Other: "OAuth state is invalid, expired, or already used"},
	}
	ErrOAuthExchangeFailed = &service.ErrCode{
		Value:   "oauth_exchange_failed",
		HTTP:    401,
		Message: &i18n.Message{ID: "error.oauth_exchange_failed", Other: "OAuth authorization exchange failed"},
	}
	ErrGitHubUserFetch = &service.ErrCode{
		Value:   "github_user_fetch_failed",
		HTTP:    502,
		Message: &i18n.Message{ID: "error.github_user_fetch_failed", Other: "Failed to fetch GitHub account information"},
	}
)

// Password policy codes used by change-password validation. ParamProvider
// pulls the runtime-configured limits so i18n messages render the actual
// thresholds without validator Param() plumbing.
var (
	ErrPasswordTooShort = &service.ErrCode{
		Value:       "password_too_short",
		HTTP:        422,
		Message:     &i18n.Message{ID: "error.password_too_short", Other: "Password must be at least {{.Min}} characters"},
		Placeholder: "Min",
		ParamProvider: func() string {
			return strconv.Itoa(8)
		},
	}
	ErrPasswordTooLong = &service.ErrCode{
		Value:       "password_too_long",
		HTTP:        422,
		Message:     &i18n.Message{ID: "error.password_too_long", Other: "Password must not exceed {{.Max}} characters"},
		Placeholder: "Max",
		ParamProvider: func() string {
			return strconv.Itoa(72)
		},
	}
)
