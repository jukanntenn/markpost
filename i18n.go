package main

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	bundle       *i18n.Bundle
	localizerMap map[string]*i18n.Localizer
)

func InitI18n() {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	enPath := filepath.Join("locales", "en.json")
	zhPath := filepath.Join("locales", "zh.json")

	_, err := bundle.LoadMessageFile(enPath)
	if err != nil {
		panic("Failed to load en.json: " + err.Error())
	}

	_, err = bundle.LoadMessageFile(zhPath)
	if err != nil {
		panic("Failed to load zh.json: " + err.Error())
	}

	localizerMap = make(map[string]*i18n.Localizer)
	localizerMap["en"] = i18n.NewLocalizer(bundle, "en")
	localizerMap["zh"] = i18n.NewLocalizer(bundle, "zh")
}

func GetMessage(c *gin.Context, messageID string) string {
	lang := GetLanguage(c)
	localizer, exists := localizerMap[lang]
	if !exists {
		localizer = localizerMap["en"]
	}

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: messageID,
		},
	})

	if err != nil {
		return messageID
	}

	return msg
}

func GetMessageWithParams(c *gin.Context, messageID string, params map[string]string) string {
	lang := GetLanguage(c)
	localizer, exists := localizerMap[lang]
	if !exists {
		localizer = localizerMap["en"]
	}

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: messageID,
		},
		TemplateData: params,
	})

	if err != nil {
		return messageID
	}

	return msg
}

func GetLanguage(c *gin.Context) string {
	lang := c.Query("lang")
	if lang == "" {
		lang = c.GetHeader("Accept-Language")
		if lang == "" {
			return "en"
		}
		lang = extractLanguageFromHeader(lang)
	}

	lang = strings.ToLower(lang)
	if lang != "en" && lang != "zh" {
		return "en"
	}

	return lang
}

func extractLanguageFromHeader(header string) string {
	parts := strings.Split(header, ",")
	if len(parts) == 0 {
		return "en"
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) >= 2 {
			lang := part[:2]
			if lang == "en" || lang == "zh" {
				return lang
			}
		}
	}

	return "en"
}

func I18nMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := GetLanguage(c)
		c.Set("language", lang)
		c.Next()
	}
}
