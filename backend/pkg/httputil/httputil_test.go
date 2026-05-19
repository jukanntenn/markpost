package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsSuccessStatus(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, true},
		{201, true},
		{299, true},
		{199, false},
		{300, false},
		{404, false},
		{500, false},
	}

	for _, tc := range tests {
		t.Run(http.StatusText(tc.code), func(t *testing.T) {
			got := isSuccessStatus(tc.code)
			if got != tc.want {
				t.Errorf("isSuccessStatus(%d) = %v, want %v", tc.code, got, tc.want)
			}
		})
	}
}

func TestCheckHTTPResponse(t *testing.T) {
	t.Run("2xx returns nil", func(t *testing.T) {
		resp := &http.Response{StatusCode: 200}
		if err := CheckHTTPResponse(resp, "/test"); err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	t.Run("non-2xx contains status code and path", func(t *testing.T) {
		resp := &http.Response{StatusCode: 404}
		err := CheckHTTPResponse(resp, "/test")
		if err == nil {
			t.Fatal("expected error for 404")
		}
		msg := err.Error()
		if !strings.Contains(msg, "status 404") {
			t.Errorf("error %q should contain status 404", msg)
		}
		if !strings.Contains(msg, "/test") {
			t.Errorf("error %q should contain /test", msg)
		}
	})
}

func TestFetchAndDecodeJSON(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"key": "value"})
		}))
		defer srv.Close()

		var m map[string]string
		if err := FetchAndDecodeJSON(srv.Client(), srv.URL, &m); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m["key"] != "value" {
			t.Errorf("m[key] = %q, want %q", m["key"], "value")
		}
	})

	t.Run("non-2xx", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		}))
		defer srv.Close()

		var m map[string]string
		err := FetchAndDecodeJSON(srv.Client(), srv.URL, &m)
		if err == nil {
			t.Fatal("expected error for 500")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("error %q should contain status 500", err.Error())
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("not-json"))
		}))
		defer srv.Close()

		var m map[string]string
		if err := FetchAndDecodeJSON(srv.Client(), srv.URL, &m); err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}
