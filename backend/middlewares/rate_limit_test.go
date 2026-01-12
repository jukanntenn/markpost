package middlewares

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/didip/tollbooth/v8"
	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"

	"markpost/utils"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newTestI18nRouter(t *testing.T) *gin.Engine {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get caller path")
	}
	localesPath := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "locales"))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ginI18n.Localize(ginI18n.WithBundle(&ginI18n.BundleCfg{
		RootPath:         localesPath,
		AcceptLanguage:   []language.Tag{language.English, language.Chinese},
		DefaultLanguage:  language.English,
		UnmarshalFunc:    toml.Unmarshal,
		FormatBundleFile: "toml",
		Loader:           utils.ActiveLocaleLoader{},
	})))
	return r
}

func TestRateLimitByIP_JSON_EN(t *testing.T) {
	r := newTestI18nRouter(t)

	lmt := tollbooth.NewLimiter(1, nil)
	lmt.SetBurst(1)
	lmt.SetMessage("error.rate_limited")
	lmt.SetStatusCode(http.StatusTooManyRequests)

	r.Use(RateLimitByIP(lmt))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	req1 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req1.Header.Set("Accept-Language", "en")
	req1.RemoteAddr = "1.2.3.4:1234"
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec1.Code, rec1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req2.Header.Set("Accept-Language", "en")
	req2.RemoteAddr = "1.2.3.4:1234"
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusTooManyRequests, rec2.Code, rec2.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != "rate_limited" {
		t.Fatalf("expected code %q, got %q", "rate_limited", resp.Code)
	}
	if resp.Message != "You have reached maximum request limit." {
		t.Fatalf("expected message %q, got %q", "You have reached maximum request limit.", resp.Message)
	}
}

func TestRateLimitByIP_JSON_ZH(t *testing.T) {
	r := newTestI18nRouter(t)

	lmt := tollbooth.NewLimiter(1, nil)
	lmt.SetBurst(1)
	lmt.SetMessage("error.rate_limited")
	lmt.SetStatusCode(http.StatusTooManyRequests)

	r.Use(RateLimitByIP(lmt))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	req1 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req1.Header.Set("Accept-Language", "zh")
	req1.RemoteAddr = "1.2.3.4:1234"
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, rec1.Code, rec1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req2.Header.Set("Accept-Language", "zh")
	req2.RemoteAddr = "1.2.3.4:1234"
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusTooManyRequests, rec2.Code, rec2.Body.String())
	}

	var resp errorResponse
	if err := json.NewDecoder(rec2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if resp.Code != "rate_limited" {
		t.Fatalf("expected code %q, got %q", "rate_limited", resp.Code)
	}
	if resp.Message != "请求过于频繁，请稍后再试" {
		t.Fatalf("expected message %q, got %q", "请求过于频繁，请稍后再试", resp.Message)
	}
}

