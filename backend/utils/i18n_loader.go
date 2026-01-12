package utils

import (
	"os"
	"path/filepath"
	"strings"
)

type ActiveLocaleLoader struct{}

func (l ActiveLocaleLoader) LoadMessage(path string) ([]byte, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	lang := strings.TrimSuffix(base, ext)
	normalized := strings.ToLower(lang)

	var candidates []string
	candidates = append(candidates, normalized)
	candidates = append(candidates, mapLocaleTag(normalized))

	if baseLang, _, ok := strings.Cut(normalized, "-"); ok && baseLang != "" {
		candidates = append(candidates, baseLang)
		candidates = append(candidates, mapLocaleTag(baseLang))
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		activePath := filepath.Join(dir, "active."+candidate+ext)
		if buf, err := os.ReadFile(activePath); err == nil {
			return buf, nil
		}
	}

	return os.ReadFile(path)
}

func mapLocaleTag(tag string) string {
	switch strings.ToLower(tag) {
	case "en":
		return "en-us"
	case "zh", "zh-cn", "zh-hans", "zh-hans-cn":
		return "zh-hans"
	default:
		return strings.ToLower(tag)
	}
}
