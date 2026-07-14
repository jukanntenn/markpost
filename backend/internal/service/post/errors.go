package post

import (
	"strconv"

	"markpost/internal/config"
	"markpost/internal/service"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Post-domain error codes. Title/body size codes use ParamProvider to surface
// the runtime-configured limits (config.Post.TitleMaxLength /
// BodyMaxBytes) in i18n messages, since the titlesize/bodysize custom
// validators return an empty Param().

var (
	ErrTitleSize = &service.ErrCode{
		Value:       "title_too_long",
		HTTP:        422,
		Message:     &i18n.Message{ID: "error.validation_titlesize", Other: "{{.Field}} exceeds the maximum of {{.Max}} characters"},
		Placeholder: "Max",
		ParamProvider: func() string {
			return strconv.Itoa(config.Get().Post.TitleMaxLength)
		},
	}
	ErrBodySize = &service.ErrCode{
		Value:       "body_too_large",
		HTTP:        422,
		Message:     &i18n.Message{ID: "error.validation_bodysize", Other: "{{.Field}} exceeds the maximum of {{.Max}} bytes"},
		Placeholder: "Max",
		ParamProvider: func() string {
			return strconv.Itoa(config.Get().Post.BodyMaxBytes)
		},
	}
)
