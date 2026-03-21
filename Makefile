# ──────────────────────────────────────────────────────────────────────────────
#  MosquittoViewer – Makefile
#
#  Targets grouped by phase:
#    development   dev, gen-keys
#    build         build, clean
#    install       install, install-service, grant-log-access   (run as root)
#    service       start, stop, restart, status, logs
#    utilities     hash-password
#    meta          deploy
# ──────────────────────────────────────────────────────────────────────────────

.DEFAULT_GOAL := help

# ── Configurable variables ─────────────────────────────────────────────────────
PREFIX          ?= /usr/local
BINDIR          ?= $(PREFIX)/bin
SYSCONFDIR      ?= /etc/mosquitto-viewer
SYSTEMD_UNITDIR ?= /etc/systemd/system
SERVICE_NAME    ?= mosquitto-viewer
SERVICE_USER    ?= mosquitto-viewer
SERVICE_GROUP   ?= mosquitto-viewer

# Space-separated list of log file paths the service must read.
# Must match every 'path:' entry in configs/config.yaml.
# Example: LOG_PATHS="/var/log/mosquitto/mosquitto.log /var/log/myapp/app.log"
LOG_PATHS       ?= /var/log/mosquitto/mosquitto.log


# ── Development ────────────────────────────────────────────────────────────────
.PHONY: dev
dev: gen-keys  ## Start Vite dev server + Go backend (hot-reload)
	@trap 'kill 0' INT; \
	  (cd frontend && npm run dev) & \
	  go run ./cmd/server --config configs/config.yaml & \
	  wait


# ── Build ──────────────────────────────────────────────────────────────────────
.PHONY: build clean

build: gen-keys  ## Compile frontend + Go binary → bin/mosquitto-viewer
	@node -v | grep -qE 'v(2[0-9]|[3-9][0-9])\.' \
	  || (echo "Node.js 20+ required (got $$(node -v))" && exit 1)
	cd frontend && npm ci && npm run build
	go build -ldflags="-s -w" -o bin/mosquitto-viewer ./cmd/server

clean:  ## Remove all build artefacts
	rm -rf bin/ web/assets web/index.html frontend/dist


# ── Keys & passwords ───────────────────────────────────────────────────────────
.PHONY: gen-keys hash-password

gen-keys:  ## Generate RSA-2048 key pair for JWT (skips if already present)
	@[ -f configs/jwt_rs256.pem ] \
	  || openssl genrsa -out configs/jwt_rs256.pem 2048
	@[ -f configs/jwt_rs256_pub.pem ] \
	  || openssl rsa -in configs/jwt_rs256.pem -pubout -out configs/jwt_rs256_pub.pem

hash-password:  ## Interactively hash a bcrypt password (requires apache2-utils / httpd-tools)
	@command -v htpasswd >/dev/null \
	  || (echo "htpasswd not found – install package 'apache2-utils' (Debian) or 'httpd-tools' (RHEL)" && exit 1)
	@read -p "Password: " pwd; \
	  htpasswd -bnBC 12 "" "$$pwd" | tr -d ':\n'; echo


# ── Install (requires root) ────────────────────────────────────────────────────
.PHONY: _ensure-user install install-service grant-log-access

# Internal target: create the system user/group if missing.
_ensure-user:
	@getent group  $(SERVICE_GROUP) >/dev/null \
	  || groupadd --system $(SERVICE_GROUP)
	@id -u $(SERVICE_USER) >/dev/null 2>&1 \
	  || useradd --system \
	       --gid $(SERVICE_GROUP) \
	       --home-dir /nonexistent \
	       --shell /usr/sbin/nologin \
	       $(SERVICE_USER)

install: build _ensure-user  ## Install binary and config files (run as root)
	install -d $(BINDIR)
	install -m 0755 bin/mosquitto-viewer $(BINDIR)/mosquitto-viewer
	install -d -m 0750 -o root -g $(SERVICE_GROUP) $(SYSCONFDIR)
	install -m 0640 -o root -g $(SERVICE_GROUP) configs/config.yaml        $(SYSCONFDIR)/config.yaml
	@sed -i \
	  -e 's|private_key_path: .*|private_key_path: "$(SYSCONFDIR)/jwt_rs256.pem"|' \
	  -e 's|public_key_path:  *.*|public_key_path:  "$(SYSCONFDIR)/jwt_rs256_pub.pem"|' \
	  $(SYSCONFDIR)/config.yaml
	install -m 0640 -o root -g $(SERVICE_GROUP) configs/jwt_rs256.pem      $(SYSCONFDIR)/jwt_rs256.pem
	install -m 0644 -o root -g $(SERVICE_GROUP) configs/jwt_rs256_pub.pem  $(SYSCONFDIR)/jwt_rs256_pub.pem
	@echo "Config installed in $(SYSCONFDIR)/"
	@echo "→ Review $(SYSCONFDIR)/config.yaml and set the correct log paths and password hash."

