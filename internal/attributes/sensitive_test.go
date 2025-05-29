package attributes

import (
	"os"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestLoadSensitiveConfig_Valid(t *testing.T) {
	content := []byte(`sensitive_data:
  - key: test.key
    value: test-value
  - key: another.key
    value: another-value
`)
	tmpfile, err := os.CreateTemp(t.TempDir(), "sensitive-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() {
		if removeErr := os.Remove(tmpfile.Name()); removeErr != nil {
			t.Fatalf("failed to remove temp file: %v", removeErr)
		}
	}()
	if _, writeErr := tmpfile.Write(content); writeErr != nil {
		t.Fatalf("failed to write to temp file: %v", writeErr)
	}
	if closeErr := tmpfile.Close(); closeErr != nil {
		t.Fatalf("failed to close temp file: %v", closeErr)
	}

	oldTable := SensitiveDataTable
	defer func() { SensitiveDataTable = oldTable }()

	err = LoadSensitiveConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(SensitiveDataTable) != 2 {
		t.Errorf("expected 2 sensitive attributes, got %d", len(SensitiveDataTable))
	}
	if SensitiveDataTable[0].Key != "test.key" || SensitiveDataTable[0].Value != "test-value" {
		t.Errorf("unexpected first attribute: %+v", SensitiveDataTable[0])
	}
}

func TestLoadSensitiveConfig_NoSensitiveData(t *testing.T) {
	content := []byte(`not_sensitive_data:
  - key: foo
    value: bar
`)
	tmpfile, err := os.CreateTemp(t.TempDir(), "sensitive-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() {
		if removeErr := os.Remove(tmpfile.Name()); removeErr != nil {
			t.Fatalf("failed to remove temp file: %v", removeErr)
		}
	}()
	if _, writeErr := tmpfile.Write(content); writeErr != nil {
		t.Fatalf("failed to write to temp file: %v", writeErr)
	}
	if closeErr := tmpfile.Close(); closeErr != nil {
		t.Fatalf("failed to close temp file: %v", closeErr)
	}

	oldTable := SensitiveDataTable
	defer func() { SensitiveDataTable = oldTable }()

	err = LoadSensitiveConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Should not override
	if len(SensitiveDataTable) != len(oldTable) {
		t.Errorf("expected SensitiveDataTable to remain unchanged")
	}
}

func TestLoadSensitiveConfig_FileNotFound(t *testing.T) {
	err := LoadSensitiveConfig("/nonexistent/path/to/file.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestGetSensitiveAttributes(t *testing.T) {
	attrs := GetSensitiveAttributes()
	if len(attrs) == 0 {
		t.Error("expected non-empty sensitive attributes")
	}
	for _, a := range attrs {
		if a.Key == "" || a.Value.AsString() == "" {
			t.Errorf("attribute has empty key or value: %+v", a)
		}
	}
}

func TestHasAttribute(t *testing.T) {
	attrs := []string{"foo", "bar", "baz"}
	if !HasAttribute(attrs, "bar") {
		t.Error("expected to find 'bar' in attrs")
	}
	if HasAttribute(attrs, "qux") {
		t.Error("did not expect to find 'qux' in attrs")
	}
}

func TestRandomAttributes(t *testing.T) {
	attrs := []attribute.KeyValue{
		attribute.String("a", "1"),
		attribute.String("b", "2"),
		attribute.String("c", "3"),
	}
	// n >= len(attrs)
	out, keys := RandomAttributes(attrs, 5)
	if len(out) != 3 || len(keys) != 3 {
		t.Errorf("expected all attributes, got %d", len(out))
	}
	// n < len(attrs)
	out, keys = RandomAttributes(attrs, 2)
	if len(out) != 2 || len(keys) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(out))
	}
}

type dummySpan struct{ trace.Span }

func (d *dummySpan) SetAttributes(_ ...attribute.KeyValue) {}

func TestInjectRandomSensitiveAttributes(t *testing.T) {
	span := &dummySpan{}
	attrs := []string{"sensitive"}
	// Run multiple times to probabilistically cover both branches
	found := false
	for i := 0; i < 100; i++ {
		keys, injected := InjectRandomSensitiveAttributes(span, attrs)
		if injected {
			found = true
			if len(keys) == 0 {
				t.Error("expected injected keys when injected is true")
			}
			break
		}
	}
	if !found {
		t.Log("InjectRandomSensitiveAttributes did not inject (random chance)")
	}
	// Should not inject if 'sensitive' is not present
	keys, injected := InjectRandomSensitiveAttributes(span, []string{"foo"})
	if injected || keys != nil {
		t.Error("expected no injection when 'sensitive' is not present")
	}
}
