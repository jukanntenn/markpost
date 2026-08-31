package admin

import (
	"markpost/internal/service"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Admin-domain error codes, per error-handling.md's "域专属码分文件" principle.

var (
	// ErrSelfForbidden guards governance actions against the actor's own
	// account: cannot change own role / disable self / delete self.
	// K.7 D3-3.
	ErrSelfForbidden = &service.ErrCode{
		Value:   "self_forbidden",
		HTTP:    403,
		Message: &i18n.Message{ID: "error.self_forbidden", Other: "Cannot perform this action on yourself"},
	}

	// ErrLastAdmin guards against demoting/removing the last remaining admin.
	// K.7 D3-3.
	ErrLastAdmin = &service.ErrCode{
		Value:   "last_admin",
		HTTP:    403,
		Message: &i18n.Message{ID: "error.last_admin", Other: "Cannot demote the last administrator"},
	}

	// ErrUnknownSetting rejects writes to runtime settings keys outside the
	// seeded set (MRFC 2026-08-23-github-login-vip-grant-strategy: v1 admits
	// only what a migration seeded).
	ErrUnknownSetting = &service.ErrCode{
		Value:   "unknown_setting",
		HTTP:    400,
		Message: &i18n.Message{ID: "error.unknown_setting", Other: "Unknown setting key"},
	}

	// ErrSettingValueShape rejects a settings write whose payload does not
	// match the key's value shape ({"enabled"} vs {"days"}).
	ErrSettingValueShape = &service.ErrCode{
		Value:   "invalid_setting_value",
		HTTP:    400,
		Message: &i18n.Message{ID: "error.invalid_setting_value", Other: "Value does not match the setting key"},
	}

	// ErrRetentionDays rejects a retention policy outside 0–3650 (0 = keep
	// forever; MRFC 2026-08-31-per-user-history-retention-policy).
	ErrRetentionDays = &service.ErrCode{
		Value:   "invalid_retention_days",
		HTTP:    400,
		Message: &i18n.Message{ID: "error.invalid_retention_days", Other: "Retention days must be 0 (forever) through 3650"},
	}
)