install-service: _ensure-user  ## Install + enable the systemd unit (run as root)
	install -m 0644 deployments/mosquitto-viewer.service \
	  $(SYSTEMD_UNITDIR)/$(SERVICE_NAME).service
	systemctl daemon-reload
	systemctl enable $(SERVICE_NAME)
	@echo "Systemd unit installed. Edit $(SYSTEMD_UNITDIR)/$(SERVICE_NAME).service"
	@echo "if you need to add ReadOnlyPaths= for extra log directories."

grant-log-access: _ensure-user  ## Grant read ACL on every path in LOG_PATHS (run as root)
	@command -v setfacl >/dev/null \
	  || (echo "setfacl not found – install package 'acl'" && exit 1)
	@for p in $(LOG_PATHS); do \
	    dir=$$(dirname "$$p"); \
	    if [ ! -d "$$dir" ]; then \
	        echo "  SKIP  $$dir  (directory not found – create it first)"; \
	        continue; \
	    fi; \
	    echo "  ACL   $$dir  →  $(SERVICE_USER):x  (traverse directory)"; \
	    setfacl -m  u:$(SERVICE_USER):x  "$$dir"; \
	    echo "  ACL   $$dir  →  $(SERVICE_USER):r  (default – inherited by new files on rotation)"; \
	    setfacl -d -m u:$(SERVICE_USER):r "$$dir"; \
	    if [ -f "$$p" ]; then \
	        echo "  ACL   $$p  →  $(SERVICE_USER):r  (existing file)"; \
	        setfacl -m u:$(SERVICE_USER):r "$$p"; \
	    else \
	        echo "  INFO  $$p  not yet created – default ACL on $$dir will cover it"; \
	    fi; \
	done
	@echo ""
	@echo "Done. Verify with:"
	@echo "  getfacl <directory>   (check default ACL line)"
	@echo "  getfacl <file>        (check file ACL)"


# ── Service management ─────────────────────────────────────────────────────────
.PHONY: start stop restart status logs

start:    ## Start the service
	systemctl start $(SERVICE_NAME)

stop:     ## Stop the service
	systemctl stop $(SERVICE_NAME)

restart:  ## Restart the service
	systemctl restart $(SERVICE_NAME)

status:   ## Show service status
	systemctl status $(SERVICE_NAME) --no-pager

logs:     ## Tail the service journal
	journalctl -u $(SERVICE_NAME) -f --no-pager


# ── Full first-time deploy (run as root) ───────────────────────────────────────
.PHONY: deploy

deploy: install install-service grant-log-access start  ## Full first-time deploy (run as root)
	@echo ""
	@echo "✓  MosquittoViewer deployed and running."
	@echo "   Binary  : $(BINDIR)/mosquitto-viewer"
	@echo "   Config  : $(SYSCONFDIR)/config.yaml"
	@echo "   Logs    : journalctl -u $(SERVICE_NAME) -f"
	@echo ""
	@echo "   To add more log sources, update:"
	@echo "     1. $(SYSCONFDIR)/config.yaml   (logs: section)"
	@echo "     2. LOG_PATHS in this Makefile (or pass on command line)"
	@echo "     3. ReadOnlyPaths= in the systemd unit"
	@echo "   Then run: make grant-log-access restart"


# ── Help ───────────────────────────────────────────────────────────────────────
.PHONY: help

help:  ## Show available targets
	@echo "Usage: make [target] [VAR=value ...]"
	@echo ""
	@echo "Key variables:"
	@echo "  LOG_PATHS    Space-separated log file paths (default: $(LOG_PATHS))"
	@echo "  SERVICE_USER System user for the service    (default: $(SERVICE_USER))"
	@echo "  SYSCONFDIR   Config directory               (default: $(SYSCONFDIR))"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z][a-zA-Z0-9_-]*:.*##' $(MAKEFILE_LIST) \
	  | grep -v '^_' \
	  | awk 'BEGIN{FS=":.*##"}{printf "  %-20s %s\n", $$1, $$2}'
