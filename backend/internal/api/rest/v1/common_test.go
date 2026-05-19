package v1

import (
	"errors"
	"reflect"
	"testing"

	"markpost/internal/service"

	"github.com/go-playground/validator/v10"
)

type testReq struct {
	Title string `json:"title" validate:"required"`
	Body  string `json:"body" validate:"min=1"`
	Count int    `json:"count" validate:"min=1"`
}

type testReqForm struct {
	Query string `form:"q" validate:"required"`
}

type testReqNoTags struct {
	Name string `validate:"required"`
}

func validateStruct(t *testing.T, v *validator.Validate, s interface{}) validator.ValidationErrors {
	t.Helper()
	err := v.Struct(s)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected validator.ValidationErrors, got %T: %v", err, err)
	}
	return ve
}

func TestParseBindingErrors_NonValidatorError(t *testing.T) {
	causes := ParseBindingErrors(errors.New("some other error"), testReq{})
	if len(causes) != 1 {
		t.Fatalf("expected 1 cause, got %d", len(causes))
	}
	if causes[0].Code != service.ErrFieldViolation {
		t.Errorf("expected ErrFieldViolation, got %s", causes[0].Code)
	}
	if causes[0].Description != "" {
		t.Errorf("expected empty description, got %q", causes[0].Description)
	}
}

func TestParseBindingErrors_RequiredTag(t *testing.T) {
	v := validator.New()
	ve := validateStruct(t, v, testReq{}) // Title is required, empty

	causes := ParseBindingErrors(ve, testReq{})
	found := false
	for _, c := range causes {
		if c.Code == service.ErrRequired {
			found = true
			if c.Description != "title" {
				t.Errorf("expected description 'title', got %q", c.Description)
			}
		}
	}
	if !found {
		t.Error("expected ErrRequired cause")
	}
}

func TestParseBindingErrors_MinTag(t *testing.T) {
	v := validator.New()
	ve := validateStruct(t, v, testReq{Title: "ok", Body: "hello", Count: 0}) // Count min=1 fails

	causes := ParseBindingErrors(ve, testReq{Title: "ok", Body: "hello", Count: 0})
	found := false
	for _, c := range causes {
		if c.Code == service.ErrMinLength {
			found = true
			if c.Description != "count" {
				t.Errorf("expected description 'count', got %q", c.Description)
			}
		}
	}
	if !found {
		t.Error("expected ErrMinLength cause")
	}
}

func TestParseBindingErrors_UnknownTag(t *testing.T) {
	type unknownTagReq struct {
		Email string `json:"email" validate:"email"`
	}
	v := validator.New()
	ve := validateStruct(t, v, unknownTagReq{Email: "not-an-email"})

	causes := ParseBindingErrors(ve, unknownTagReq{Email: "not-an-email"})
	if len(causes) == 0 {
		t.Fatal("expected at least one cause")
	}
	for _, c := range causes {
		if c.Code != service.ErrFieldViolation {
			t.Errorf("expected ErrFieldViolation for unknown tag, got %s", c.Code)
		}
	}
}

func TestParseBindingErrors_PointerToStruct(t *testing.T) {
	v := validator.New()
	ve := validateStruct(t, v, &testReq{})

	causes := ParseBindingErrors(ve, &testReq{})
	found := false
	for _, c := range causes {
		if c.Code == service.ErrRequired && c.Description == "title" {
			found = true
		}
	}
	if !found {
		t.Error("expected ErrRequired with description 'title' for pointer-to-struct request")
	}
}

func TestParseBindingErrors_FormTagFallback(t *testing.T) {
	v := validator.New()
	ve := validateStruct(t, v, testReqForm{})

	causes := ParseBindingErrors(ve, testReqForm{})
	if len(causes) == 0 {
		t.Fatal("expected at least one cause")
	}
	if causes[0].Description != "q" {
		t.Errorf("expected description 'q' from form tag, got %q", causes[0].Description)
	}
}

func TestParseBindingErrors_NoJsonOrFormTag(t *testing.T) {
	v := validator.New()
	ve := validateStruct(t, v, testReqNoTags{})

	causes := ParseBindingErrors(ve, testReqNoTags{})
	if len(causes) == 0 {
		t.Fatal("expected at least one cause")
	}
	if causes[0].Description != "Name" {
		t.Errorf("expected description 'Name' (field name fallback), got %q", causes[0].Description)
	}
}

type resolveField struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
	Query string `form:"q"`
	Bare  string
	Ign   string `json:"-"`
}

func TestResolveFieldName_JsonTag(t *testing.T) {
	got := resolveFieldName(reflect.TypeOf(resolveField{}), "Email")
	if got != "email" {
		t.Errorf("got %q, want %q", got, "email")
	}
}

func TestResolveFieldName_JsonTagOmitEmpty(t *testing.T) {
	got := resolveFieldName(reflect.TypeOf(resolveField{}), "Name")
	if got != "name" {
		t.Errorf("got %q, want %q", got, "name")
	}
}

func TestResolveFieldName_FormTagFallback(t *testing.T) {
	got := resolveFieldName(reflect.TypeOf(resolveField{}), "Query")
	if got != "q" {
		t.Errorf("got %q, want %q", got, "q")
	}
}

func TestResolveFieldName_NoTag(t *testing.T) {
	got := resolveFieldName(reflect.TypeOf(resolveField{}), "Bare")
	if got != "Bare" {
		t.Errorf("got %q, want %q", got, "Bare")
	}
}

func TestResolveFieldName_JsonIgnoreTag(t *testing.T) {
	got := resolveFieldName(reflect.TypeOf(resolveField{}), "Ign")
	if got != "Ign" {
		t.Errorf("got %q, want %q", got, "Ign")
	}
}

func TestResolveFieldName_PointerToStruct(t *testing.T) {
	got := resolveFieldName(reflect.TypeOf(&resolveField{}), "Email")
	if got != "email" {
		t.Errorf("got %q, want %q", got, "email")
	}
}

func TestResolveFieldName_NonStructType(t *testing.T) {
	got := resolveFieldName(reflect.TypeOf(0), "Anything")
	if got != "Anything" {
		t.Errorf("got %q, want %q", got, "Anything")
	}
}
