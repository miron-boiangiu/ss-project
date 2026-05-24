package utils

import "image"

// WordBox is a single OCR-detected word together with its engine confidence
// and position. It links a word's location in the source image to its location
// in the joined OCR text, which is what lets the medical parser attribute a
// confidence score to each extracted field.
type WordBox struct {
	Word       string          // recognized word, verbatim (incl. any leading/internal whitespace)
	Confidence float64         // Tesseract confidence, 0–100
	BBox       image.Rectangle // bounding box in source-image pixel coordinates
	Offset     int             // byte offset of the word's first byte within the joined OCR text
}

// minConfidence returns the lowest confidence among all words whose text span
// [Offset, Offset+len(Word)) overlaps the half-open range [start, end).
//
// It returns 0 when no word overlaps the range; callers treat that the same as
// "field not extracted". len(Word) is a byte length, matching the byte offsets
// produced by the regexes in medical_parser.go.
func minConfidence(words []WordBox, start, end int) float64 {
	if start >= end {
		return 0
	}
	min := 0.0
	found := false
	for _, w := range words {
		wStart := w.Offset
		wEnd := w.Offset + len(w.Word)
		// Half-open interval overlap test.
		if wStart < end && start < wEnd {
			if !found || w.Confidence < min {
				min = w.Confidence
				found = true
			}
		}
	}
	if !found {
		return 0
	}
	return min
}

// overallConfidence returns the minimum across all recorded field confidences,
// or 0 when the map is empty (nothing was extracted). Min is used — rather than
// mean — so a single badly-read field pulls the whole document into review.
func overallConfidence(fieldConfidences map[string]float64) float64 {
	min := 0.0
	first := true
	for _, c := range fieldConfidences {
		if first || c < min {
			min = c
			first = false
		}
	}
	return min
}

// ShouldReview reports whether a document should be flagged for manual review:
// its overall confidence is positive (something was extracted) yet below the
// configured threshold. An overall confidence of 0 means nothing was extracted
// at all — an OCR failure rather than a review case — so it is never flagged.
func ShouldReview(overallConfidence, threshold float64) bool {
	return overallConfidence > 0 && overallConfidence < threshold
}
