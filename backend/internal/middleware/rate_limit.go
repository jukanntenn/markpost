package middleware

import (
	"strconv"

	"markpost/internal/service"

	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
	"github.com/gin-gonic/gin"
)

// RateLimitByIP returns a rate limiting middleware keyed on the gin-resolved
// client IP (which applies the trusted-proxy logic in SetTrustedProxies). The
// limiter is intended for the public read path, where IP is the only available
// identifier. An unresolvable IP (empty ClientIP) yields 429 rather than
// collapsing anonymous clients into a shared bucket.
func RateLimitByIP(lmt *limiter.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			abortWithError(c, service.NewServiceError(service.ErrRateLimited, "rate limit exceeded"))
			return
		}
		if httpErr := tollbooth.LimitByKeys(lmt, []string{ip}); httpErr != nil {
			abortWithError(c, service.NewServiceError(service.ErrRateLimited, "rate limit exceeded"))
			return
		}
		c.Next()
	}
}

// RateLimitByUserID returns a rate limiting middleware keyed on the user_id
// previously placed in the gin context by an auth middleware (PostKey or
// AuthWithBlacklist). Keying on user_id (rather than the raw post_key or IP)
// means rotating a credential or coming from a new IP cannot evade the limit,
// and unifies the dimension across the write paths. An absent user_id yields
// 429 — the limiter must run after the auth middleware that sets it.
//
// limiters are applied in order; passing more than one chains them (all must
// pass), which is how L2 expresses its 10/min plus 1000/day caps.
func RateLimitByUserID(limiters ...*limiter.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := userIDFromContext(c)
		if !ok {
			abortWithError(c, service.NewServiceError(service.ErrRateLimited, "rate limit exceeded"))
			return
		}
		for _, lmt := range limiters {
			if httpErr := tollbooth.LimitByKeys(lmt, []string{userID}); httpErr != nil {
				abortWithError(c, service.NewServiceError(service.ErrRateLimited, "rate limit exceeded"))
				return
			}
		}
		c.Next()
	}
}

// userIDFromContext extracts the authenticated actor's stable key as a string.
// The auth middlewares set "user_id" (int); the rate limiter keys on it so the
// dimension is identical across PostKey and JWT paths.
func userIDFromContext(c *gin.Context) (string, bool) {
	raw, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	switch v := raw.(type) {
	case int:
		if v <= 0 {
			return "", false
		}
		return strconv.Itoa(v), true
	case int64:
		if v <= 0 {
			return "", false
		}
		return strconv.FormatInt(v, 10), true
	default:
		return "", false
	}
}

// NewLimiter constructs a tollbooth limiter from a per-second rate and burst.
// It is a thin convenience around tollbooth.New so route wiring stays readable;
// the middleware deliberately bypasses tollbooth's own IP lookup and response
// writer (IP is resolved by gin via trusted proxies; the 429 body is a custom
// i18n JSON via apierr), so no message/content-type is configured here.
func NewLimiter(perSecond float64, burst int) *limiter.Limiter {
	lmt := tollbooth.NewLimiter(perSecond, nil)
	lmt.SetBurst(burst)
	return lmt
}
