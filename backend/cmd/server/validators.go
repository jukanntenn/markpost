package main

import (
	"unicode/utf8"

	"markpost/internal/config"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func RegisterValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("titlesize", validateTitleLength)
		_ = v.RegisterValidation("bodysize", validateBodySize)
	}
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
