package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"

	"mqtt-streaming-server/schema"
)

// compileMedicalDataSchema parses and compiles the embedded schema. Failures
// here mean the schema file itself is broken, so the helper fails the test
// rather than returning an error every caller would have to plumb through.
// The schema bytes come from the schema package — //go:embed disallows ".."
// in patterns, so the embed lives next to the .json file.
func compileMedicalDataSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema.MedicalDataV1))
	if err != nil {
		t.Fatalf("UnmarshalJSON(schema): %v", err)
	}
	const id = "medical-data.v1.json"
	c := jsonschema.NewCompiler()
	if err := c.AddResource(id, doc); err != nil {
		t.Fatalf("AddResource: %v", err)
	}
	sch, err := c.Compile(id)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return sch
}

// validatePayload decodes payload as a JSON value and validates it against sch.
// Test inputs are constructed by marshalling Go values, so a JSON-decode error
// here is a test-setup bug, not a validation outcome — surface it loudly.
func validatePayload(t *testing.T, sch *jsonschema.Schema, payload []byte) error {
	t.Helper()
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("UnmarshalJSON(instance): %v", err)
	}
	return sch.Validate(inst)
}

// hasErrorKind reports whether any node in the ValidationError tree carries an
// ErrorKind that the predicate accepts. The library nests sub-errors under the
// top-level "doesn't validate" error, so per-keyword kinds (Maximum, Type,
// Enum, Required, ...) live in the Causes subtree — a plain ErrorKind check on
// the top error misses them.
func hasErrorKind(err error, match func(jsonschema.ErrorKind) bool) bool {
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return false
	}
	var walk func(*jsonschema.ValidationError) bool
	walk = func(e *jsonschema.ValidationError) bool {
		if e.ErrorKind != nil && match(e.ErrorKind) {
			return true
		}
		for _, c := range e.Causes {
			if walk(c) {
				return true
			}
		}
		return false
	}
	return walk(ve)
}

// validPayloadFromParser parses sampleCertificate and returns the resulting
// MedicalData as a map so negative-case tests can mutate one key cheaply.
// Going Go → JSON → map is deliberate: it proves the parser's actual output
// shape is the baseline we're perturbing, rather than a hand-rolled fixture
// that could silently drift from MedicalData.
func validPayloadFromParser(t *testing.T) map[string]any {
	t.Helper()
	words := wordsFromText(sampleCertificate, 92)
	data := ParseMedicalCertificate(sampleCertificate, words)
	if data == nil {
		t.Fatal("ParseMedicalCertificate returned nil for sampleCertificate")
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal MedicalData: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal into map: %v", err)
	}
	return m
}

// TestMedicalDataSchema_ValidatesParserOutput is the positive case: the parser
// output for sampleCertificate must satisfy the schema today and on every
// future schema bump.
func TestMedicalDataSchema_ValidatesParserOutput(t *testing.T) {
	sch := compileMedicalDataSchema(t)

	words := wordsFromText(sampleCertificate, 92)
	data := ParseMedicalCertificate(sampleCertificate, words)
	if data == nil {
		t.Fatal("ParseMedicalCertificate returned nil for sampleCertificate")
	}
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("json.Marshal(MedicalData): %v", err)
	}

	if err := validatePayload(t, sch, payload); err != nil {
		t.Fatalf("parser output failed schema validation: %v", err)
	}
}

// TestMedicalDataSchema_NegativeCases covers the four locked-in violation
// types: numeric range, primitive type, enum membership, required-field
// presence. Each subtest starts from the same valid payload and mutates one
// key, so a regression in the schema that lets one of these slip through
// fails just that subtest.
func TestMedicalDataSchema_NegativeCases(t *testing.T) {
	sch := compileMedicalDataSchema(t)

	cases := []struct {
		name       string
		mutate     func(map[string]any)
		expectKind func(jsonschema.ErrorKind) bool
		kindLabel  string
	}{
		{
			name:       "overall_confidence above maximum",
			mutate:     func(m map[string]any) { m["overall_confidence"] = 150 },
			expectKind: func(k jsonschema.ErrorKind) bool { _, ok := k.(*kind.Maximum); return ok },
			kindLabel:  "*kind.Maximum",
		},
		{
			name:       "overall_confidence wrong type",
			mutate:     func(m map[string]any) { m["overall_confidence"] = "high" },
			expectKind: func(k jsonschema.ErrorKind) bool { _, ok := k.(*kind.Type); return ok },
			kindLabel:  "*kind.Type",
		},
		{
			name:       "document_type not in enum",
			mutate:     func(m map[string]any) { m["document_type"] = "prescription" },
			expectKind: func(k jsonschema.ErrorKind) bool { _, ok := k.(*kind.Enum); return ok },
			kindLabel:  "*kind.Enum",
		},
		{
			name:       "missing required needs_review",
			mutate:     func(m map[string]any) { delete(m, "needs_review") },
			expectKind: func(k jsonschema.ErrorKind) bool { _, ok := k.(*kind.Required); return ok },
			kindLabel:  "*kind.Required",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			payload := validPayloadFromParser(t)
			c.mutate(payload)
			raw, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal mutated payload: %v", err)
			}
			err = validatePayload(t, sch, raw)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !hasErrorKind(err, c.expectKind) {
				t.Fatalf("validation failed but no %s cause found; error=%v", c.kindLabel, err)
			}
		})
	}
}
