package utils

import (
	"testing"
)

// FuzzParseMedicalCertificate fuzz tests the ParseMedicalCertificate function
// with arbitrary OCR text inputs to ensure it never panics and maintains basic
// invariants.
func FuzzParseMedicalCertificate(f *testing.F) {
	// Seed corpus from the existing test fixtures.
	f.Add(sampleCertificate)
	f.Add(splitDoc)
	f.Add("")
	f.Add("OCR failed")
	f.Add("no recognizable medical fields are present in this line of text")
	f.Add("UNITATEA MEDICALA: Clinica\nFISA DE APTITUDINE NR. 42\nNUME: Popescu\nCNP: 1234567890123")
	f.Add("APT: [X]  INAPT: []")
	f.Add("garbage input !@#$%^&*()")

	f.Fuzz(func(t *testing.T, ocrText string) {
		words := wordsFromText(ocrText, 92)

		data := ParseMedicalCertificate(ocrText, words)

		// nil result is valid for empty/failed OCR.
		if data == nil {
			return
		}

		// Non-nil result should always have document type set.
		if data.DocumentType != "fisa_aptitudine" {
			t.Errorf("DocumentType = %q, want \"fisa_aptitudine\"", data.DocumentType)
		}

		// Schema version should always be populated.
		if data.SchemaVersion != "1.0.0" {
			t.Errorf("SchemaVersion = %q, want \"1.0.0\"", data.SchemaVersion)
		}

		// FieldConfidences must never be nil.
		if data.FieldConfidences == nil {
			t.Error("FieldConfidences is nil")
		}

		// Empty field confidences implies overall 0.
		if len(data.FieldConfidences) == 0 && data.OverallConfidence != 0 {
			t.Errorf("OverallConfidence = %v, want 0 when no fields extracted", data.OverallConfidence)
		}

		// All confidence values must be in [0, 100].
		for key, conf := range data.FieldConfidences {
			if conf < 0 || conf > 100 {
				t.Errorf("FieldConfidences[%q] = %v, want in [0, 100]", key, conf)
			}
		}
	})
}
