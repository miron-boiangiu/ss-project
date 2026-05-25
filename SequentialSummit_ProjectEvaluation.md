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
                                                ┌────────────────────────────┐
                                                │   ocr-service (sandboxed)  │
                                                │   • distroless, no shell   │
                                                │   • cap_drop ALL, ro FS    │
                                                │   • internal-only network  │
                                                └────────────▲───────────────┘
                                                             │ HTTP /ocr
                                                             │
IoT Devices → MQTT Broker (Mosquitto, mTLS) → Go API (parser, RBAC, encryption)
                                                             │
                                                             ▼
                                                       PostgreSQL ◄── React Frontend
```

OCR runs in a **sibling container** isolated from the API's secrets, the database, and the internet. The API process no longer links against libtesseract (built with `CGO_ENABLED=0`, shipped on distroless/static). A code-execution bug in the OCR engine cannot reach the encryption key, JWT secret, or DB credentials.

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
| OCR Text Extraction | Tesseract OCR (Romanian + English) running in a sandboxed `ocr-service` container | ✅ Implemented |
| OCR Sandboxing | `ocr-service` in distroless container: `cap_drop: ALL`, `no-new-privileges`, read-only rootfs, non-root user, internal-only Docker network (no internet/DB/broker reachability), resource limits, no host port exposure. CI-enforced via `scripts/verify-sandbox.sh` (15 hardening assertions) | ✅ Implemented |
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
| OCR Sandbox Design | OCR isolation, threat model, hardening contract | `docs/ocr-sandbox.md` |
| Frontend README | Frontend-specific setup and usage | `client/README.md` |
| Medical Images Guide | Guide for uploading medical certificate images | `medical-images/README.md` |

### CI/CD Evidence

- **GitHub Actions** workflow defined in `.github/workflows/ci.yml`
- Triggers on push / PR to `main` branch + `workflow_dispatch` (manual)
- Jobs:
  1. Go backend: `go build` (CGO disabled), `go test` with coverage
  2. React frontend: `yarn lint`, `yarn build`
  3. Generate TLS certificates for Docker secrets (`./scripts/generate-certs.sh`)
  4. Build both Docker images via `docker compose build` (go-api + ocr-service)
  5. Boot the full stack (`docker compose up -d --wait`)
  6. Run `./scripts/verify-sandbox.sh` — 15 hardening assertions (distroless, no caps, read-only FS, non-root, internal-only network, no exposed ports, resource limits)
  7. Tear down on completion; dump container logs on failure
- A separate **CodeQL** workflow runs static analysis on every push / PR
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

# 4. Verify sandbox configuration (15 hardening assertions)
./scripts/verify-sandbox.sh

# 5. Send test images
python3 scripts/upload_folder.py medical-images/
```

API: `http://localhost:8080`  
Frontend: `http://localhost:5173`  
MQTT Broker: `localhost:8883` (mTLS)  
OCR Service: not exposed to host — reachable only from `go-api` over the internal `ocr-net`

---

## 4. Security & Compliance

### Threat Modeling & Mitigations

We model three primary attacker positions and trace each through the system to identify trust boundaries and mitigations.

#### Threat 1 — Malicious image content (highest impact)

**Attacker capability**: any party able to publish on `ssproject/images/{device_id}` (a device on the same network as the broker; pre-mTLS, anyone) can submit arbitrary bytes labeled as an image. The image flows through libtesseract, libleptonica, and image decoders (libjpeg/libpng/libtiff), which are dense C/C++ code with a long history of *out-of-bounds write*, *heap overflow*, and *integer overflow* CVEs.

**Trophy if RCE succeeds in the OCR engine**:
- `ENCRYPTION_KEY` (AES-256-GCM master key for PHI)
- `JWT_SECRET` (forges tokens for any role)
- PostgreSQL credentials → exfiltration of every record
- MQTT broker connection → command devices, push malicious payloads back to them
- Public-internet egress → exfiltrate to C2

