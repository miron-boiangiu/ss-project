# OCR Extraction Subsystem — Design

## 1. Overview

The OCR extraction subsystem turns photographed Romanian *Fișă de Aptitudine*
(occupational-medicine aptitude forms) into structured JSON that the rest of
the platform can store, search, review, and audit. It runs inside the `go-api`
container: the MQTT broker handler in `server/broker/broker.go` receives an
image, runs Tesseract OCR with verbose bounding boxes, hands the joined text
and per-word boxes to the regex-based parser in `server/utils/medical_parser.go`,
and persists a flattened `Photo` row in PostgreSQL. Each extraction also
carries per-field confidence scores and an aggregate "needs review" flag that
the frontend surfaces on the photos page. The output contract is a versioned
JSON Schema kept in `server/schema/medical-data.v1.json`.

## 2. OCR engine choice

We use **Tesseract** (via the [gosseract](https://github.com/otiai10/gosseract)
CGo bindings, Romanian language pack `tesseract-ocr-data-ron`). The
alternatives we evaluated were Mistral OCR (hosted API), EasyOCR (PyTorch),
and PaddleOCR (PaddlePaddle). The decision was driven by four constraints
from the TARA threat model rather than by raw accuracy.

**PHI sovereignty.** Patient names, CNPs (Romanian national IDs), and medical
opinions are PHI. Anything sent to a hosted OCR API (Mistral OCR, AWS Textract,
Google Document AI, …) becomes a third-party data flow we'd have to model in
the threat analysis, contract for, and audit. Tesseract runs locally; the
image never leaves the network boundary, no per-document egress event exists
in the first place. EasyOCR and PaddleOCR are also local, so they tie on this
dimension — but they lose on the others below.

**Sandboxability.** The TARA model treats the OCR engine as an
attacker-controlled input handler: a crafted image could exploit a parser
vulnerability in the engine's image decoder. Tesseract is a small C++ codebase
with a long deployment history, packageable into a distroless or gVisor-wrapped
container with no Python runtime, no GPU driver surface, and no torch/paddle
dynamic-graph machinery. PyTorch/PaddlePaddle pull in megabytes of
shared-object surface area we don't need and don't want to attest.

**Offline operation.** The deployment target is a hospital network that
cannot assume outbound internet. Mistral OCR, being hosted, is disqualified
outright by this constraint regardless of accuracy.

**CPU-only.** No GPUs in the target environment, and the demo runs in Docker
Compose on a developer laptop. Tesseract is happy on CPU; EasyOCR and
PaddleOCR run on CPU but are noticeably slower and pull in heavier dependency
trees per the sandboxability point above.

Mistral OCR and PaddleOCR would almost certainly produce higher per-word
confidence and better handling of skewed/blurry forms than Tesseract. We
judged the accuracy uplift not worth the PHI-egress, sandboxing, and offline
tradeoffs for v1 of this subsystem. The threshold and review-queue design
(sections 5–6) is the compensating control: low-confidence extractions are
flagged for human review rather than silently mis-recorded.

## 3. Output schema

The output contract is a JSON Schema (draft 2020-12) at
[`../server/schema/medical-data.v1.json`](../server/schema/medical-data.v1.json),
embedded into the Go binary by `server/schema/schema.go` as
`schema.MedicalDataV1`. Anything that needs the schema bytes at runtime — the
test suite today, a future runtime validator tomorrow — pulls them from that
package rather than re-reading the file.

Structurally the schema is an object with five required keys
(`document_type`, `schema_version`, `overall_confidence`, `needs_review`,
`field_confidences`) and 20+ optional medical fields covering header info
(unit, address, telephone, form number), employer info, personal data (name,
surname, CNP), professional data (profession, workplace), the control-type
checkbox set (`control_angajare`, `control_periodic`, …), the medical-opinion
checkbox set (`aviz_apt`, `aviz_apt_conditionat`, …), the textual opinion
(`aviz_medical`), recommendations, and two date fields. `field_confidences`
is a map of snake_case field name to a number in `[0, 100]`. Top-level
`additionalProperties: false` keeps unknown fields out — strict by design so
schema drift surfaces as a test failure, not silently.

**Versioning policy.** The schema file name is `medical-data.v1.json` and the
`schema_version` field starts at `"1.0.0"` (semver, pattern
`^\d+\.\d+\.\d+$`). Backward-compatible additions (new optional fields, new
enum members in `document_type`) bump the minor version. Breaking changes
(removing or renaming a field, tightening a constraint) ship as
`medical-data.v2.json` alongside the v1 file, with `document_type` discriminating
the two. The expected first v2 case is a prescription document class
(`document_type: "prescription"`) which would introduce a medication field
that the aptitude form does not have.

## 4. Spec mapping

The project spec phrases task 2.B as extracting *"(Nume Pacient, Medicație,
Data Expirării)"* in a standardized JSON format. That triple is an
**illustrative example** of structured extraction, not a binding schema; the
actual document we process is a Romanian *Fișă de Aptitudine*, which has no
medication field at all (medication belongs to a prescription document, a
separate class). The mapping we use:

- **Nume Pacient** → `nume` + `prenume` (Romanian forms split given and family
  names into separate labelled fields).
- **Medicație** → not applicable to this document class; would belong to a
  future `document_type: "prescription"` schema (see section 3).
- **Data Expirării** → `data_urm_examinari` ("date of next examination",
  i.e. the expiry of the current medical clearance).

In addition to those three concept slots we extract 20+ further structured
fields: `cnp` (national ID), employer and medical-unit identification and
contact details, profession, workplace, the six control-type checkboxes and
their derived `tip_control` string, the four medical-opinion checkboxes and
their derived `aviz_medical` string, free-text recommendations, and the
extraction date. The schema documents the full set.

