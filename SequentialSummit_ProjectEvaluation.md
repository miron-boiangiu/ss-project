# Sequential Summit

## 1. Cover Page

### Project Title
**Sequential Summit** — IoT Image Capture & Medical Data Processing Platform

### Team Members

| Name | Email |
|------|-------|
| Victor-Miron Boiangiu | miron.boiangiu@gmail.com |
| Andrei Gorneanu | andreigorneanu@gmail.com |
| Teodor Suteu | teodor.suteu07@gmail.com |
| Vlad Grigore | grigorevlad2000@yahoo.com |
| Andrei Seceleanu | seceleanuandrei278@gmail.com |

---

## 2. Project Summary

### Project Overview

**Sequential Summit** is an IoT image capture platform that receives photos from devices (ESP32-CAM, Android, Python scripts) via MQTT, performs OCR using Tesseract to extract structured data from Romanian medical aptitude certificates (*Fișa de Aptitudine*), and provides a web interface for browsing, searching, and managing captured images with role-based access control.

**Key Technologies:**
- **Backend:** Go 1.25, PostgreSQL, Mosquitto MQTT
- **Frontend:** React 19, TypeScript, Vite, TailwindCSS, Recharts
- **OCR:** Tesseract (Romanian language pack) via gosseract
- **Security:** mTLS for MQTT, JWT authentication, AES-256-GCM encryption, bcrypt, RBAC
- **CI/CD:** GitHub Actions, Docker Compose
- **Cloud (optional):** AWS S3

**Architecture:**
```
IoT Devices → MQTT Broker (Mosquitto, mTLS) → Go API (OCR + Parser) → PostgreSQL → React Frontend
```

### Default OSSF Criticality Score Result

**Score: `0.19156`** (out of ~1.0)


| Signal | Value |
|---|---|
| `created_since` | 2 months |
| `updated_since` | 0 months |
| `contributor_count` | 5 |
| `org_count` | 1 |
| `commit_frequency` | 0.65/day |
| `recent_release_count` | 0 |
| `star_count` | 0 |
| `license` | none |
| `open_issues` / `closed_issues` | 0 / 0 |
| `issue_comment_frequency` | 0 |
| `github_mention_count` | 0 |

---

## 3. Functionality, Documentation, Execution

### Working Features

| Feature | Description | Status |
|---------|-------------|--------|
| MQTT Image Ingestion | Devices send images via MQTT (`ssproject/images/{device_id}`) | ✅ Implemented |
| Device Registration | Devices self-register via `register/{device_id}` topic | ✅ Implemented |
| OCR Text Extraction | Tesseract OCR with Romanian language support | ✅ Implemented |
| Medical Certificate Parsing | Extracts structured data (CNP, name, medical opinion, etc.) | ✅ Implemented |
| JSON Schema Validation | Validates parsed output against `medical-data.v1.json` | ✅ Implemented |
| Per-Field Confidence Scoring | Confidence score per extracted field with review threshold | ✅ Implemented |
| JWT Authentication | Login/register with HMAC-SHA256 JWT tokens (24h expiry) | ✅ Implemented |
| RBAC | Role-based access (user = read-only, admin = full access) | ✅ Implemented |
| mTLS for MQTT | Mutual TLS authentication for MQTT (port 8883) | ✅ Implemented |
| AES-256-GCM Encryption | Encrypted PHI fields (CNP, names, addresses, etc.) | ✅ Implemented |
| Reports API | Medical reports (expirations, anonymized, performance, statistics) | ✅ Implemented |
| Dark Mode | UI dark mode support | ✅ Implemented |
| CI/CD Pipeline | GitHub Actions build, test, lint, Docker build | ✅ Implemented |
| SBOM Generation | CycloneDX SBOM of Go dependencies | ✅ Implemented |


### Developer & User Documentation

| Document | Description | Location |
|----------|-------------|----------|
| mTLS MQTT Design | mTLS implementation design doc | `docs/mtls-mqtt.md` |
| RBAC Design | Role-based access control design | `docs/rbac.md` |
| OCR Extraction Design | OCR subsystem design with JSON Schema | `docs/ocr-extraction.md` |
| Frontend README | Frontend-specific setup and usage | `client/README.md` |
| Medical Images Guide | Guide for uploading medical certificate images | `medical-images/README.md` |

### CI/CD Evidence

- **GitHub Actions** workflow defined in `.github/workflows/ci.yml`
- Triggers on push / PR to `main` branch
- Jobs:
  1. Go backend: `go build`, `go test` with coverage
  2. React frontend: `yarn lint`, `yarn build`
  3. Docker build for API image
