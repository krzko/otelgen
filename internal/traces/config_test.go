package traces

import (
	"testing"
)

func TestHeaderValue_Set_Valid(t *testing.T) {
	h := HeaderValue{}
	err := h.Set("foo=bar")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if h["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", h)
	}
}

func TestHeaderValue_Set_Invalid(t *testing.T) {
	h := HeaderValue{}
	err := h.Set("foobar")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestHeaderValue_String(t *testing.T) {
	h := HeaderValue{"foo": "bar"}
	want := "map[foo:bar]"
	if h.String() != want {
		t.Errorf("expected %q, got %q", want, h.String())
	}
}
