package delivery

// ErrorCategory classifies a delivery failure so it can drive retry policy,
// metrics labels, and the admin history filter. The empty string means
// "uncategorized" (legacy rows from before the column existed).
type ErrorCategory string

const (
	// CategoryCardRejected: the upstream rejected the card content (e.g. Feishu
	// code 11246 — invalid image keys). Permanent: retrying cannot change the
	// content, so the attempt fails fast instead of burning the backoff budget.
	CategoryCardRejected ErrorCategory = "card_rejected"
	// CategoryUpstreamClientError: HTTP 4xx except 429 — the webhook is
	// misconfigured or revoked. Permanent.
	CategoryUpstreamClientError ErrorCategory = "upstream_client_error"
	// CategoryUpstreamServerError: HTTP 5xx or 429 (rate limited). Transient —
	// retry after backoff.
	CategoryUpstreamServerError ErrorCategory = "upstream_server_error"
	// CategoryUpstreamBusinessError: a non-zero upstream business code that is
	// not specifically recognized. Conservative default: retryable, so a message
	// that could eventually succeed is not dropped prematurely.
	CategoryUpstreamBusinessError ErrorCategory = "upstream_business_error"
	// CategoryNetwork: the request never completed (DNS, timeout, connection
	// refused). Transient.
	CategoryNetwork ErrorCategory = "network"
	// CategoryInternal: our own data or logic error (post/channel not found,
	// payload marshal failure). Retryable by default — it surfaces bugs/data
	// issues while still giving transient hiccups a chance.
	CategoryInternal ErrorCategory = "internal"
)

// DeliveryError wraps a delivery failure with a Category and a Retryable
// decision. Error()/Unwrap() forward to Cause so existing string assertions
// and the last_error column text are unchanged; the category is purely an
// additional, structured dimension.
type DeliveryError struct {
	Category  ErrorCategory
	Retryable bool
	Cause     error
}

func (e *DeliveryError) Error() string {
	if e == nil {
		return "delivery error"
	}
	if e.Cause == nil {
		return "delivery error"
	}
	return e.Cause.Error()
}

func (e *DeliveryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func newDeliveryError(cat ErrorCategory, retryable bool, cause error) *DeliveryError {
	return &DeliveryError{Category: cat, Retryable: retryable, Cause: cause}
}

// classifyFeishuCode maps a Feishu webhook business code to a category and a
// retry decision. Only the specifically-known permanent code (11246, card
// content rejected) is marked non-retryable; every other code defaults to
// retryable so an eventually-succeedable message is not dropped.
func classifyFeishuCode(code int) (ErrorCategory, bool) {
	switch code {
	case 11246: // card content rejected (e.g. invalid image keys)
		return CategoryCardRejected, false
	default:
		return CategoryUpstreamBusinessError, true
	}
}

// classifyHTTPStatus maps a non-2xx HTTP status to a category and retry
// decision, following standard retry semantics: 429 and 5xx are transient
// (server fault / rate limit); other 4xx are permanent client errors.
func classifyHTTPStatus(status int) (ErrorCategory, bool) {
	if status == 429 || status >= 500 {
		return CategoryUpstreamServerError, true
	}
	return CategoryUpstreamClientError, false
}