- Artifacts: test results and coverage reports uploaded

#### Execution Steps

```bash
# 1. Clone and setup
git clone <repo>
cp ENV_EXAMPLE .env
./scripts/generate-certs.sh

# 2. Start services
docker compose up --build

# 3. Start frontend (separate terminal)
cd client && yarn install && yarn dev

# 4. Send test images
python3 scripts/upload_folder.py medical-images/
```

API: `http://localhost:8080`  
Frontend: `http://localhost:5173`  
MQTT Broker: `localhost:8883` (mTLS)

---

## 4. Security & Compliance

### Threat Modeling & Mitigations

> **TODO** 
>
**Currently implemented security mitigations:**
- mTLS for MQTT (mutual certificate verification, `require_certificate true`)
- JWT authentication with 24h token expiry and HMAC-SHA256 signing
- RBAC (user/admin roles)
- AES-256-GCM encryption for sensitive PHI fields at rest
- bcrypt password hashing
- CORS configuration

### MISRA / CERT Compliance

Static analysis was performed using `go vet`, `staticcheck`, `golangci-lint`, and additional Go analysis tools.

#### Analysis Summary by Tool

**1. `go vet ./...`**
```
(no issues found)
```

**2. `staticcheck ./...`**
```
utils/medical_parser.go:334:4: this value of aptIdx is never used (SA4006)
utils/medical_parser.go:451:5: should omit nil check; len() for nil slices is defined as zero (S1009)
utils/medical_parser.go:462:5: should omit nil check; len() for nil slices is defined as zero (S1009)
utils/medical_parser.go:473:6: func containsChecked is unused (U1000)
utils/medical_parser.go:497:2: should use 'return regexp.MustCompile(patternLeft).MatchString(text)' instead of 'if regexp.MustCompile(patternLeft).MatchString(text) { return true }; return false' (S1008)
```

**3. `golangci-lint run ./...`**
```
routes/device.go:81:27: Error return value of `(*encoding/json.Encoder).Encode` is not checked (errcheck)
routes/device.go:122:27: Error return value of `(*encoding/json.Encoder).Encode` is not checked (errcheck)
routes/init.go:72:27: Error return value of `(*encoding/json.Encoder).Encode` is not checked (errcheck)
main.go:58:23: Error return value of `ocrClient.SetLanguage` is not checked (errcheck)
utils/medical_parser.go:473:6: func `containsChecked` is unused (unused)
utils/medical_parser.go:497:2: S1008: should use 'return ...' instead of 'if ... { return true }; return false' (gosimple)
utils/medical_parser.go:451:5: S1009: should omit nil check; len() for nil slices is defined as zero (gosimple)
utils/medical_parser.go:462:5: S1009: should omit nil check; len() for nil slices is defined as zero (gosimple)
repository/photo_repository.go:57:3: ineffectual assignment to argIdx (ineffassign)
utils/medical_parser.go:329:3: ineffectual assignment to inaptIdx (ineffassign)
utils/medical_parser.go:334:4: ineffectual assignment to aptIdx (ineffassign)
utils/medical_parser.go:402:11: SA9003: empty branch (staticcheck)
```

**4. Variable Shadowing (`go vet -vettool=$(which shadow)`)**
```
broker/broker.go:71:7: declaration of "err" shadows declaration at line 62
routes/user.go:96:5: declaration of "err" shadows declaration at line 89
main.go:47:5: declaration of "err" shadows declaration at line 40
utils/medical_parser_schema_test.go:28:5: declaration of "err" shadows declaration at line 22
```

**5. Cyclomatic Complexity (`gocyclo -over 15 .`)**
```
45 utils ParseMedicalCertificate utils/medical_parser.go:78:1
16  main main                       main.go:24:1
```

**6. Formatting Issues (`gofmt -d`)**
- `broker/broker.go` — misaligned struct field (indentation)
- `routes/init.go` — import ordering + mixed tabs/spaces
- `routes/reports.go` — missing trailing newline
- `utils/storage.go` — trailing whitespace on blank lines

#### Issues Summary

| Category | Count | Severity |
|----------|-------|----------|
| Unchecked errors | 4 | Medium |
| Unused code/variables | 4 | Low |
| Variable shadowing | 4 | Low |
| Empty branch | 1 | Low |
| Style/simplification | 4 | Low |
| Cyclomatic complexity (>15) | 2 | Low |
| Formatting | 4 files | Style |

### Testing & Coverage