**Mitigations (layered defense)**:
1. **Process isolation** — OCR moved into a sibling container (`ocr-service/`). The Go API is built with `CGO_ENABLED=0` and contains no libtesseract; an exploit against the OCR engine cannot run inside the API process.
2. **Distroless runtime** — `ocr-service` ships on `gcr.io/distroless/cc-debian12:nonroot`. No shell, no package manager, no busybox utilities — post-exploit tooling is unavailable.
3. **Capability stripping** — `cap_drop: ALL` removes every Linux capability; `no-new-privileges:true` prevents setuid escalation.
4. **Read-only root filesystem** — attacker cannot persist a dropper or modify the binary; `/tmp` is a 64 MiB tmpfs for tesseract's internal use only.
5. **Non-root user** — runs as uid 65532 (`nonroot`), so kernel privilege boundaries remain intact.
6. **Network isolation** — `ocr-service` sits on a Docker network with `internal: true`. No DNS resolution or routing to the public internet, to `postgres`, or to the MQTT broker. The only reachable peer is `go-api`.
7. **No host exposure** — the OCR HTTP port (9090) is *not* mapped to the host; verified explicitly by `verify-sandbox.sh`.
8. **Resource limits** — `mem_limit: 512m`, `pids_limit: 256`, `cpus: 1.5` bound damage from decompression bombs / fork bombs.
9. **Magic-byte input validation** — `image.DecodeConfig` rejects non-image bytes before libtesseract sees them (defense in depth; the sandbox is the real defense).
10. **CI-enforced contract** — `scripts/verify-sandbox.sh` runs in CI on every PR and asserts each of the above; regressions break the build.

#### Threat 2 — Unauthenticated MQTT publisher

**Mitigation**: mTLS on port 8883 with `require_certificate true`. Both broker and clients must present certificates signed by our CA. Without a valid client certificate, the broker refuses the connection. Plain-text 1883 is no longer accepted in production configuration.

#### Threat 3 — Stolen database access / disk compromise

**Mitigation**: AES-256-GCM encryption applied at the application layer to all PHI fields (CNP, names, addresses, phone numbers, medical recommendations, employer details) before being written to PostgreSQL. The encryption key never touches disk — provided via `ENCRYPTION_KEY` environment variable at startup. Passwords are hashed with bcrypt (default cost factor). A database dump alone leaks no PHI.

**Currently implemented security mitigations (summary):**
- **OCR sandboxing**: distroless `ocr-service` container with `cap_drop: ALL`, `no-new-privileges`, read-only rootfs, non-root user, on `internal: true` Docker network with no external/DB/broker reachability and no host port exposure (see Threat 1 above)
- **API hardening**: `go-api` is also distroless (`gcr.io/distroless/static-debian12:nonroot`) with no shell and `CGO_ENABLED=0` — no C dependencies linked in
- **CI-enforced sandbox contract**: `scripts/verify-sandbox.sh` runs in CI; any regression in hardening flags fails the build
- **mTLS for MQTT**: mutual certificate verification (`require_certificate true`), port 8883 only
- **JWT authentication**: 24h token expiry, HMAC-SHA256 signing
- **RBAC**: user (read-only) vs. admin (full access) enforced in route middleware
- **AES-256-GCM encryption** for PHI at rest (CNP, names, addresses, phones, employer fields, recommendations)
- **bcrypt** password hashing
- **CORS** configuration on the HTTP API

### MISRA / CERT Compliance

Static analysis was performed using `go vet`, `staticcheck`, `golangci-lint`, and additional Go analysis tools. The outputs below are from a run prior to the OCR sandboxing refactor; `go vet` was re-verified clean after the refactor. Line numbers in `staticcheck` / `golangci-lint` / `shadow` outputs may have shifted in `broker/broker.go` and `main.go`, which were significantly rewritten.

#### Analysis Summary by Tool

