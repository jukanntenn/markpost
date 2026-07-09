package web

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cespare/xxhash/v2"
)

// TestCSSHash_MatchesMinifiedAsset asserts the content-addressing contract:
// CSSHash equals xxhash64 of the embedded, minified CSS bytes.
func TestCSSHash_MatchesMinifiedAsset(t *testing.T) {
	bytes := CSSBytes()
	if len(bytes) == 0 {
		t.Fatal("embedded CSS asset is empty")
	}
	expected := fmt.Sprintf("%016x", xxhash.Sum64(bytes))
	if CSSHash != expected {
		t.Errorf("CSSHash = %q, want xxhash of asset %q", CSSHash, expected)
	}
	if len(CSSHash) != 16 {
		t.Errorf("expected 16-hex-char hash, got %q (%d chars)", CSSHash, len(CSSHash))
	}
	if assetURL := "post." + CSSHash + ".css"; !strings.Contains(assetURL, CSSHash) {
		t.Errorf("asset URL %q must embed CSSHash", assetURL)
	}
}
