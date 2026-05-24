# Sequential Summit

IoT image capture platform that receives photos from devices via MQTT, performs OCR (Tesseract), extracts structured data from Romanian medical certificates, and provides a web interface for browsing and searching captured images.

## Secrets

### Environment secrets

Copy the example environment file and fill in your values:

```bash
cp ENV_EXAMPLE .env
```

At minimum, generate a random 32-byte hex key for `ENCRYPTION_KEY` and a strong value for `JWT_SECRET`:

```bash
# Generate ENCRYPTION_KEY (32 bytes, hex-encoded)
openssl rand -hex 32

# Generate JWT_SECRET
openssl rand -hex 32
```

### TLS certificates (mTLS for MQTT)

The MQTT broker and API communicate over TLS with mutual authentication.  
Generate the CA, server, and client certificates:

```bash
./scripts/generate-certs.sh
```

This creates the following files in `./secrets/`:

| Secret file       | Used by   | Mounted in container at       |
|-------------------|-----------|-------------------------------|
| `ca.crt`          | broker, go-api | `/run/secrets/ca.crt`    |
| `server.crt`      | broker    | `/run/secrets/server.crt`     |
| `server.key`      | broker    | `/run/secrets/server.key`     |
| `web.crt`         | go-api    | `/run/secrets/web.crt`        |
| `web.key`         | go-api    | `/run/secrets/web.key`        |
| `python-sender-1.*` / `folder-uploader.*` | client devices | — |

Secrets are defined under the top-level `secrets:` key in `docker-compose.yml` and mounted into containers at `/run/secrets/`. The broker reads them directly from its config (`broker/mosquitto.conf`), while `server/main.go` loads `web.crt`/`web.key` for its MQTT client cert and `ca.crt` to verify the broker.

## Quick Start

1. Generate secrets and configure the environment (see [Secrets](#secrets) above).
2. Start all services:
   ```bash
   docker compose up --build
   ```
3. Start the frontend dev server (in a separate terminal):
   ```bash
   cd client
   yarn install
   yarn dev
   ```

The API is available at `http://localhost:8080` and the frontend at `http://localhost:5173`.

## Services

| Service  | Port | Description                        |
|----------|------|------------------------------------|
| go-api   | 8080 | REST API (Go)                      |
| postgres | 5432 | PostgreSQL database                |
| broker   | 8883 | Mosquitto MQTT broker (mTLS)       |
| client   | 5173 | React/Vite frontend (run locally)  |

## Architecture

```
IoT Devices
    │
    │ MQTT (images, registration, disconnect)
    ▼
┌──────────┐       ┌──────────┐
│  MQTT    │       │ React    │
│  Broker  │       │ Frontend │
└────┬─────┘       └────┬─────┘
     │                   │ HTTP
     ▼                   ▼
┌────────────────────────────┐
│         Go API             │
│  • OCR (Tesseract)         │
│  • Medical cert parsing    │
│  • JWT authentication      │
│  • AES-256-GCM encryption  │
└────────────┬───────────────┘
             │
             ▼
┌──────────────────┐
│   PostgreSQL     │
└──────────────────┘
```

### Backend (server/)

- **domain/** — Interfaces and data models (User, Device, Photo)
- **repository/** — PostgreSQL implementations using pgx
- **routes/** — HTTP handlers with JWT auth middleware
- **broker/** — MQTT message handlers for device communication
- **utils/** — OCR helpers, medical certificate parser, AES encryption

### Frontend (client/src/)

- **pages/** — Login, photos, devices, statistics
- **components/** — Reusable UI components
- **contexts/AuthContext.tsx** — JWT token management

### MQTT Topics

| Topic                          | Direction      | Purpose                  |
|--------------------------------|----------------|--------------------------|
| `ssproject/images/{device_id}` | Device → API   | Image payload            |
| `ssproject/commands`           | API → Device   | CAPTURE, START-LIVE, STOP-LIVE |
| `register/{device_id}`        | Device → API   | Device self-registration |
| `device/id/{device_id}`       | Device → API   | Disconnect signal        |

## Security

- **Passwords** are hashed with bcrypt before storage
- **Personal/medical data** (CNP, names, addresses, phone numbers, medical recommendations) is encrypted with AES-256-GCM at the application level before being written to the database
- **Authentication** uses JWT tokens (24h expiry) with HMAC-SHA256 signing
- **Encryption key** is a 32-byte key provided via the `ENCRYPTION_KEY` environment variable

## Development

### Run tests

```bash
cd server
go test ./... -v
```

### Rebuild after code changes

```bash
docker compose up --build
```

### Database

The schema is automatically applied on first run via `migrations/001_init.sql`. To reset the database, remove the Docker volume:

```bash
docker compose down -v
docker compose up --build
```
