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
- Node.js 20+ (Vite requires minimum Node 20.19+ or 22.12+)
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

Install binary and config to system paths:

```bash
sudo make install
```

This builds:
- Frontend into [web](web)
- Backend binary at `bin/mosquitto-viewer`

## Deployment

1. Install binary + config:

```bash
sudo make install
```

This also creates the system user/group `mosquitto-viewer` when missing.
`make install` also rewrites JWT key paths in `/etc/mosquitto-viewer/config.yaml` to absolute paths under `/etc/mosquitto-viewer/`.

2. Install and enable systemd service:

```bash
sudo make install-systemd
sudo make grant-log-access
sudo make systemd-reload
sudo make enable-service
```

Shortcut:

```bash
sudo make systemd
```

If your Mosquitto log path differs, override it:

```bash
sudo make grant-log-access LOG_PATH=/path/to/mosquitto.log
```

3. Configure a reverse proxy:
      - Nginx: [deployments/nginx.conf](deployments/nginx.conf)
      - Nginx HTTP-only (behind upstream reverse proxy): [deployments/nginx-http.conf](deployments/nginx-http.conf)
      - Apache: [deployments/apache.conf](deployments/apache.conf)
      - Apache HTTP-only (behind upstream reverse proxy): [deployments/apache-http.conf](deployments/apache-http.conf)

Nginx (Debian/Ubuntu):

```bash
sudo cp deployments/nginx.conf /etc/nginx/sites-available/mosquitto-viewer.conf
sudo ln -sf /etc/nginx/sites-available/mosquitto-viewer.conf /etc/nginx/sites-enabled/mosquitto-viewer.conf
sudo nginx -t
sudo systemctl reload nginx
```

Nginx HTTP-only (behind upstream reverse proxy, Debian/Ubuntu):

```bash
sudo cp deployments/nginx-http.conf /etc/nginx/sites-available/mosquitto-viewer.conf
sudo ln -sf /etc/nginx/sites-available/mosquitto-viewer.conf /etc/nginx/sites-enabled/mosquitto-viewer.conf
sudo nginx -t
sudo systemctl reload nginx
```

Apache (Debian/Ubuntu):

```bash
sudo cp deployments/apache.conf /etc/apache2/sites-available/mosquitto-viewer.conf
sudo a2enmod ssl headers proxy proxy_http proxy_wstunnel rewrite ratelimit
sudo a2ensite mosquitto-viewer.conf
sudo apache2ctl configtest
sudo systemctl reload apache2
```

Apache HTTP-only (behind upstream reverse proxy, Debian/Ubuntu):

```bash
sudo cp deployments/apache-http.conf /etc/apache2/sites-available/mosquitto-viewer.conf
sudo a2enmod headers proxy proxy_http proxy_wstunnel rewrite ratelimit
sudo a2ensite mosquitto-viewer.conf
sudo apache2ctl configtest
sudo systemctl reload apache2
```

Apache (RHEL/CentOS/Rocky):

```bash
sudo cp deployments/apache.conf /etc/httpd/conf.d/mosquitto-viewer.conf
sudo apachectl configtest
sudo systemctl reload httpd
```

Apache HTTP-only (behind upstream reverse proxy, RHEL/CentOS/Rocky):

```bash
sudo cp deployments/apache-http.conf /etc/httpd/conf.d/mosquitto-viewer.conf
sudo apachectl configtest
sudo systemctl reload httpd
```

4. Optional service commands:

```bash
sudo make restart-service
sudo make service-status
```

If you get `status=217/USER`, run:

```bash
sudo make prepare-service-user
sudo make install
sudo make systemd
sudo make restart-service
sudo make service-status
```

If logs do not appear and service shows `permission denied` on Mosquitto log file:

```bash
sudo apt-get install -y acl    # Debian/Ubuntu
sudo make grant-log-access
sudo make restart-service
```

## Security Summary

- RS256 signed JWT tokens.
- Short-lived access token + refresh token flow.
- Auth required for logs API and WebSocket.
- Reverse proxy hardening (Nginx or Apache), including TLS termination and security headers.
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
