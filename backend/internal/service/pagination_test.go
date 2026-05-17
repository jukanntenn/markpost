package service

import (
	"testing"
)

func TestValidatePagination(t *testing.T) {
	tests := []struct {
		name      string
		page      int
		limit     int
		wantOff   int
		wantPage  int
		wantLimit int
		wantErr   bool
	}{
		{"valid", 2, 10, 10, 2, 10, false},
		{"limit over 100", 1, 101, 0, 0, 0, true},
		{"limit exactly 100", 1, 100, 0, 1, 100, false},
		{"page zero", 0, 10, 0, 1, 10, false},
		{"page negative", -1, 10, 0, 1, 10, false},
		{"limit zero", 1, 0, 0, 1, 20, false},
		{"limit negative", 1, -5, 0, 1, 20, false},
		{"both zero", 0, 0, 0, 1, 20, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			offset, page, limit, err := ValidatePagination(tc.page, tc.limit)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				se, ok := AsServiceError(err)
				if !ok {
					t.Fatalf("expected *service.Error, got %T", err)
				}
				if se.Code != ErrInvalidRequest {
					t.Fatalf("expected code %s, got %s", ErrInvalidRequest, se.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if offset != tc.wantOff {
				t.Errorf("offset = %d, want %d", offset, tc.wantOff)
			}
			if page != tc.wantPage {
				t.Errorf("page = %d, want %d", page, tc.wantPage)
			}
			if limit != tc.wantLimit {
				t.Errorf("limit = %d, want %d", limit, tc.wantLimit)
			}
		})
	}
}

func TestCalcTotalPages(t *testing.T) {
	tests := []struct {
		name  string
		total int64
		limit int
		want  int
	}{
		{"exact division", 20, 10, 2},
		{"with remainder", 21, 10, 3},
		{"single page", 5, 10, 1},
		{"zero total", 0, 10, 0},
		{"zero limit", 10, 0, 0},
		{"negative limit", 10, -1, 0},
		{"total less than limit", 3, 10, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalcTotalPages(tc.total, tc.limit)
			if got != tc.want {
				t.Errorf("CalcTotalPages(%d, %d) = %d, want %d", tc.total, tc.limit, got, tc.want)
			}
		})
	}
}
