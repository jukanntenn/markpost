package main

import (
	"reflect"
	"strings"
	"unicode/utf8"

	"markpost/internal/config"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// RegisterValidators registers the custom validator rules and the tag-name
// resolver so validator FieldError.Field() returns the JSON/form field name
// directly (no reflection needed at the call site). See error-handling.md
// "RegisterTagNameFunc".
func RegisterValidators() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		if jsonTag := f.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			if name := strings.Split(jsonTag, ",")[0]; name != "" {
				return name
			}
		}
		if formTag := f.Tag.Get("form"); formTag != "" {
			if name := strings.Split(formTag, ",")[0]; name != "" {
				return name
			}
		}
		return f.Name
	})
	_ = v.RegisterValidation("titlesize", validateTitleLength)
	_ = v.RegisterValidation("bodysize", validateBodySize)
}

func validateTitleLength(fl validator.FieldLevel) bool {
	cfg := config.Get()
	if cfg.Post.TitleMaxLength <= 0 {
		return true
	}
	return utf8.RuneCountInString(fl.Field().String()) <= cfg.Post.TitleMaxLength
}

func validateBodySize(fl validator.FieldLevel) bool {
	cfg := config.Get()
	if cfg.Post.BodyMaxBytes <= 0 {
		return true
	}
	return len([]byte(fl.Field().String())) <= cfg.Post.BodyMaxBytes
}
