// Package schema embeds the JSON Schemas that describe the OCR
// extraction subsystem's output contracts.
package schema

import _ "embed"

// MedicalDataV1 is the JSON Schema (draft 2020-12) for v1 of the
// medical_data extraction output. See ../docs/ocr-extraction.md for the
// design rationale and versioning policy.
//
//go:embed medical-data.v1.json
var MedicalDataV1 []byte
