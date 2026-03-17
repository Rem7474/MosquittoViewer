# MosquittoViewer

MosquittoViewer is a secure real-time dashboard for MQTT broker logs (Mosquitto + custom plugin).
It provides authentication, live streaming over WebSocket, filtering, and export tools from a single Go binary serving a Vue 3 frontend.

## Stack

- Backend: Go
- Frontend: Vue 3 + Vite + TypeScript
- Auth: JWT RS256 (access 15m / refresh 7d)
- Streaming: WebSocket
- Log watcher: fsnotify + tail behavior
- Optional persistence: SQLite (build tag `sqlite`)

## Prerequisites

- Go 1.21+
- Node.js 18+
- npm
- OpenSSL
- GNU Make (or compatible `make`)

## Quick Start

1. Clone and enter project.
2. Generate keys:

```bash
make gen-keys
```

3. Configure [configs/config.yaml](configs/config.yaml) (log path, admin hash, JWT key paths).
4. Start dev mode (backend + frontend):

```bash
make dev
```

5. Open <http://localhost:5173> in development.

## Configuration

Main config is [configs/config.yaml](configs/config.yaml).

- `server.host` / `server.port`: HTTP bind address.
- `log.path`: Mosquitto log file path.
- `log.format`: `mosquitto_standard` or `custom`.
- `log.custom_regex`: named groups for custom parser (`ts`, `level`, `plugin`, `msg`, `client_id`, `topic`).
- `log.buffer_size`: in-memory ring buffer size.
- `auth.users`: local users with bcrypt hash.
- `auth.jwt.*`: RS256 key paths and token TTLs.
- `sqlite.enabled`: optional historical storage.

### Password Hash

```bash
make hash-password
```

### JWT Keys

```bash
openssl genrsa -out configs/jwt_rs256.pem 2048
openssl rsa -in configs/jwt_rs256.pem -pubout -out configs/jwt_rs256_pub.pem
```

## Supported Log Formats

### Standard Mosquitto

```text
1710000000: New connection from 192.168.1.10 on port 1883.
1710000001: New client connected from 192.168.1.10 as my-client (p2, c1, k60).
1710000002: Client my-client disconnected.
```

Parser extracts:
- Unix timestamp
- Level inference (`ERROR`, `WARN`, `DEBUG`, `INFO`)
- Client ID
- Topic when available

### Custom Plugin Regex

Configure `log.format: custom` and set `log.custom_regex`:

```yaml
custom_regex: '^(?P<ts>\d+): \[(?P<level>\w+)\] \[(?P<plugin>\w+)\] (?P<msg>.+)$'
```

Named groups supported: `ts`, `level`, `plugin`, `msg`, `client_id`, `topic`.

## Architecture

```text
Mosquitto log file
      |
      v
[logwatcher/fsnotify] ---> [parser] ---> [ring buffer]
      |                                 |
      |                                 +--> GET /api/logs
      v
   channel ----------------------------> [ws hub] ---> browser clients

Browser
  - POST /api/auth/login
  - POST /api/auth/refresh
  - GET /api/logs
  - GET /api/ws?token=...
```

## Production Build

```bash
make build
```

This builds:
- Frontend into [web](web)
- Backend binary at `bin/mosquitto-viewer`

## Deployment

1. Copy binary to `/usr/local/bin/mosquitto-viewer`.
2. Copy config to `/etc/mosquitto-viewer/config.yaml`.
3. Install systemd unit from [deployments/mosquitto-viewer.service](deployments/mosquitto-viewer.service).
4. Configure Nginx from [deployments/nginx.conf](deployments/nginx.conf).
5. Enable and start service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now mosquitto-viewer
```

## Security Summary

- RS256 signed JWT tokens.
- Short-lived access token + refresh token flow.
- Auth required for logs API and WebSocket.
- Nginx rate limiting for auth/API endpoints.
- TLS termination and security headers.
- systemd hardening options enabled.

## Tests

Run backend tests:

```bash
go test ./...
```

Includes tests for:
- log parser
- JWT generation/validation
- WebSocket hub broadcast/drop behavior
