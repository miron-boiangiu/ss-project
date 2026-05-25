# OCR Sandbox — Hardening Design

## 1. Overview

The OCR pipeline is the platform's largest attack surface: it accepts arbitrary
image bytes, decodes them in libjpeg/libpng, and processes text in libtesseract
(a C++ codebase with a long history of parser CVEs). A crafted image could
achieve remote code execution inside the OCR process.

The sandbox ensures that even if an attacker achieves RCE through the OCR
engine, the compromised process:

- holds no secrets (no DB credentials, no encryption keys, no JWT secrets)
- has no capabilities, no shell, no package manager
- runs on a filesystem that is read-only (except a tiny `tmpfs /tmp`)
- is on a Docker network with **no egress** — cannot reach the internet, the
  database, or the MQTT broker
- has hard resource limits that prevent DoS via decoder bombs

The sandbox is verified programmatically by
[`scripts/verify-sandbox.sh`](../scripts/verify-sandbox.sh) — 16 checks that
must all pass in CI before a deployment is accepted.

## 2. Architecture

```
                    internet  (blocked for ocr-net)
                        |
                   [docker host]
                        |
                   [backend net] -- internal only
    ┌───────────────┬──┴──┬────────────────┐
    │               │     │                │
 postgres        broker  go-api ──── [ocr-net: internal=true]
    │               │     │                │
    └───────────────┴─────┘          ocr-service
                                      (distroless, no caps,
                                       read-only FS, non-root,
                                       resource-limited)
```

Two Docker networks:

- **`backend`** — carries traffic among `go-api`, `postgres`, and `broker`.
- **`ocr-net`** — marked `internal: true` in `docker-compose.yml:100`. Only
  `go-api` and `ocr-service` are attached to it. No egress to the internet,
  no route to `postgres` or `broker`.

`go-api` sits on both networks; `ocr-service` sits only on `ocr-net`.

## 3. Communication

The OCR engine lives in a **separate microservice** (`ocr-service/`) rather
than being linked as a CGo library into `go-api`. This is the key architectural
change that enables the sandbox.

### 3.1 Interface

Defined in `server/utils/ocr_types.go:16`:

```go
type OCRRunner interface {
    Extract(ctx context.Context, imageData []byte) (string, []WordBox, error)
}
```

### 3.2 HTTP Client

`HTTPOCRClient` (`server/utils/ocr_client.go`) implements `OCRRunner` by
sending raw image bytes via `POST /ocr` to the sandboxed service. Configured
via environment variables:

| Variable | Default | Description |
|---|---|---|
| `OCR_SERVICE_URL` | `http://ocr-service:9090` | Base URL of the sandboxed OCR service |
| `OCR_TIMEOUT_SECONDS` | 90 | Per-request HTTP timeout |

The broker (`server/broker/broker.go:87`) calls `ocr.Extract(ctx, body)` with
a `context.WithTimeout` derived from the configured timeout.

### 3.3 OCR Service

`ocr-service/` (`ocr-service/main.go`) is a minimal Go HTTP server that:

- Accepts raw image bytes on `POST /ocr`
- Validates that input is a decodable JPEG/PNG via `image.DecodeConfig`
  (defense in depth — the sandbox is the real protection)
- Runs Tesseract via `gosseract/v2` behind a `sync.Mutex` (gosseract is not
  thread-safe)
- Enforces a payload size limit via `http.MaxBytesReader` (default 25 MiB)
- Exposes `GET /healthz` returning `"ok"` for health checks

**No secrets, no database, no outbound network.** The binary lives in a
distroless container with only libtesseract shared libraries and trained data.

## 4. Container Hardening

All constraints are declared in `docker-compose.yml:38–56`.

### 4.1 Distroless Base Image

Both `ocr-service` and `go-api` run on
`gcr.io/distroless/cc-debian12:nonroot` / `gcr.io/distroless/static-debian12:nonroot`.
These images contain:

- **no shell** (`/bin/sh`, `/bin/bash`, `/bin/dash`)
- **no package manager** (`apt`, `dpkg`, `apk`)
- **no utilities** (`curl`, `wget`, `nc`, `vi`, `less`)

