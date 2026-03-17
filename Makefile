.PHONY: dev build install prepare-service-user install-systemd systemd systemd-reload enable-service restart-service service-status deploy clean install-tools gen-keys hash-password

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
SYSCONFDIR ?= /etc/mosquitto-viewer
SYSTEMD_UNITDIR ?= /etc/systemd/system
SERVICE_NAME ?= mosquitto-viewer
SERVICE_USER ?= mosquitto-viewer
SERVICE_GROUP ?= mosquitto-viewer

gen-keys:
	@[ -f configs/jwt_rs256.pem ] || openssl genrsa -out configs/jwt_rs256.pem 2048
	@[ -f configs/jwt_rs256_pub.pem ] || openssl rsa -in configs/jwt_rs256.pem -pubout -out configs/jwt_rs256_pub.pem

dev: gen-keys
	@trap 'kill 0' INT; \
	  (cd frontend && npm run dev) & \
	  go run ./cmd/server --config configs/config.yaml & \
	  wait

build: gen-keys
	@node -v | grep -E 'v(2[0-9]|[3-9][0-9])\.' > /dev/null || (echo "Error: Node.js 20+ required for Vite. Current version: $$(node -v)" && exit 1)
	cd frontend && npm install && npm run build
	go build -ldflags="-s -w" -o bin/mosquitto-viewer ./cmd/server

prepare-service-user:
	@getent group $(SERVICE_GROUP) >/dev/null || groupadd --system $(SERVICE_GROUP)
	@id -u $(SERVICE_USER) >/dev/null 2>&1 || useradd --system --gid $(SERVICE_GROUP) --home-dir /nonexistent --shell /usr/sbin/nologin $(SERVICE_USER)

install: build prepare-service-user
	install -d $(BINDIR)
	install -m 0755 bin/mosquitto-viewer $(BINDIR)/mosquitto-viewer
	install -d -m 0750 -o root -g $(SERVICE_GROUP) $(SYSCONFDIR)
	install -m 0640 -o root -g $(SERVICE_GROUP) configs/config.yaml $(SYSCONFDIR)/config.yaml
	@sed -i 's|private_key_path: .*|private_key_path: "$(SYSCONFDIR)/jwt_rs256.pem"|' $(SYSCONFDIR)/config.yaml
	@sed -i 's|public_key_path: .*|public_key_path: "$(SYSCONFDIR)/jwt_rs256_pub.pem"|' $(SYSCONFDIR)/config.yaml
	install -m 0640 -o root -g $(SERVICE_GROUP) configs/jwt_rs256.pem $(SYSCONFDIR)/jwt_rs256.pem
	install -m 0644 -o root -g $(SERVICE_GROUP) configs/jwt_rs256_pub.pem $(SYSCONFDIR)/jwt_rs256_pub.pem

install-systemd:
	install -d $(SYSTEMD_UNITDIR)
	install -m 0644 deployments/mosquitto-viewer.service $(SYSTEMD_UNITDIR)/$(SERVICE_NAME).service

systemd: prepare-service-user install-systemd systemd-reload enable-service

systemd-reload:
	systemctl daemon-reload

enable-service:
	systemctl enable --now $(SERVICE_NAME)

restart-service:
	systemctl restart $(SERVICE_NAME)

service-status:
	systemctl status $(SERVICE_NAME) --no-pager

deploy: install install-systemd systemd-reload enable-service

clean:
	rm -rf bin/ web/assets web/index.html frontend/dist

hash-password:
	@read -p "Mot de passe: " pwd; \
	  htpasswd -bnBC 12 "" "$$pwd" | tr -d ':\n'; echo
