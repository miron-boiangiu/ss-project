# MQTT over mTLS 

## Overview

The MQTT broker (`broker`) requires mutual TLS (mTLS) authentication. Every
client — the Go backend (`go-api`), Python senders in `scripts/`, and any future
MCP agents — must present a certificate signed by the project CA when connecting.

## Certificate Authority & Certificate Generation

A self-contained shell script at [`/scripts/generate-certs.sh`](../scripts/generate-certs.sh)
generates the entire PKI. Running it produces a `secrets/` directory at the
project root with:

| File | Purpose |
|---|---|
| `ca.key` / `ca.crt` | Root CA keypair (self-signed) |
| `server.key` / `server.crt` | Broker server cert (CN=`broker`) |
| `web.key` / `web.crt` | Go backend client cert (CN=`web`) |
| `python-sender-1.key` / `python-sender-1.crt` | Sample device cert (CN=`python-sender-1`) |
| `folder-uploader.key` / `folder-uploader.crt` | Symlink copies of python-sender-1 for `upload_folder.py` |

The CA certificate is self-signed with a 10-year validity (configurable via
`DAYS`). Private keys use 4096-bit RSA. Run:

```bash
./scripts/generate-certs.sh
```

`secrets/` is gitignored (entry in `.gitignore`) and must be generated locally
or provisioned out of band.

## Broker Configuration (`broker/mosquitto.conf`)

The broker listens on **port 8883** with `require_certificate true`. The old
anonymous listener on port 1883 has been removed. The configuration references
Docker secrets paths (`/run/secrets/ca.crt`, `server.crt`, `server.key`).

## Docker Compose Setup (`docker-compose.yml`)

Three secrets are declared at the top level (`ca_crt`, `server_crt`, `server_key`)
and mounted into the `broker` container. Two additional secrets (`web_crt`,
`web_key`) are mounted into the `go-api` container. The broker port mapping
changed from `1883:1883` to `8883:8883`.

## Go Backend (`server/main.go`)

The Go MQTT client connects via `tls://broker:8883` instead of `tcp://broker:1883`.
A `tls.Config` is built from the Docker secrets path `/run/secrets/`:

1. `/run/secrets/ca.crt` is loaded into `RootCAs` to verify the broker's cert.
2. `/run/secrets/web.crt` and `/run/secrets/web.key` are loaded as the client
   certificate pair via `tls.LoadX509KeyPair`.

This `tls.Config` is passed to the MQTT client via `opts.SetTLSConfig(tlsConfig)`.

## Broker Info Endpoint (`server/routes/init.go`)

The `/broker-info` endpoint now reads `MQTT_TLS_PORT` from the environment
(defaulting to `8883`) rather than returning the hardcoded plain-TCP port `1883`.
A corresponding `MQTT_TLS_PORT=8883` entry was added to `ENV_EXAMPLE` and to
the `go-api` service environment in `docker-compose.yml`.

## Python Client Scripts

Both `scripts/send_image.py` and `scripts/upload_folder.py`:

- Connect to port `8883` (was `1883`).
- Load `ca.crt`, the device-specific `.crt` and `.key` from the `secrets/` directory.
- Call `client.tls_set(...)` with `tls_version=ssl.PROTOCOL_TLSv1_2`.

## Client Identity

Each MQTT client identifies itself with:

- `client_id` matching the CN in its certificate (e.g. `"web"`, `"python-sender-1"`).
- A certificate signed by the project CA.

The broker uses the `require_certificate true` directive to enforce that only
clients with a valid CA-signed cert can connect.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `MQTT_TLS_PORT` | `8883` | TLS port the broker listens on |