If an attacker gets code execution, they cannot spawn a reverse shell, install
tools, or explore the filesystem interactively.

### 4.2 Linux Capabilities

```
cap_drop: ALL
```

Every Linux capability is dropped. The process cannot:

- `CAP_NET_RAW` — cannot craft raw packets or use ping
- `CAP_NET_ADMIN` — cannot change network configuration
- `CAP_SYS_ADMIN` — cannot mount, `ptrace`, or access kernel namespaces
- `CAP_SETUID` / `CAP_SETGID` — cannot change user identity

### 4.3 No New Privileges

```
security_opt:
  - no-new-privileges:true
```

Even if the process somehow execs a binary with setuid bits, the kernel will
ignore them. Privilege escalation via `suid` binaries is blocked.

### 4.4 Read-Only Root Filesystem

```
read_only: true
tmpfs:
  - /tmp:size=64m,mode=1777
```

The entire container filesystem is read-only except for a 64 MiB `tmpfs` at
`/tmp` (which Tesseract may need for temporary scratch files). An attacker
cannot write persistent payloads or modify binaries.

### 4.5 Non-Root User

Runs as uid 65532 (`nonroot:nonroot` in the distroless image). The `go-api`
container additionally maps the host user via `user: "${UID}:${GID}"`.

### 4.6 Resource Limits

| Resource | Limit | Rationale |
|---|---|---|
| Memory | 512 MiB | Prevents OOM exhaustion via decoder bombs |
| Swap | 512 MiB | No swap beyond memory limit |
| PIDs | 256 | Fork bomb protection |
| CPUs | 1.5 | Prevents CPU-starving sibling containers |

These limits are verified in CI checks 6–8 of `verify-sandbox.sh`.

## 5. Network Isolation

### 5.1 Internal Network

`ocr-net` is declared as `internal: true` (`docker-compose.yml:100`). Docker
blocks all egress to networks other than those explicitly attached. The
container cannot reach:

- The public internet
- The Docker host's external interfaces
- Other Docker networks (`backend`)

### 5.2 Single-Network Attachment

`ocr-service` is attached **only** to `ocr-net` (`docker-compose.yml:54`). It
is not on `backend`, so it cannot resolve or connect to:

- `postgres` (port 5432)
- `broker` (port 1883/8883)

### 5.3 No Published Ports

`ocr-service` has no `ports:` block in `docker-compose.yml`. Port 9090 is
reachable **only** from other containers on `ocr-net` (i.e. `go-api`). The host
cannot reach it directly.

### 5.4 Verified in CI

Checks 9–15 of `verify-sandbox.sh` confirm:

1. `ocr-net` is `internal: true`
2. `ocr-service` is on `ocr-net` only
3. Port 9090 is not exposed on the host (`curl localhost:9090` fails)
4. No internet egress from `ocr-net` (`wget https://1.1.1.1/` fails)
5. Cannot reach `postgres:5432` from `ocr-net`
6. Cannot reach `broker:1883` from `ocr-net`
7. Sanity: `ocr-service:9090/healthz` **is** reachable from inside `ocr-net`

## 6. Threat Model

### 6.1 Attack Path

```
Image malitioasă → MQTT → go-api → HTTP → ocr-service
                                       │
                                       ├─ libjpeg / libpng (decoder)
                                       ├─ leptonica (image processing)
                                       └─ libtesseract (OCR engine)
```

The attacker controls the image bytes. Any of the three libraries above could
contain a parser vulnerability that yields code execution.

### 6.2 What the Attacker Gains

In a worst-case scenario where the attacker achieves full RCE inside the
`ocr-service` container:

| Asset | Reachable? | Why |
|---|---|---|
| `ENCRYPTION_KEY` | No | Not present in container |
| `JWT_SECRET` | No | Not present in container |
| Postgres credentials | No | Not present in container; `ocr-net` cannot route to `backend` |
| MQTT broker | No | `ocr-net` cannot route to `backend` |
| Internet egress | No | `ocr-net` is `internal: true` |
| Other containers | No | Read-only FS, no capabilities, no shell |
| `go-api` secrets | No | But **can** send HTTP responses back to `go-api` on port 9090 (the OCR endpoint) |