**Test Statistics:**
- Total test functions: **42** (across 6 test files)
- Test files: `broker_test.go`, `device_test.go`, `photo_test.go`, `user_test.go`, `medical_parser_test.go`, `medical_parser_schema_test.go`
- Mock package: Generated mocks at `server/mocks/domain_mock.go`

| Package | Test File | Tests |
|---------|-----------|-------|
| broker | `broker/broker_test.go` | Device registration & disconnection (8 tests) |
| routes | `routes/device_test.go` | Device API endpoints |
| routes | `routes/photo_test.go` | Photo API endpoints |
| routes | `routes/user_test.go` | User auth endpoints |
| utils | `utils/medical_parser_test.go` | OCR parsing logic |
| utils | `utils/medical_parser_schema_test.go` | JSON Schema validation |

**Code Coverage (obtained `2026-05-24`):**

```
$ go test $(go list ./... | grep -v -E 'schema|mocks|domain') -coverprofile=coverage.out && go tool cover -func=coverage.out

ok  	mqtt-streaming-server/broker	0.008s	coverage: 29.1% of statements
ok  	mqtt-streaming-server/routes	0.798s	coverage: 39.9% of statements
ok  	mqtt-streaming-server/utils	0.011s	coverage: 34.8% of statements

total:	(statements)			26.1%
```

| Package | Coverage |
|---------|----------|
| `broker` | 29.1% |
| `routes` | 39.9% |
| `utils` | 34.8% |
| `repository` | 0.0% (no tests) |
| **Overall** (excl. schema/mocks/domain) | **26.1%** |

**Testing Strategy:**
- Unit tests for OCR parsing and medical data extraction
- Route/API tests with HTTP handlers
- JSON Schema validation tests
- Mock-based testing for repository layer

**Fuzzing:**
- Implemented `FuzzParseMedicalCertificate` in `server/utils/medical_parser_fuzz_test.go`
- Ran 108,599 executions over 11s with 4 workers, discovering 93 interesting inputs — all PASS
- Seed corpus includes real fixtures, edge cases (empty, "OCR failed"), and garbage input
- Invariants verified: no panics, `DocumentType`/`SchemaVersion` always set, confidences in `[0, 100]`, `OverallConfidence` consistent with field map
> ```bash
> go test -fuzz=FuzzParseMedicalCertificate -fuzztime=1h ./utils/
> ```

### SBOM & Dependencies

**SBOM Report:**
- Format: CycloneDX 1.6
- Generator: Syft 1.26.1
- Location: `server/sbom.cdx.json`
- Scope: Go module dependencies

**Key Dependencies:**
| Dependency | Version | Purpose |
|------------|---------|---------|
| github.com/eclipse/paho.mqtt.golang | v1.5.1 | MQTT client |
| github.com/golang-jwt/jwt/v4 | v4.5.2 | JWT auth |
| github.com/jackc/pgx/v5 | v5.9.2 | PostgreSQL driver |
| github.com/otiai10/gosseract/v2 | v2.4.1 | Tesseract OCR |
| github.com/santhosh-tekuri/jsonschema/v6 | v6.0.2 | JSON Schema validation |
| golang.org/x/crypto | v0.45.0 | bcrypt + crypto utilities |
| github.com/aws/aws-sdk-go-v2 | v1.41.5 | AWS S3 integration |

> Vulnerability scan run against the SBOM (via `grype dir:. --only-fixed`):
> ```
> No vulnerabilities found
> ```

### Fixing Own Vulnerabilities

All 6 vulnerabilities found by Grype in the SBOM scan were resolved via dependency upgrades in `server/go.mod`. After all upgrades, `grype dir:. --only-fixed` reports **No vulnerabilities found**.

| Dependency | Old Version | New Version | Vulnerabilities Fixed |
|---|---|---|---|
| `github.com/eclipse/paho.mqtt.golang` | v1.5.0 | v1.5.1 | GHSA-32fw-gq77-f2f2 (Medium) |
| `golang.org/x/crypto` | v0.38.0 | v0.45.0 | GHSA-j5w8-q4qc-rx2x (Medium), GHSA-f6x5-jh6r-wrfv (Medium) |
| `github.com/aws/aws-sdk-go-v2/service/s3` | v1.79.4 | v1.97.3 | GHSA-xmrv-pmrh-hhx2 (Medium) |
| `github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream` (indirect) | v1.6.10 | v1.7.8 | GHSA-xmrv-pmrh-hhx2 (Medium) |
| `github.com/jackc/pgx/v5` | v5.7.4 | v5.9.2 | GHSA-9jj7-4m8r-rfcm (Critical), GHSA-j88v-2chj-qfwx (Low) |

