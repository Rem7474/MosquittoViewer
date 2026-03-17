.PHONY: dev build clean install-tools gen-keys hash-password

gen-keys:
	@[ -f configs/jwt_rs256.pem ] || openssl genrsa -out configs/jwt_rs256.pem 2048
	@[ -f configs/jwt_rs256_pub.pem ] || openssl rsa -in configs/jwt_rs256.pem -pubout -out configs/jwt_rs256_pub.pem

dev: gen-keys
	@trap 'kill 0' INT; \
	  (cd frontend && npm run dev) & \
	  go run ./cmd/server --config configs/config.yaml & \
	  wait

build: gen-keys
	cd frontend && npm install && npm run build
	go build -ldflags="-s -w" -o bin/mosquitto-viewer ./cmd/server

clean:
	rm -rf bin/ web/assets web/index.html frontend/dist

hash-password:
	@read -p "Mot de passe: " pwd; \
	  htpasswd -bnBC 12 "" "$$pwd" | tr -d ':\n'; echo
