package delivery

import "testing"

func TestChannelKind_IsValid(t *testing.T) {
	tests := []struct {
		kind ChannelKind
		want bool
	}{
		{ChannelKindFeishu, true},
		{"slack", false},
		{"", false},
		{"FEISHU", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			if got := tt.kind.IsValid(); got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.kind, got, tt.want)
			}
		})
	}
}

func TestChannel_TableName(t *testing.T) {
	ch := Channel{}
	if ch.TableName() != "delivery_channels" {
		t.Errorf("TableName() = %q, want %q", ch.TableName(), "delivery_channels")
	}
}

func TestChannelConfiguration_Value(t *testing.T) {
	t.Run("non-nil configuration", func(t *testing.T) {
		c := ChannelConfiguration{
			"webhook_url":   "https://example.com/hook",
			"card_link_url": "https://example.com/{{.QID}}",
		}
		v, err := c.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s, ok := v.(string)
		if !ok {
			t.Fatal("expected string value")
		}
		if s == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("nil configuration", func(t *testing.T) {
		var c ChannelConfiguration
		v, err := c.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "{}" {
			t.Errorf("expected '{}', got %v", v)
		}
	})
}

func TestChannelConfiguration_Scan(t *testing.T) {
	t.Run("valid JSON string", func(t *testing.T) {
		var c ChannelConfiguration
		err := c.Scan(`{"webhook_url":"https://example.com","card_link_url":""}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		feishu := c.Feishu()
		if feishu.WebhookURL != "https://example.com" {
			t.Errorf("webhook_url = %q, want %q", feishu.WebhookURL, "https://example.com")
		}
	})

	t.Run("nil value", func(t *testing.T) {
		var c ChannelConfiguration
		err := c.Scan(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Error("expected non-nil map")
		}
	})

	t.Run("empty bytes", func(t *testing.T) {
		var c ChannelConfiguration
		err := c.Scan([]byte{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Error("expected non-nil map")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		var c ChannelConfiguration
		err := c.Scan(42)
		if err == nil {
			t.Fatal("expected error for unsupported type")
		}
	})
}

func TestChannelConfiguration_Feishu(t *testing.T) {
	c := ChannelConfiguration{
		"webhook_url":   "https://hook.example.com",
		"card_link_url": "https://custom.example.com/{{.QID}}",
	}
	feishu := c.Feishu()
	if feishu.WebhookURL != "https://hook.example.com" {
		t.Errorf("webhook_url = %q, want %q", feishu.WebhookURL, "https://hook.example.com")
	}
	if feishu.CardLinkURL != "https://custom.example.com/{{.QID}}" {
		t.Errorf("card_link_url = %q, want %q", feishu.CardLinkURL, "https://custom.example.com/{{.QID}}")
	}
}

func TestChannelConfiguration_stringField(t *testing.T) {
	c := ChannelConfiguration{"key": "value"}
	if got := c.stringField("key"); got != "value" {
		t.Errorf("got %q, want %q", got, "value")
	}
	if got := c.stringField("missing"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	c["non_string"] = 42
	if got := c.stringField("non_string"); got != "" {
		t.Errorf("got %q, want empty for non-string", got)
	}
}
