package delivery

import (
	"strings"
	"testing"

	"markpost/internal/domain/delivery"
	"markpost/internal/service"
)

func TestNormalizeAndValidateKind(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		want    delivery.ChannelKind
		wantErr bool
	}{
		{"lowercase feishu", "feishu", delivery.ChannelKindFeishu, false},
		{"mixed case Feishu", "Feishu", delivery.ChannelKindFeishu, false},
		{"whitespace padded", "  feishu  ", delivery.ChannelKindFeishu, false},
		{"unsupported kind", "slack", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeAndValidateKind(tt.kind)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeAndValidateKind(%q) error = %v, wantErr %v", tt.kind, err, tt.wantErr)
			}
			if err != nil {
				se, ok := service.AsServiceError(err)
				if !ok {
					t.Fatalf("expected service.ServiceError, got %T", err)
				}
				if se.Code != service.ErrValidation {
					t.Errorf("expected code %q, got %q", service.ErrValidation, se.Code)
				}
			}
			if got != tt.want {
				t.Errorf("normalizeAndValidateKind(%q) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestValidateWebhookURL(t *testing.T) {
	t.Run("valid URL", func(t *testing.T) {
		cleaned, err := validateWebhookURL("  https://example.com/webhook  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleaned != "https://example.com/webhook" {
			t.Errorf("cleaned = %q, want %q", cleaned, "https://example.com/webhook")
		}
	})

	t.Run("invalid URL scheme", func(t *testing.T) {
		_, err := validateWebhookURL("ftp://example.com/hook")
		assertValidation(t, err, "invalid webhook URL")
	})

	t.Run("unparseable URL", func(t *testing.T) {
		_, err := validateWebhookURL("://not-a-url")
		assertValidation(t, err, "invalid webhook URL")
	})
}

func assertValidation(t *testing.T, err error, contains string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", contains)
	}
	se, ok := service.AsServiceError(err)
	if !ok {
		t.Fatalf("expected service.ServiceError, got %T", err)
	}
	if se.Code != service.ErrValidation {
		t.Errorf("expected code %q, got %q", service.ErrValidation, se.Code)
	}
	if !strings.Contains(se.Description, contains) {
		t.Errorf("expected description to contain %q, got %q", contains, se.Description)
	}
}
