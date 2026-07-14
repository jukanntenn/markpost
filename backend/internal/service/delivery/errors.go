package delivery

import (
	"markpost/internal/service"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Delivery-domain error codes. Channel CRUD business-validation failures reuse
// the shared service.ErrValidation (HTTP 422) with inline descriptions; this
// file holds the few delivery-specific codes that benefit from a dedicated
// machine-readable value for frontend logic.

var (
	// ErrUnsupportedChannelKind is returned when a channel kind is not one of
	// the supported values. HTTP 422: the request parses but the kind value is
	// semantically invalid (api-design.md §3.1).
	ErrUnsupportedChannelKind = &service.ErrCode{
		Value:   "unsupported_channel_kind",
		HTTP:    422,
		Message: &i18n.Message{ID: "error.unsupported_channel_kind", Other: "Unsupported channel kind"},
	}
)
