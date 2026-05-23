package utils

import (
	"strings"
	"testing"
)

// sampleCertificate is a synthetic OCR text whose layout and labels mirror a
// Fișă de Aptitudine closely enough for the real parser regexes to match. It
// has a "Societate" marker so topPart/bottomPart handling is exercised.
const sampleCertificate = `UNITATEA MEDICALA: Clinica Medicala SRL
ADRESA: Strada Sanatatii 10
TEL: 0211234567
FISA DE APTITUDINE NR. 42
Societate, unitate, etc: Acme Industries SRL
Adresa: Bulevardul Muncii 5
Telefon: 0219876543
NUME: Popescu
PRENUME: Ion
CNP: 1234567890123
Profesie / functie: Inginer
Locul de munca: Hala Productie`

// splitDoc puts an extracted field three lines below the "Societate" marker so
// the topPart/bottomPart offset translation can be checked in isolation.
const splitDoc = `ADRESA: Strada Veche 1
Societate, unitate, etc: Acme
filler line between sections
Telefon: 0712345678`

// wordsFromText tokenises text on whitespace into WordBoxes whose Offset is the
// byte position of each word — mirroring how the broker builds the joined OCR
// text and word list. Every word gets defaultConf; tests override individual
// words afterwards via setConf.
func wordsFromText(text string, defaultConf float64) []WordBox {
	var words []WordBox
	i := 0
	for i < len(text) {
		for i < len(text) && (text[i] == ' ' || text[i] == '\n' || text[i] == '\t') {
			i++
		}
		if i >= len(text) {
			break
		}
		start := i
		for i < len(text) && text[i] != ' ' && text[i] != '\n' && text[i] != '\t' {
			i++
		}
		words = append(words, WordBox{
			Word:       text[start:i],
			Confidence: defaultConf,
			Offset:     start,
		})
	}
	return words
}

// setConf sets the confidence of every word equal to target. It fails loudly if
// target matches nothing, so a typo in a test can't silently pass.
func setConf(t *testing.T, words []WordBox, target string, conf float64) {
	t.Helper()
	found := false
	for i := range words {
		if words[i].Word == target {
			words[i].Confidence = conf
			found = true
		}
	}
	if !found {
		t.Fatalf("setConf: no word %q in the word list", target)
	}
}

func TestParseMedicalCertificate_CleanExtraction(t *testing.T) {
	words := wordsFromText(sampleCertificate, 92)
	data := ParseMedicalCertificate(sampleCertificate, words)

	if data == nil {
		t.Fatal("ParseMedicalCertificate returned nil")
	}
	if data.OverallConfidence != 92 {
		t.Errorf("OverallConfidence = %v, want 92", data.OverallConfidence)
	}
	if ShouldReview(data.OverallConfidence, 80) {
		t.Error("ShouldReview = true, want false for an all-high-confidence document")
	}
	// Spot-check fields drawn from ocrText, topPart and bottomPart.
	for _, key := range []string{"unitate_medicala", "cnp", "nume", "telefon_angajator"} {
		if c, ok := data.FieldConfidences[key]; !ok {
			t.Errorf("FieldConfidences[%q] missing", key)
		} else if c != 92 {
			t.Errorf("FieldConfidences[%q] = %v, want 92", key, c)
		}
	}
	if len(data.FieldConfidences) < 10 {
		t.Errorf("extracted %d fields, want at least 10", len(data.FieldConfidences))
	}
}

func TestParseMedicalCertificate_OneFieldBelowThreshold(t *testing.T) {
	words := wordsFromText(sampleCertificate, 92)
	setConf(t, words, "1234567890123", 60) // the CNP value word
	data := ParseMedicalCertificate(sampleCertificate, words)

	if got := data.FieldConfidences["cnp"]; got != 60 {
		t.Errorf("FieldConfidences[\"cnp\"] = %v, want 60", got)
	}
	if data.OverallConfidence != 60 {
		t.Errorf("OverallConfidence = %v, want 60 (driven down by the cnp field)", data.OverallConfidence)
	}
	if !ShouldReview(data.OverallConfidence, 80) {
		t.Error("ShouldReview = false, want true (overall 60 < threshold 80)")
	}
	if got := data.FieldConfidences["nume"]; got != 92 {
		t.Errorf("FieldConfidences[\"nume\"] = %v, want 92 (unrelated field must be unaffected)", got)
	}
}

func TestParseMedicalCertificate_FieldMissingEntirely(t *testing.T) {
	// Drop the CNP line so its regex never matches.
	text := strings.ReplaceAll(sampleCertificate, "CNP: 1234567890123\n", "")
	words := wordsFromText(text, 92)
	data := ParseMedicalCertificate(text, words)

	if _, ok := data.FieldConfidences["cnp"]; ok {
		t.Error("FieldConfidences[\"cnp\"] present, want absent (field not in document)")
	}
	if data.CNP != "" {
		t.Errorf("CNP = %q, want empty (regex did not match)", data.CNP)
	}
	if data.OverallConfidence != 92 {
		t.Errorf("OverallConfidence = %v, want 92 (a missing field must not lower it)", data.OverallConfidence)
	}
	if ShouldReview(data.OverallConfidence, 80) {
		t.Error("ShouldReview = true, want false")
	}
}