### 6.3 What the Attacker Can Do

- Abuse the OC**R endpoint itself** (e.g. return forged OCR results to poison
  extracted medical data). This is an accepted risk — the confidence model and
  review-queue design (`ShouldReview` at `server/utils/ocr_types.go:64`) are
  the compensating control: forged or garbage text yields low-confidence fields
  that are flagged for human review.
- Consume CPU/memory within the configured resource limits.

### 6.4 Out of Scope (Intentionally)

- **gVisor / runsc** — distroless + `cap_drop: ALL` + `internal: true` network
  satisfies the requirement. A gVisor runtime sandbox would add defense in
  depth but also a host-level dependency that the current deployment target
  does not have.
- **mTLS between go-api and ocr-service** — `ocr-net` is already internal-only.
  The threat model does not consider an attacker who already has code execution
  on the Docker host or on `go-api`; at that point the sandbox is moot.
- **Deep image validation before OCR** — currently only a magic-byte check via
  `image.DecodeConfig`. The sandbox is the real defense.

## 7. Verification

Run the verification script after starting the stack:

```bash
docker compose up -d --wait
./scripts/verify-sandbox.sh   # must exit 0
```

The script performs 16 checks in four categories:

| # | Category | Check |
|---|---|---|
| 1 | Container surface | No shell in ocr-service (distroless) |
| 2 | Container surface | `cap_drop: ALL` |
| 3 | Container surface | `no-new-privileges:true` |
| 4 | Container surface | Read-only root filesystem |
| 5 | Container surface | Non-root user |
| 6 | Resource limits | Memory limit > 0 |
| 7 | Resource limits | PIDs limit > 0 |
| 8 | Resource limits | CPU limit > 0 |
| 9 | Network isolation | `ocr-net` is `internal: true` |
| 10 | Network isolation | ocr-service is only on `ocr-net` |
| 11 | Network isolation | Port 9090 not exposed on host |
| 12 | Network isolation | No internet egress from `ocr-net` |
| 13 | Network isolation | Cannot reach postgres from `ocr-net` |
| 14 | Network isolation | Cannot reach broker from `ocr-net` |
| 15 | Network isolation | ocr-service **is** reachable from `ocr-net` (sanity) |
| 16 | API surface | `go-api` has no shell (distroless) |

If any check fails, the script exits 1 with the failing check name(s).

## 8. CI Integration

The verification script runs automatically in CI after `docker compose up -d`.
A failing sandbox check blocks the pipeline — the same way a failing unit test
blocks a merge.

## 9. Migration from In-Process OCR

| Aspect | Before | After |
|---|---|---|
| OCR engine | Linked as CGo (`gosseract`) into `go-api` | Separate service (`ocr-service/`) |
| `go-api` build | `CGO_ENABLED=1`, Alpine | `CGO_ENABLED=0`, distroless/static |
| Network | Single `backend` network | `backend` + `ocr-net` (internal) |
| Secrets in OCR process | All (`ENCRYPTION_KEY`, `JWT_SECRET`, DB creds) | None |
| Shell in containers | Alpine (busybox + ash) | Distroless (no shell) |
| Capabilities | Default | `cap_drop: ALL` |
| Resource limits | None | 512 MiB mem, 256 PIDs, 1.5 CPUs |
| Root filesystem | Read-write | Read-only (+ 64 MiB tmpfs) |

## 10. Key Files

| Purpose | Path |
|---|---|
| OCR service binary | `ocr-service/main.go` |
| OCR service Dockerfile | `ocr-service/Dockerfile` |
| Docker Compose | `docker-compose.yml` |
| `go-api` Dockerfile | `server/Dockerfile` |
| OCR runner interface | `server/utils/ocr_types.go` |
| HTTP OCR client | `server/utils/ocr_client.go` |
| Broker OCR integration | `server/broker/broker.go` |
| Verification script | `scripts/verify-sandbox.sh` |
| Image sender (test) | `scripts/send_image.py` |
