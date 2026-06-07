// Package testutil provides test helpers for gin engine setup.
package testutil

import (
	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"
)

// ValidatorRegistration holds a custom validator tag and its validation function.
type ValidatorRegistration struct {
	Tag string
	Fn  validator.Func
}

// TestEngineConfig holds configuration options for creating a test gin engine.
type TestEngineConfig struct {
	LocalesPath string
	Validators  []ValidatorRegistration
}

// NewTestEngine creates a gin engine configured for testing with optional i18n and custom validators.
func NewTestEngine(cfg TestEngineConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	if cfg.LocalesPath != "" {
		r.Use(ginI18n.Localize(ginI18n.WithBundle(&ginI18n.BundleCfg{
			RootPath:         cfg.LocalesPath,
			AcceptLanguage:   []language.Tag{language.English},
			DefaultLanguage:  language.English,
			UnmarshalFunc:    toml.Unmarshal,
			FormatBundleFile: "toml",
		})))
	}

	if len(cfg.Validators) > 0 {
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			for _, vr := range cfg.Validators {
				_ = v.RegisterValidation(vr.Tag, vr.Fn)
			}
		}
	}

	return r
}
