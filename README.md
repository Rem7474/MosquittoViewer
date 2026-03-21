# MosquittoViewer

Real-time dashboard for MQTT broker logs (Mosquitto and any other log file).
Single Go binary embeds a Vue 3 frontend; streams log lines via WebSocket with authentication, filtering, and export.

## Stack

- **Backend**: Go — fsnotify tail, ring buffer, WebSocket hub, JWT RS256
- **Frontend**: Vue 3 + Vite + TypeScript — virtual scroller, live filters
- **Auth**: JWT RS256 (access 15 min / refresh 7 days)
- **Optional persistence**: SQLite (build tag `sqlite`, disabled by default)

## Prerequisites

| Tool | Minimum version |
|------|----------------|
| Go | 1.21+ |
| Node.js | 20.19+ or 22.12+ (Vite requirement) |
| npm | bundled with Node |
| OpenSSL | any modern version |
| GNU Make | any |

On Debian/Ubuntu for production: `sudo apt-get install acl apache2-utils`.

---

## Quick Start (development)

```bash
# 1. Generate JWT key pair (skipped if files already exist)
make gen-keys

# 2. Edit configs/config.yaml – set at least one log source and the admin password hash
#    (see Configuration section below for the hash command)

# 3. Start Vite dev server + Go backend
make dev
```

Open <http://localhost:5173>.

---

## Configuration

All settings live in [`configs/config.yaml`](configs/config.yaml).

### Log sources

Any number of log files can be watched simultaneously.
Each entry under `logs:` is independent (own parser, own buffer).

```yaml
logs:
  - name: "mosquitto"                          # label shown in the UI
    path: "/var/log/mosquitto/mosquitto.log"
    format: "mosquitto_standard"               # or "custom"
    buffer_size: 500                           # in-memory ring buffer (entries)
    read_existing_on_start: true               # load last N lines on startup

  - name: "myapp"
    path: "/var/log/myapp/app.log"
    format: "custom"
    custom_regex: '^(?P<ts>\d{4}-\d{2}-\d{2}T[\d:.Z]+) (?P<level>\w+) (?P<msg>.+)$'
    buffer_size: 300
    read_existing_on_start: false
```

**Format `mosquitto_standard`** — parses native Mosquitto lines:
```
1710000000: New client connected from 192.168.1.10 as my-client (p2, c1, k60).
```
Extracts: Unix timestamp, level (inferred), client ID, topic.

**Format `custom`** — user-defined regex with named groups:

| Group | Description |
|-------|-------------|
| `ts` | timestamp (Unix int or RFC3339) |
| `level` | log level string |
| `msg` | message |
| `client_id` | optional MQTT client id |
| `topic` | optional MQTT topic |
| `plugin` | optional plugin name |

### Other settings

```yaml
server:
  host: "127.0.0.1"
  port: 8080

auth:
  users:
    - username: "admin"
      password_hash: "$2a$12$..."   # bcrypt – see below
  jwt:
    private_key_path: "./configs/jwt_rs256.pem"
    public_key_path:  "./configs/jwt_rs256_pub.pem"
    access_token_ttl:  "15m"
    refresh_token_ttl: "168h"

debug: false
```

### Generating a password hash

```bash
make hash-password
```

Requires `htpasswd` (`apache2-utils` on Debian, `httpd-tools` on RHEL).
Paste the output into `password_hash:` in `config.yaml`.

---

## Architecture

```
Log file A ──► [watcher A / parser] ──► ring buffer A ──► GET /api/logs?source=A
                        │
Log file B ──► [watcher B / parser] ──► ring buffer B ──► GET /api/logs?source=B
                        │                                  GET /api/logs  (all)
                        ▼
                 subscribe channel
                        │
                        ▼
                   [ws hub] ──────────────────────────────► WS /api/ws
                                                            (streams all sources)

Browser
  POST /api/auth/login
  POST /api/auth/refresh
  GET  /api/sources         ← list of configured sources
  GET  /api/logs[?source=]  ← buffered history
  GET  /api/ws?token=…      ← live stream
```

Real-time flow: WebSocket delivers entries as they arrive.
The REST endpoint is called **once on page load** to fill the initial history; there is no polling.

---

## Build

```bash
make build
```

Produces `bin/mosquitto-viewer` (Go binary with embedded frontend).

```bash
make clean   # remove build artefacts
```

---

## Deployment

### 1 — Build and install files

```bash
sudo make install
```

- Builds binary and frontend.
- Copies binary to `/usr/local/bin/mosquitto-viewer`.
- Creates system user/group `mosquitto-viewer` if missing.
- Installs config to `/etc/mosquitto-viewer/config.yaml` with JWT paths rewritten to absolute.

