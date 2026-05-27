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