**Fix:** Ran `go get <dependency>@<patched-version>` for each vulnerable package, followed by `go mod tidy` to clean up the dependency graph. Verified with `go build ./...`, `go vet ./...`, `go test ./...`, and a final Grype scan.

## 5. Team Contributions

Git contribution statistics gathered via `git shortlog` and `git log --numstat --all`.

| Team Member | Lines Added | Lines Removed | Number of Commits |
|-------------|------------|---------------|-------------------|
| Victor-Miron Boiangiu | 13,724 | 863 | 10 |
| Andrei Gorneanu | 1,771 | 281 | 23 |
| Teodor Suteu | 2,581 | 427 | 10 |
| Vlad Grigore | 1,190 | 82 | 5 |
| Andrei Seceleanu | 749 | 551 | 4 |

---

## 6. Medical Reports

The system provides a **Reports API** with the following report types, accessible via `GET /api/reports?type=<report_type>`:

| Report Type | Endpoint | Description |
|-------------|----------|-------------|
| `expirari` | `/api/reports?type=expirari` | Medical records expiring within 30 days (decrypted PHI) |
| `anonimizate` | `/api/reports?type=anonimizate` | Anonymized dataset without PHI for research |
| `performanta` | `/api/reports?type=performanta` | System performance: photo counts per device |
| `activitate_1m` | `/api/reports?type=activitate_1m` | Last 30 days activity with medical status breakdown |
| `statistici` | `/api/reports?type=statistici` | All-time statistics (aviz medical, tip control) |

### Report Details

#### 1. Expirari Report (`expirari`)
- Fetches records where `data_urm_examinari` falls within the next 30 days
- Decrypts PHI fields (name, prenume) using AES-256-GCM
- Returns: `id`, `nume`, `prenume`, `aviz_medical`, `data_urm_examinari`

#### 2. Anonymized Report (`anonimizate`)
- Excludes all personal identifiable information (name, CNP, etc.)
- Returns: `id`, `timestamp`, `aviz_medical`, `control_angajare`, `control_periodic`

#### 3. Performance Report (`performanta`)
- Groups by `device_id` and counts total photos processed
- Returns: `device_id`, `total_photos`
- *Future: average OCR latency and confidence*

#### 4. Monthly Activity Report (`activitate_1m`)
- Aggregates medical checks from the last 30 days
- Returns: `total_last_month`, `status_breakdown` (APT, APT CONDITIONAT, etc.)

#### 5. General Statistics Report (`statistici`)
- All-time aggregate using PostgreSQL `FILTER (WHERE ...)`:
  - Total documents processed
  - Aviz medical breakdown (APT, APT CONDITIONAT, INAPT TEMPORAR, INAPT, NEPROCESAT)
  - Control type breakdown (angajare, periodic, adaptare, reluare, supraveghere, alte)

### Structured Medical Data Extracted (JSON Schema v1.0.0)

The OCR parser extracts the following fields from Romanian medical aptitude certificates:

| Category | Fields |
|----------|--------|
| Unitate Medicală | `unitate_medicala`, `adresa_unitate_medicala`, `telefon_unitate_medicala` |
| Angajator | `societate_unitate`, `adresa_angajator`, `telefon_angajator` |
| Date Personale | `nume`, `prenume`, `cnp` |
| Date Profesionale | `profesie_functie`, `loc_de_munca` |
| Control Medical | `tip_control`, `control_angajare`, `control_periodic`, `control_adaptare`, `control_reluare`, `control_supraveghere`, `control_alte` |
| Aviz Medical | `aviz_medical`, `aviz_apt`, `aviz_apt_conditionat`, `aviz_inapt_temporar`, `aviz_inapt` |
| Date | `data`, `data_urm_examinari`, `recomandari` |
| Confidence | `overall_confidence`, `field_confidences` (per-field, 0–100) |

### Sample Medical Image

Location: `medical-images/Screenshot 2026-01-20 at 13.57.50.png`

> 
> python3 scripts/upload_folder.py medical-images/
>

---

## 7. OSSF Criticality Score

The OSSF criticality score was run against the repository using the tool installed via `go install`:

```bash
# Install
go install github.com/ossf/criticality_score/cmd/criticality_score@latest

# Run with default scorer
criticality_score -format text -depsdev-disable https://github.com/miron-boiangiu/ss-project

# Run with original_pike.yml config
criticality_score -format text -depsdev-disable \
  -scoring-config original_pike.yml https://github.com/miron-boiangiu/ss-project
```

**Result: `0.19156`** (default and original_pike scores are identical).

---

## 8. Appendices (optional)