> **Edit `/etc/mosquitto-viewer/config.yaml`** after install to set the correct log paths and password hash before starting the service.

### 2 — Install and enable the systemd unit

```bash
sudo make install-service
```

If you watch log files outside `/var/log/mosquitto`, add the corresponding directories to `ReadOnlyPaths=` in `/etc/systemd/system/mosquitto-viewer.service` before enabling:

```ini
ReadOnlyPaths=/var/log/mosquitto
ReadOnlyPaths=/var/log/myapp
```

### 3 — Grant log file read permissions

The service runs as an unprivileged user (`mosquitto-viewer`).
POSIX ACLs let it read the log files without changing their ownership.

```bash
# Single source (default):
sudo make grant-log-access

# Multiple sources – space-separated list:
sudo make grant-log-access LOG_PATHS="/var/log/mosquitto/mosquitto.log /var/log/myapp/app.log"
```

This sets two ACL entries per directory:
- **Access ACL** on the file itself (if it exists already).
- **Default ACL** on the parent directory — inherited automatically by every new file created there, including files created by log rotation or service restart. This means you do **not** need to re-run this command after `logrotate`.

Verify:
```bash
getfacl /var/log/mosquitto        # look for "default:user:mosquitto-viewer:r--"
getfacl /var/log/mosquitto/mosquitto.log
```

### 4 — Start

```bash
sudo make start
```

### Full first-time deploy (steps 1–4 in one command)

```bash
sudo make deploy LOG_PATHS="/var/log/mosquitto/mosquitto.log"
```

---

## Service management

```bash
make start    # systemctl start mosquitto-viewer
make stop     # systemctl stop  mosquitto-viewer
make restart  # systemctl restart mosquitto-viewer
make status   # systemctl status mosquitto-viewer
make logs     # journalctl -u mosquitto-viewer -f
```

---

## Reverse proxy

Sample configs are in [`deployments/`](deployments/):

| File | Use |
|------|-----|
| `nginx.conf` | Nginx with TLS |
| `nginx-http.conf` | Nginx HTTP-only (behind an upstream TLS proxy) |
| `apache.conf` | Apache with TLS |
| `apache-http.conf` | Apache HTTP-only |

**Nginx (Debian/Ubuntu)**:
```bash
sudo cp deployments/nginx.conf /etc/nginx/sites-available/mosquitto-viewer.conf
sudo ln -sf /etc/nginx/sites-available/mosquitto-viewer.conf \
            /etc/nginx/sites-enabled/mosquitto-viewer.conf
sudo nginx -t && sudo systemctl reload nginx
```

**Apache (Debian/Ubuntu)**:
```bash
sudo cp deployments/apache.conf /etc/apache2/sites-available/mosquitto-viewer.conf
sudo a2enmod ssl headers proxy proxy_http proxy_wstunnel rewrite ratelimit
sudo a2ensite mosquitto-viewer.conf
sudo apache2ctl configtest && sudo systemctl reload apache2
```

**Apache (RHEL/Rocky)**:
```bash
sudo cp deployments/apache.conf /etc/httpd/conf.d/mosquitto-viewer.conf
sudo apachectl configtest && sudo systemctl reload httpd
```

---

## Troubleshooting

**`permission denied` on log file**

```bash
sudo apt-get install -y acl     # Debian/Ubuntu
sudo make grant-log-access LOG_PATHS="<your paths>"
sudo make restart
```

**`status=217/USER` in systemd**

The system user was not created. Re-run the full install:
```bash
sudo make install
sudo make install-service
sudo make restart
```

**Logs not updating in the browser**

Check the WebSocket connection indicator (top-right of the UI). If it shows OFFLINE:
- Verify the service is running: `make status`
- Check browser console for WebSocket errors
- Ensure the reverse proxy forwards `Upgrade: websocket` headers

**Log rotation breaks reading**

Check that the default ACL is present on the directory (not just the file):
```bash
getfacl /var/log/mosquitto | grep default
# Expected: default:user:mosquitto-viewer:r--
```
If missing, re-run `make grant-log-access`.

---

## Security

- **JWT RS256** — asymmetric signatures; private key stays on the server.
- **Short-lived tokens** — access token expires in 15 min; refresh token in 7 days.
- **Auth required** on `/api/logs`, `/api/sources`, and WebSocket upgrade.
- **Unprivileged service user** — no shell, no home directory.
- **systemd hardening** — `NoNewPrivileges`, `ProtectSystem=strict`, `PrivateTmp`, `PrivateDevices`, filesystem paths limited to what is needed.
- **Reverse proxy** — TLS termination and security headers handled externally (see sample configs).

---

## Tests

```bash
go test ./...
```

Covers: log parser, JWT generation/validation, WebSocket hub broadcast/drop behavior.
