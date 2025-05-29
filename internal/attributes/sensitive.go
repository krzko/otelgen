// Package attributes provides utilities for handling sensitive attributes and their injection into OpenTelemetry spans.
package attributes

import (
	"io"
	"math/rand/v2"
	"os"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v3"
)

// SensitiveAttribute represents a single sensitive attribute (key-value pair).
type SensitiveAttribute struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// SensitiveConfig is the YAML config structure for sensitive data.
type SensitiveConfig struct {
	SensitiveData []SensitiveAttribute `yaml:"sensitive_data"`
}

// SensitiveDataTable is a list of sensitive attributes that can be injected into spans.
// In the future, this can be loaded from config or a database.
var SensitiveDataTable = []SensitiveAttribute{
	// Personally Identifiable Information (PII) - Core
	{Key: "user.ssn", Value: "123-456-789"},
	{Key: "user.email", Value: "fakeuser@example.com"},
	{Key: "user.phone", Value: "+1-555-123-4567"},
	{Key: "user.address", Value: "123 Fake St, Faketown, USA"},
	{Key: "user.dob", Value: "1990-01-01"},
	{Key: "user.name", Value: "John Doe"}, // Added for full names
	{Key: "passport.number", Value: "X12345678"},
	{Key: "driver_license.number", Value: "DL-123456789"},                          // Added driver's license
	{Key: "gen.ai.key", Value: "sk-proj-GxIEuyULqUrqIoBSjlhyym8zIaudjdK7i4OJpZz2"}, // OpenaI fake key

	// Financial PII (often overlaps with PHI if tied to healthcare payments)
	{Key: "credit.card", Value: "4111-1111-1111-1111"},
	{Key: "bank.account", Value: "9876543210"},
	{Key: "health_plan.beneficiary_number", Value: "HPBN-XYZ-789"}, // Specific to healthcare

	// Protected Health Information (PHI) - Specific Identifiers
	{Key: "medical_record.number", Value: "MRN-789012"},    // Crucial for healthcare
	{Key: "health.diagnosis_code", Value: "J45.909"},       // ICD-10 codes, highly sensitive
	{Key: "health.procedure_code", Value: "87220"},         // CPT codes, highly sensitive
	{Key: "health.medication", Value: "Amoxicillin 500mg"}, // Medication names
	{Key: "device.serial_number", Value: "DEV-SN-12345"},   // For medical devices

	// OpenTelemetry Semantic Conventions - High Risk for PHI/PII
	{Key: "db.statement", Value: "SELECT * FROM patients WHERE id = 'MRN-789012'"},         // Database query
	{Key: "url.full", Value: "https://api.example.com/patients/john.doe@example.com/data"}, // Full URL with PII/PHI
	{Key: "http.request.header.authorization", Value: "Bearer fake-token-abcdef"},          // Auth tokens can contain PII/PHI
	{Key: "http.request.header.x_patient_id", Value: "PHI-PATIENT-ID-001"},                 // Custom headers with PII/PHI
	{Key: "net.peer.ip", Value: "203.0.113.45"},                                            // Client IP
	{Key: "ip.address", Value: "192.168.1.100"},                                            // Generic IP address
	{Key: "web.url", Value: "https://patientportal.example.com/profile"},                   // Web URL (similar to url.full)
	{Key: "biometric.fingerprint", Value: "fingerprint-hash-abc"},                          // Biometric data
	{Key: "image.full_face", Value: "image-data-base64-xyz"},                               // Full face images
}

// LoadSensitiveConfig loads sensitive data from a YAML config file and overrides SensitiveDataTable.
// Returns an error if loading or parsing fails.
func LoadSensitiveConfig(path string) error {
	f, err := os.Open(path) // #nosec G304 -- path is controlled by config, not user input
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	var cfg SensitiveConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}
	if len(cfg.SensitiveData) > 0 {
		SensitiveDataTable = cfg.SensitiveData
	}
	return nil
}

// GetSensitiveAttributes returns the sensitive data as OpenTelemetry attributes.
func GetSensitiveAttributes() []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, len(SensitiveDataTable))
	for i, s := range SensitiveDataTable {
		attrs[i] = attribute.String(s.Key, s.Value)
	}
	return attrs
}

// HasAttribute checks if a key is present in the attributes slice.
func HasAttribute(attrs []string, key string) bool {
	for _, a := range attrs {
		if a == key {
			return true
		}
	}
	return false
}

// RandomAttributes returns a random subset of n attributes from the input slice,
// and a slice of the injected keys.
func RandomAttributes(attrs []attribute.KeyValue, n int) ([]attribute.KeyValue, []string) {
	if n >= len(attrs) {
		keys := make([]string, len(attrs))
		for i, a := range attrs {
			keys[i] = string(a.Key)
		}
		return attrs, keys
	}
	indices := rand.Perm(len(attrs))[:n]
	result := make([]attribute.KeyValue, n)
	keys := make([]string, n)
	for i, idx := range indices {
		result[i] = attrs[idx]
		keys[i] = string(attrs[idx].Key)
	}
	return result, keys
}

// InjectRandomSensitiveAttributes injects a random number of sensitive attributes into the span if 'sensitive' is present in attributes.
// It returns the injected keys and whether any were injected.
func InjectRandomSensitiveAttributes(span trace.Span, attributes []string) (injectedKeys []string, injected bool) {
	sensitiveAttrs := GetSensitiveAttributes()
	if HasAttribute(attributes, "sensitive") && rand.Float64() < 0.1 {
		n := rand.IntN(len(sensitiveAttrs)) + 1 // at least 1
		attrs, keys := RandomAttributes(sensitiveAttrs, n)
		span.SetAttributes(attrs...)
		if len(keys) > 0 {
			span.SetAttributes(attribute.Bool("mock.sensitive.present", true))
			span.SetAttributes(attribute.String("mock.sensitive.attributes", strings.Join(keys, ",")))
			return keys, true
		}
	}
	return nil, false
}

// In the future, add functions to load SensitiveDataTable from config or database.
