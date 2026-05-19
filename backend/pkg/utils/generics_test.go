package utils

import (
	"reflect"
	"slices"
	"strconv"
	"testing"
)

func TestMapSlice_NonEmpty(t *testing.T) {
	got := MapSlice([]int{1, 2, 3}, func(n int) string {
		return strconv.Itoa(n)
	})
	want := []string{"1", "2", "3"}
	if !slices.Equal(got, want) {
		t.Errorf("mapSlice = %v, want %v", got, want)
	}
}

func TestMapSlice_Empty(t *testing.T) {
	got := MapSlice([]int{}, func(n int) string {
		return strconv.Itoa(n)
	})
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestMapSlice_Nil(t *testing.T) {
	got := MapSlice[int, string](nil, func(n int) string {
		return strconv.Itoa(n)
	})
	if len(got) != 0 {
		t.Errorf("expected empty slice for nil input, got %v", got)
	}
}

func TestMapSlice_TypeTransformation(t *testing.T) {
	type src struct{ V int }
	type dst struct{ Out int }
	got := MapSlice([]src{{1}, {2}}, func(s src) dst {
		return dst{Out: s.V * 10}
	})
	want := []dst{{10}, {20}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("mapSlice = %v, want %v", got, want)
	}
}

func TestDeref(t *testing.T) {
	t.Run("non-nil int", func(t *testing.T) {
		v := 5
		got := Deref(&v)
		if got != 5 {
			t.Errorf("deref(&5) = %d, want 5", got)
		}
	})

	t.Run("nil int", func(t *testing.T) {
		got := Deref[int](nil)
		if got != 0 {
			t.Errorf("deref[int](nil) = %d, want 0", got)
		}
	})

	t.Run("non-nil string", func(t *testing.T) {
		s := "hello"
		got := Deref(&s)
		if got != "hello" {
			t.Errorf("deref(&\"hello\") = %q, want \"hello\"", got)
		}
	})

	t.Run("nil string", func(t *testing.T) {
		got := Deref[string](nil)
		if got != "" {
			t.Errorf("deref[string](nil) = %q, want empty string", got)
		}
	})

	t.Run("non-nil struct", func(t *testing.T) {
		type item struct{ X int }
		v := item{X: 42}
		got := Deref(&v)
		if got != (item{X: 42}) {
			t.Errorf("deref(&item{X:42}) = %+v, want {X:42}", got)
		}
	})

	t.Run("nil struct", func(t *testing.T) {
		type item struct{ X int }
		got := Deref[item](nil)
		if got != (item{}) {
			t.Errorf("deref[item](nil) = %+v, want zero value", got)
		}
	})
}
