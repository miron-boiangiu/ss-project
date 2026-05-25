# Extensible Reporting API

## Overview

A dynamic reporting engine that uses the **Registry Pattern** to map string keys to report-generation functions. Adding a new report requires two lines of code — one for the function and one entry in the registry — with no changes to routing or dispatching logic.

The endpoint is `GET /api/reports?type=<report_name>`, registered in `server/routes/init.go:40` and wrapped in the `withAuth` JWT middleware.

## Architecture

### Dispatcher (`server/routes/reports.go`)

`GenerateReportHandler(db)` reads the `?type=` query parameter, looks up the corresponding `ReportFunc` from the `ReportRegistry` map, forwards any additional query parameters as a `params` map, and writes the result as JSON.

```go
func GenerateReportHandler(db *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        reportType := r.URL.Query().Get("type")
        generateFunc, exists := repository.ReportRegistry[reportType]
        // ... 404 if missing, execute if found
        data, err := generateFunc(r.Context(), db, params)
        json.NewEncoder(w).Encode(data)
    }
}
```

### Registry (`server/repository/reports.go`)

```go
type ReportFunc func(ctx context.Context, db *pgxpool.Pool, params map[string]string) (interface{}, error)

var ReportRegistry = map[string]ReportFunc{
    "expirari":      GenerateExpirariReport,
    "anonimizate":   GenerateAnonimizateReport,
    "performanta":   GeneratePerformantaReport,
    "activitate_1m": GenerateActivitate1LunaReport,
    "statistici":    GenerateStatisticiGeneraleReport,
}
```

### Route Registration (`server/routes/init.go:40`)

```go
mux.Handle("/api/reports", withAuth(http.HandlerFunc(GenerateReportHandler(db))))
```

## Implemented Reports

### Expirations — `?type=expirari`

Fetches records with a populated `data_urm_examinari` where the next examination date is within the next 30 days or in the past. Decrypts `nume` and `prenume` in-memory via `utils.Decrypt()` before returning.

**Query logic:** All rows with `data_urm_examinari != ''` are fetched; date strings are parsed in Go (supports `DD.MM.YYYY` and `YYYY-MM-DD` formats). Rows with a parsed date before `now + 30 days` are included.

**Response shape:**
```json
[
  {
    "id": "uuid",
    "nume": "Popescu",
    "prenume": "Ion",
    "aviz_medical": "APT",
    "data_urm_examinari": "15.06.2026"
  }
]
```

### Anonymized Research Data — `?type=anonimizate`

Extracts a safe dataset excluding all PII/PHI columns. The SQL query selects only `id`, `timestamp`, `aviz_medical`, `control_angajare`, and `control_periodic` — no names, CNP, addresses, or phone numbers.

**Response shape:**
```json
[
  {
    "id": "uuid",
    "timestamp": "2026-05-24T10:30:00Z",
    "aviz_medical": "APT",
    "control_angajare": true,
    "control_periodic": false
  }
]
```

### System Performance — `?type=performanta`

Aggregates total document throughput grouped by `device_id`. Prepared struct includes commented-out `average_latency_ms` and `average_confidence` fields for when the Confidence Thresholding PR is merged.

**Response shape:**
```json
[
  {
    "device_id": "esp-cam-01",
    "total_photos": 142
  }
]
```

### Recent Activity (30 days) — `?type=activitate_1m`

Counts all medical checks from the last 30 days using `WHERE timestamp >= NOW() - INTERVAL '30 days'`, grouped by `aviz_medical`. Empty `aviz_medical` values are labelled `"PENDING_REVIEW"`.

**Response shape:**
```json
{
  "total_last_month": 85,
  "status_breakdown": {
    "APT": 42,
    "APT CONDITIONAT": 18,
    "INAPT TEMPORAR": 10,
    "INAPT": 5,
    "PENDING_REVIEW": 10
  }
}
```

### All-Time Statistics — `?type=statistici`

Uses PostgreSQL's conditional aggregation (`COUNT(*) FILTER (WHERE ...)`) to compute 11 statistical data points in a single `db.QueryRow` round-trip:

- `total_documente` — total photos
- `avize` — breakdown: `apt`, `apt_conditionat`, `inapt_temporar`, `inapt`, `neprocesat`
- `controale` — breakdown: `angajare`, `periodic`, `adaptare`, `reluare`, `supraveghere`, `alte`

**Response shape:**
```json
{
  "total_documente": 500,
  "avize": {
    "apt": 200,
    "apt_conditionat": 100,
    "inapt_temporar": 50,
    "inapt": 30,
    "neprocesat": 120
  },
  "controale": {
    "angajare": 150,
    "periodic": 200,
    "adaptare": 80,
    "reluare": 40,
    "supraveghere": 30,
    "alte": 20
  }
}
```

## Adding a New Report

1. Write a function in `server/repository/reports.go` matching the `ReportFunc` signature.
2. Add one entry to `ReportRegistry`.

```go
func GenerateNewReport(ctx context.Context, db *pgxpool.Pool, params map[string]string) (interface{}, error) {
    // query, scan, return
}

var ReportRegistry = map[string]ReportFunc{
    // ...
    "new_report": GenerateNewReport,
}
```

No routing changes, no switch statements, no handler modifications.

## Security

- All reports require JWT authentication via the `withAuth` middleware.
- The expirations report decrypts sensitive fields in application memory only — ciphertext is never exposed.
- The anonymized report explicitly omits PII/PHI columns from the SQL query.
- Query parameters beyond `type` (e.g. `startDate`, `endDate`) are forwarded as the `params` map for future filtering use.
