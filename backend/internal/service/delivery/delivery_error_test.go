package delivery

import (
	"errors"
	"fmt"
	"testing"
)

func TestClassifyFeishuCode(t *testing.T) {
	cases := []struct {
		code      int
		wantCat   ErrorCategory
		wantRetry bool
	}{
		{11246, CategoryCardRejected, false},
		{0, CategoryUpstreamBusinessError, true},
		{19021, CategoryUpstreamBusinessError, true},
		{-1, CategoryUpstreamBusinessError, true},
	}
	for _, tt := range cases {
		t.Run(fmt.Sprintf("code_%d", tt.code), func(t *testing.T) {
			cat, retry := classifyFeishuCode(tt.code)
			if cat != tt.wantCat {
				t.Errorf("code=%d category=%s want %s", tt.code, cat, tt.wantCat)
			}
			if retry != tt.wantRetry {
				t.Errorf("code=%d retry=%v want %v", tt.code, retry, tt.wantRetry)
			}
		})
	}
}

func TestClassifyHTTPStatus(t *testing.T) {
	cases := []struct {
		status    int
		wantCat   ErrorCategory
		wantRetry bool
	}{
		{400, CategoryUpstreamClientError, false},
		{403, CategoryUpstreamClientError, false},
		{404, CategoryUpstreamClientError, false},
		{422, CategoryUpstreamClientError, false},
		{429, CategoryUpstreamServerError, true},
		{500, CategoryUpstreamServerError, true},
		{502, CategoryUpstreamServerError, true},
		{503, CategoryUpstreamServerError, true},
	}
	for _, tt := range cases {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			cat, retry := classifyHTTPStatus(tt.status)
			if cat != tt.wantCat {
				t.Errorf("status=%d category=%s want %s", tt.status, cat, tt.wantCat)
			}
			if retry != tt.wantRetry {
				t.Errorf("status=%d retry=%v want %v", tt.status, retry, tt.wantRetry)
			}
		})
	}
}

func TestDeliveryErrorWrapping(t *testing.T) {
	cause := fmt.Errorf("feishu api code=11246 msg=ErrCode: 200570")
	err := newDeliveryError(CategoryCardRejected, false, cause)

	if err.Error() != cause.Error() {
		t.Errorf("Error() = %q, want cause text %q", err.Error(), cause.Error())
	}
	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(cause) = false, want true (Unwrap broken)")
	}

	var derr *DeliveryError
	if !errors.As(err, &derr) {
		t.Fatalf("errors.As(*DeliveryError) = false")
	}
	if derr.Category != CategoryCardRejected {
		t.Errorf("category=%s want card_rejected", derr.Category)
	}
	if derr.Retryable {
		t.Errorf("retryable=true want false")
	}
}
