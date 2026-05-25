# PostgreSQL Migration & Encryption-at-Rest

## Overview

The persistence layer was migrated from MongoDB to PostgreSQL (via `github.com/jackc/pgx/v5` with `pgxpool.Pool`). This change provides stronger relational guarantees, full SQL querying (including `ILIKE` for text search), and a more standard database for structured medical data. Alongside the migration, 13 sensitive fields now use AES-256-GCM application-level encryption before storage.

The four boolean medical-opinion fields (`aviz_apt`, `aviz_apt_conditionat`, `aviz_inapt_temporar`, `aviz_inapt`) were removed in favor of the single `aviz_medical TEXT` field, since these states are mutually exclusive.

## Infrastructure Changes

The `docker-compose.yml` `mongo-db` service was replaced with:

```yaml
postgres:
  image: postgres:16-alpine
  environment:
    POSTGRES_USER: ${POSTGRES_USER}
    POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    POSTGRES_DB: ${POSTGRES_DB}
  volumes:
    - ./migrations:/docker-entrypoint-initdb.d
  ports:
    - "5432:5432"
```

The schema is auto-initialized at container startup by mounting `migrations/001_init.sql` into the PostgreSQL `docker-entrypoint-initdb.d` directory.

## Database Schema

### `users` table

| Column | Type | Constraints |
|---|---|---|
| `email` | `TEXT` | `PRIMARY KEY` |
| `password` | `TEXT` | `NOT NULL` (bcrypt hash) |
| `role` | `TEXT` | `NOT NULL DEFAULT 'user'` |

### `devices` table

