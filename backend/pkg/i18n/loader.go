// Package i18n provides internationalization utilities.
package i18n

import (
	"io/fs"
	"os"
	"path/filepath"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// ActiveLocaleLoader loads locale messages from the filesystem.
type ActiveLocaleLoader struct{}

// LoadMessage loads a message file for the given locale.
func (ActiveLocaleLoader) LoadMessage(locale string) ([]byte, error) {
	localesPath := "./locales"
	filePath := filepath.Join(localesPath, locale+".toml")

	return os.ReadFile(filePath)
}

// MustLoadMessage loads a message file and panics on error.
func (ActiveLocaleLoader) MustLoadMessage(locale string) []byte {
	data, err := ActiveLocaleLoader{}.LoadMessage(locale)
	if err != nil {
		panic(err)
	}
	return data
}

// LoadLocales loads all locale files into the bundle.
func (ActiveLocaleLoader) LoadLocales(bundle *i18n.Bundle) error {
	localesPath := "./locales"

	return filepath.WalkDir(localesPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".toml" {
			_, err := bundle.LoadMessageFile(path)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

var _ ginI18n.Loader = ActiveLocaleLoader{}