## 5. Confidence model

For each regex-extracted text field, the parser computes a per-field
confidence by mapping the regex match indices back to the source word stream.
Tesseract's `gosseract.GetBoundingBoxesVerbose()` returns one record per word
with the engine's confidence (0–100) and Tesseract's own block/paragraph/line
numbers; the broker reconstructs the joined text so that a `\n` is inserted
wherever the `(BlockNum, ParNum, LineNum)` tuple changes (preserving the line
boundaries the parser regexes rely on), and records each word's byte offset
into that joined text. See `server/utils/ocr_types.go` for the `WordBox`
type and the `minConfidence` / `overallConfidence` / `ShouldReview` helpers.

`minConfidence(words, start, end)` returns the **minimum** Tesseract
confidence across the words overlapping the regex's matched span. We use min
rather than mean because the failure mode we care about is a single garbled
character — one bad digit in a 13-digit CNP makes the whole CNP wrong, and
that bad digit will pull the per-field min down while leaving the mean nearly
unchanged. The same logic applies field-to-document: `overall_confidence` is
the min across all populated `field_confidences` entries.

**Zero-confidence-as-not-extracted convention.** `minConfidence` returns 0
both when no word overlapped the span (no extraction) and when every overlapping
word had a Tesseract confidence of 0 (extraction is junk we cannot trust). We
treat both as "we have no useful information about this field" and omit the
field from `field_confidences` entirely. `overall_confidence` is therefore a
min over only the fields we extracted with positive confidence; the absence
of a field name from the map is itself the signal. `needs_review` is set when
`overall_confidence > 0 && overall_confidence < threshold` — the `> 0` guard
keeps a document where nothing was extracted (OCR failure) from being mistaken
for a low-confidence document worth reviewing.

## 6. Threshold

The review threshold is configured via the `OCR_REVIEW_THRESHOLD` environment
variable, read in `server/main.go` and threaded through to the broker handler.
Default is `80`.

The project spec literally specifies 95%. That figure is appropriate for
higher-quality engines — Mistral OCR, AWS Textract, or Google Document AI
routinely produce per-word confidences above 95 on legibly-printed text — but
not for Tesseract on photographed Romanian medical forms. In our empirical
testing, Tesseract per-word confidence on legibly-printed sample images
clusters in the **70–90 range**; perfectly readable form rows often score
75–85. A literal 95% threshold would flag essentially every document, which
defeats the purpose of the flag.

`80` was calibrated empirically against the sample form
(`medical-images/Screenshot 2026-01-20 at 13.57.50.png`) and a handful of
deliberately-blurred variants: at 80, clean photographs do not always pass
(Tesseract is still noisy on Romanian diacritics), but blurred or distorted
photographs reliably fall under. Operators and demo presenters can retune the
threshold at startup without rebuilding by setting
`OCR_REVIEW_THRESHOLD=<value>` in `.env`; the demo plan exercises both 80
and 95 to show the trade-off.

## 7. Known limitations

The following are known limitations of v1 of this subsystem:

- **Date fields often come back empty.** Both `data` and `data_urm_examinari`
  may be empty strings even when the form clearly shows them. The parser's
  date regex doesn't match the formats Tesseract emits for these forms in
  practice. The JSON Schema therefore declines to constrain the format of
  these fields beyond `"type": "string"`, so the absence of a date does not
  fail validation.
- **Field values may include trailing label fragments.** Regex capture groups
  for text fields are greedy enough to sometimes pull in the leading token of
  the next field's label (e.g. `nume: "Vaduva NUME"`). The schema treats these
  as opaque strings; consumers should expect occasional trailing-token noise
  and not parse the values further without sanitization.
- **Lower-section binary fields are not currently reliable.** The medical
  opinion checkboxes (`aviz_apt`, `aviz_apt_conditionat`, `aviz_inapt_temporar`,
  `aviz_inapt`) are extracted on `MedicalData` but the postgres migration
  intentionally does not persist them on the `photos` table — their extraction
  rate is too low to be useful. The derived textual `aviz_medical` field is
  what consumers should rely on for now.
- **Two-copy form layouts may leak content across copies.** The standard
  Romanian *Fișă de Aptitudine* is printed as two identical copies side-by-side
  on the same page (one for the employer, one for the worker). Tesseract reads
  both copies; the parser tries to scope to the left copy via a "Societate"
  split, but content from the right copy can still leak into header fields
  like `unitate_medicala` when the left/right split is imperfect.

All four are tracked for a future iteration. Per-field confidence in v1
already exposes these failure modes to reviewers — fields with leaked or
truncated content tend to score low and either get omitted from
`field_confidences` (zero-confidence words) or land in the "needs review"
bucket.

## 8. Sandboxing & PHI

OCR runs inside the `go-api` Docker container. Images and extracted text
never leave the compose network during normal operation: storage is on the
local volume and PostgreSQL, the encryption-at-rest column set is owned by
task 2.C, and there are no outbound HTTP calls from the OCR path. Log
redaction is implemented in the broker (`server/broker/broker.go`): the
extracted-data log line emits only field counts, the overall confidence, and
the `needs_review` flag — never field values. A `docker logs go-api | grep`
for patient identifiers returns nothing.

Hardening beyond this — building `go-api` from a distroless base image,
running the Tesseract subprocess inside a gVisor sandbox, and adding mTLS
between services — is part of the broader TARA hardening (tasks 3.x) and is
explicitly out of scope for the OCR PR series. The current model assumes the
`go-api` container itself is trusted; future work will narrow that
assumption.