| Column | Type | Constraints |
|---|---|---|
| `id` | `TEXT` | `PRIMARY KEY` (UUID, auto-generated via `uuid_generate_v4()`) |
| `device_id` | `TEXT` | `UNIQUE NOT NULL` |
| `device_name` | `TEXT` | `NOT NULL DEFAULT ''` |
| `device_status` | `TEXT` | `NOT NULL DEFAULT 'active'` |
| `ip_address` | `TEXT` | `NOT NULL DEFAULT ''` |
| `port` | `TEXT` | `NOT NULL DEFAULT ''` |
| `last_seen` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` |

Indexed on `device_id` (`idx_devices_device_id`).

### `photos` table

| Column | Type | Constraints | Encrypted |
|---|---|---|---|
| `id` | `TEXT` | `PRIMARY KEY` (UUID, auto-generated) | No |
| `timestamp` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | No |
| `image_type` | `TEXT` | `NOT NULL DEFAULT ''` | No |
| `device_id` | `TEXT` | `NOT NULL DEFAULT ''` | No |
| `text` | `TEXT` | `NOT NULL DEFAULT ''` | No |
| `unitate_medicala` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `adresa_unitate_medicala` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `telefon_unitate_medicala` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `numar_fisa` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `societate_unitate` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `adresa_angajator` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `telefon_angajator` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `nume` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `prenume` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `cnp` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `profesie_functie` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `loc_de_munca` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `tip_control` | `TEXT` | `NOT NULL DEFAULT ''` | No |
| `control_angajare` | `BOOLEAN` | `NOT NULL DEFAULT FALSE` | No |
| `control_periodic` | `BOOLEAN` | `NOT NULL DEFAULT FALSE` | No |
| `control_adaptare` | `BOOLEAN` | `NOT NULL DEFAULT FALSE` | No |
| `control_reluare` | `BOOLEAN` | `NOT NULL DEFAULT FALSE` | No |
| `control_supraveghere` | `BOOLEAN` | `NOT NULL DEFAULT FALSE` | No |
| `control_alte` | `BOOLEAN` | `NOT NULL DEFAULT FALSE` | No |
| `aviz_medical` | `TEXT` | `NOT NULL DEFAULT ''` | No |
| `recomandari` | `TEXT` | `NOT NULL DEFAULT ''` | Yes |
| `data` | `TEXT` | `NOT NULL DEFAULT ''` | No |
| `data_urm_examinari` | `TEXT` | `NOT NULL DEFAULT ''` | No |
| `needs_review` | `BOOLEAN` | `NOT NULL DEFAULT FALSE` | No |
| `overall_confidence` | `DOUBLE PRECISION` | `NOT NULL DEFAULT 0` | No |
| `field_confidences` | `JSONB` | `NOT NULL DEFAULT '{}'::jsonb` | No |
| `document_type` | `TEXT` | `NOT NULL DEFAULT 'fisa_aptitudine'` | No |
| `schema_version` | `TEXT` | `NOT NULL DEFAULT '1.0.0'` | No |

Indexed on `timestamp DESC` (`idx_photos_timestamp`) and `device_id` (`idx_photos_device_id`).

## Encryption Subsystem

### Algorithm

AES-256-GCM is implemented in `server/utils/crypto.go`. A 32-byte key is decoded from the hex-encoded `ENCRYPTION_KEY` environment variable. Each encryption generates a random 12-byte nonce prepended to the ciphertext; the combined payload is base64-encoded for storage.

```go
func Encrypt(plaintext string) (string, error) {
    // aes.NewCipher(key) → cipher.NewGCM(block)
    // random nonce → aesGCM.Seal(nonce, nonce, plaintext, nil)
    // base64.StdEncoding.EncodeToString(ciphertext)
}
```

```go
func Decrypt(ciphertext string) (string, error) {
    // base64.StdEncoding.DecodeString(data)
    // split nonce / encrypted
    // aesGCM.Open(nil, nonce, encrypted, nil)
}
```

Empty strings are passed through without encryption or decryption.

### Encrypted Fields (13 total)

| Field | Reason |
|---|---|
| `cnp` | National ID number (PII) |
| `nume` / `prenume` | Personal name (PII) |
| `profesie_functie` | Profession/role (professional data) |
| `loc_de_munca` | Workplace (professional data) |
| `numar_fisa` | Medical file number |
| `recomandari` | Medical recommendations |
| `unitate_medicala` | Medical unit name |
| `societate_unitate` | Employer/company name |
| `adresa_unitate_medicala` | Medical unit address |
| `adresa_angajator` | Employer address |
| `telefon_unitate_medicala` | Medical unit phone |
| `telefon_angajator` | Employer phone |

### Fields NOT Encrypted

`text` (OCR output, used for `ILIKE` search), `device_id`, `timestamp`, `image_type` (used for filtering), `aviz_medical` (medical opinion string — one of `"APT"`, `"APT CONDITIONAT"`, `"INAPT TEMPORAR"`, `"INAPT"`), and the control-type boolean checkboxes (low sensitivity, used for queries).

## Breaking Changes: Boolean → String for Medical Opinion

The previous four boolean fields have been removed:

| Removed | Replaced by |
|---|---|
| `aviz_apt` | → |
| `aviz_apt_conditionat` | → `aviz_medical TEXT` |
| `aviz_inapt_temporar` | → (one of `"APT"`, `"APT CONDITIONAT"`, `"INAPT TEMPORAR"`, `"INAPT"`) |
| `aviz_inapt` | → |

The medical parser still uses booleans internally to determine which checkbox was ticked, then derives the string value. The frontend statistics page (`client/src/pages/statisticsPage/index.tsx`) now counts by matching the `aviz_medical` string instead of checking individual booleans.

## Environment Variables

### Added

| Variable | Description |
|---|---|
| `POSTGRES_USER` | PostgreSQL user |
| `POSTGRES_PASSWORD` | PostgreSQL password |
| `POSTGRES_DB` | Database name (e.g. `mqtt_streaming_server`) |
| `ENCRYPTION_KEY` | 32 bytes (64 hex characters), generated via `openssl rand -hex 32` |

### Removed

| Variable |
|---|
| `MONGO_INITDB_ROOT_USERNAME` |
| `MONGO_INITDB_ROOT_PASSWORD` |

## Dependencies

### Added

- `github.com/jackc/pgx/v5` — PostgreSQL driver with connection pooling (`pgxpool.Pool`)
- `github.com/google/uuid` — UUID generation for Go struct `Photo.ID`

### Removed

- `go.mongodb.org/mongo-driver` — MongoDB driver (all `bson` struct tags removed)

## Backend Changes

- **`server/main.go`** — Connects via `pgxpool.New()` instead of `mongo.Connect()`.
- **`server/broker/broker.go`** — Accepts `*pgxpool.Pool`; uses `pgx.ErrNoRows` instead of `mongo.ErrNoDocuments`.
- **`server/routes/*.go`** — All `Init*Routes` functions now accept `*pgxpool.Pool`.
- **`server/domain/*.go`** — All `bson:"..."` struct tags removed; `Photo.ID` changed from `primitive.ObjectID` to `string` (UUID).
- **`server/repository/*.go`** — Fully rewritten to use `pgx` with `pgxpool.Pool`. Encrypted fields are encrypted before `INSERT` and decrypted after `SELECT`.
