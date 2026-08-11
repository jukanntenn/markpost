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
)