func TestParseMedicalCertificate_AllFieldsMissing(t *testing.T) {
	text := "no recognizable medical fields are present in this line of text"
	words := wordsFromText(text, 92)
	data := ParseMedicalCertificate(text, words)

	if data == nil {
		t.Fatal("ParseMedicalCertificate returned nil, want non-nil even when nothing is extracted")
	}
	if len(data.FieldConfidences) != 0 {
		t.Errorf("FieldConfidences = %v, want empty", data.FieldConfidences)
	}
	if data.OverallConfidence != 0 {
		t.Errorf("OverallConfidence = %v, want 0", data.OverallConfidence)
	}
	if ShouldReview(data.OverallConfidence, 80) {
		t.Error("ShouldReview = true, want false (nothing extracted is not a review case)")
	}
}

func TestParseMedicalCertificate_ZeroConfidenceOverlap(t *testing.T) {
	// The cnp regex matches, but every word overlapping the captured span has
	// confidence 0 — so the field counts as "not extracted": the value is still
	// returned, but no FieldConfidences entry is recorded.
	words := wordsFromText(sampleCertificate, 92)
	setConf(t, words, "1234567890123", 0)
	data := ParseMedicalCertificate(sampleCertificate, words)

	if _, ok := data.FieldConfidences["cnp"]; ok {
		t.Error("FieldConfidences[\"cnp\"] present, want absent (overlapping words are all 0-confidence)")
	}
	if data.CNP != "1234567890123" {
		t.Errorf("CNP = %q, want \"1234567890123\" (value is still extracted)", data.CNP)
	}
	if data.OverallConfidence != 92 {
		t.Errorf("OverallConfidence = %v, want 92 (the 0-conf field is excluded, not collapsing the score)", data.OverallConfidence)
	}
}

func TestParseMedicalCertificate_TopBottomSplit(t *testing.T) {
	// telefon_angajator is extracted from bottomPart only; its match index is
	// relative to bottomPart. Lowering its value word must still register,
	// which only works if the bottomPart->global offset translation is correct.
	words := wordsFromText(splitDoc, 92)
	setConf(t, words, "0712345678", 55)
	data := ParseMedicalCertificate(splitDoc, words)

	if got := data.FieldConfidences["telefon_angajator"]; got != 55 {
		t.Errorf("FieldConfidences[\"telefon_angajator\"] = %v, want 55 "+
			"(a wrong value means the bottomPart->global offset translation is broken)", got)
	}
	// adresa_unitate_medicala comes from topPart (base 0) and must stay default.
	if got := data.FieldConfidences["adresa_unitate_medicala"]; got != 92 {
		t.Errorf("FieldConfidences[\"adresa_unitate_medicala\"] = %v, want 92", got)
	}
}

func TestParseMedicalCertificate_NilOnEmptyInput(t *testing.T) {
	for _, in := range []string{"", "OCR failed"} {
		if data := ParseMedicalCertificate(in, nil); data != nil {
			t.Errorf("ParseMedicalCertificate(%q, nil) = %+v, want nil", in, data)
		}
	}
}

func TestMinConfidence(t *testing.T) {
	// Word spans: alpha[0,5) bravo[6,11) gamma[12,17) delta[18,23)
	words := []WordBox{
		{Word: "alpha", Confidence: 90, Offset: 0},
		{Word: "bravo", Confidence: 70, Offset: 6},
		{Word: "gamma", Confidence: 0, Offset: 12},
		{Word: "delta", Confidence: 55, Offset: 18},
	}
	tests := []struct {
		name       string
		start, end int
		want       float64
	}{
		{"full overlap of one word", 6, 11, 70},
		{"partial overlap of two words", 3, 8, 70},
		{"no overlap (gap between words)", 5, 6, 0},
		{"no overlap (past last word)", 23, 30, 0},
		{"multi-word overlap including a zero word", 0, 23, 0},
		{"overlap a zero-confidence word", 12, 17, 0},
		{"multi-word overlap excluding the zero word", 0, 11, 70},
		{"empty range", 10, 10, 0},
		{"inverted range", 15, 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := minConfidence(words, tt.start, tt.end); got != tt.want {
				t.Errorf("minConfidence(words, %d, %d) = %v, want %v", tt.start, tt.end, got, tt.want)
			}
		})
	}
}

func TestOverallConfidence(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]float64
		want float64
	}{
		{"nil map", nil, 0},
		{"empty map", map[string]float64{}, 0},
		{"single entry", map[string]float64{"a": 80}, 80},
		{"min of several", map[string]float64{"a": 80, "b": 60, "c": 95}, 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := overallConfidence(tt.in); got != tt.want {
				t.Errorf("overallConfidence(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestShouldReview(t *testing.T) {
	const threshold = 80.0
	tests := []struct {
		name              string
		overallConfidence float64
		want              bool
	}{
		{"just below threshold", 79.99, true},
		{"exactly at threshold", 80.00, false},
		{"just above threshold", 80.01, false},
		{"well below threshold", 50, true},
		{"zero — nothing extracted", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldReview(tt.overallConfidence, threshold); got != tt.want {
				t.Errorf("ShouldReview(%v, %v) = %v, want %v", tt.overallConfidence, threshold, got, tt.want)
			}
		})
	}
}

func TestExtractField(t *testing.T) {
	const text = "CNP: 1234567890123 end"

	value, start, end := extractField(text, `(?i)CNP[:;]?\s*(\d+)`)
	if value != "1234567890123" {
		t.Errorf("value = %q, want \"1234567890123\"", value)
	}
	if got := text[start:end]; got != "1234567890123" {
		t.Errorf("text[%d:%d] = %q, want the captured group span", start, end, got)
	}

	value, start, end = extractField(text, `(?i)NUME[:;]?\s*([A-Za-z]+)`)
	if value != "" || start != 0 || end != 0 {
		t.Errorf("no-match result = (%q, %d, %d), want (\"\", 0, 0)", value, start, end)
	}
}