**1. `go vet ./...`** *(re-run after sandboxing refactor)*
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
- Total test functions: **35** unit tests + **1** fuzz target (across 7 test files)
- Test files: `broker_test.go`, `device_test.go`, `photo_test.go`, `user_test.go`, `medical_parser_test.go`, `medical_parser_schema_test.go`, `medical_parser_fuzz_test.go`
- Mock package: Generated mocks at `server/mocks/domain_mock.go`

| Package | Test File | Tests |
|---------|-----------|-------|
| broker | `broker/broker_test.go` | Device registration placeholder (1 test, table-driven, awaiting cases post-refactor) |
| routes | `routes/device_test.go` | Device API endpoints (5 tests) |
| routes | `routes/photo_test.go` | Photo API endpoints (7 tests) |
| routes | `routes/user_test.go` | User auth endpoints (9 tests) |
| utils | `utils/medical_parser_test.go` | OCR parsing logic (11 tests) |
| utils | `utils/medical_parser_schema_test.go` | JSON Schema validation (2 tests) |
| utils | `utils/medical_parser_fuzz_test.go` | Property-based fuzzing for parser invariants (1 fuzz target) |

**Code Coverage (refreshed after the sandboxing refactor):**

```
$ go test $(go list ./... | grep -v -E 'schema|mocks|domain') -coverprofile=coverage.out && go tool cover -func=coverage.out

ok  	mqtt-streaming-server/broker	0.004s	coverage: 0.0% of statements
ok  	mqtt-streaming-server/routes	0.600s	coverage: 40.8% of statements
ok  	mqtt-streaming-server/utils	0.010s	coverage: 32.1% of statements

total:	(statements)			21.4%
```

| Package | Coverage |
|---------|----------|
| `broker` | 0.0% (existing test was a TODO placeholder; broker was refactored to depend on `utils.OCRRunner` interface — needs new mock-based tests) |
| `routes` | 40.8% |
| `utils` | 32.1% |
| `repository` | 0.0% (no tests) |
| **Overall** (excl. schema/mocks/domain) | **21.4%** |

The drop from 26.1% → 21.4% is explained by the broker refactor: the previous broker test exercised gosseract directly; after sandboxing it depends on an `OCRRunner` interface and the existing test was reduced to a table-driven placeholder. Re-establishing broker coverage with mock-based tests is the next testing priority.

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

**Key Dependencies (server/):**
| Dependency | Version | Purpose |
|------------|---------|---------|
| github.com/eclipse/paho.mqtt.golang | v1.5.1 | MQTT client (mTLS) |
| github.com/golang-jwt/jwt/v4 | v4.5.2 | JWT auth |
| github.com/jackc/pgx/v5 | v5.9.2 | PostgreSQL driver |
| github.com/santhosh-tekuri/jsonschema/v6 | v6.0.2 | JSON Schema validation |
| golang.org/x/crypto | v0.45.0 | bcrypt + crypto utilities |
| github.com/aws/aws-sdk-go-v2 | v1.41.5 | AWS S3 integration |

> **Note**: `github.com/otiai10/gosseract/v2` (Tesseract OCR binding) is **no longer a dependency of the API server**. After the sandboxing refactor, OCR runs out-of-process in the `ocr-service` container, and `server/go.mod` is now CGO-free.

**Key Dependencies (ocr-service/):**
| Dependency | Version | Purpose |
|------------|---------|---------|
| github.com/otiai10/gosseract/v2 | v2.4.1 | Tesseract OCR (via libtesseract / leptonica) |

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

## 5. Team Contri-Git contribution statistics gathered via `git shortlog` and `git log --numstat --all --no-merges --use-mailmap`. A `.mailmap` file at the repository root canonicalizes alternate identities (web-only commits, alternate emails) to the names below.

| Team Member | Lines Added | Lines Removed | Number of Commits |
|-------------|------------|---------------|-------------------|
| Victor-Miron Boiangiu | 13,724 | 863 | 10 |
| Andrei Gorneanu | 1,771 | 281 | 23 |
| Teodor Suteu | 2,581 | 427 | 10 |
| Vlad Grigore | 1,190 | 82 | 5 |
| Andrei Seceleanu | 763 | 565 | 9 |

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
