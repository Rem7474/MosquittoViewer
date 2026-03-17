package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/example/mosquitto-viewer/internal/config"
)

func TestGenerateAndValidate(t *testing.T) {
	cfg := testJWTConfig(t)

	access, refresh, err := GenerateTokenPair("alice", cfg)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ValidateAccessToken(access, cfg); err != nil {
		t.Fatalf("access token should be valid: %v", err)
	}
	if _, err := ValidateRefreshToken(refresh, cfg); err != nil {
		t.Fatalf("refresh token should be valid: %v", err)
	}
}

func TestExpiredToken(t *testing.T) {
	cfg := testJWTConfig(t)
	cfg.AccessTokenTTL = "-1m"

	access, _, err := GenerateTokenPair("bob", cfg)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ValidateAccessToken(access, cfg); err == nil {
		t.Fatal("expected expired access token to be rejected")
	}
}

func TestWrongType(t *testing.T) {
	cfg := testJWTConfig(t)

	_, refresh, err := GenerateTokenPair("carol", cfg)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ValidateAccessToken(refresh, cfg); err == nil {
		t.Fatal("expected refresh token to be rejected as access token")
	}
}

func testJWTConfig(t *testing.T) config.JWTConfig {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	privPath := filepath.Join(dir, "jwt_rs256.pem")
	pubPath := filepath.Join(dir, "jwt_rs256_pub.pem")

	privBytes := x509.MarshalPKCS1PrivateKey(key)
	if err := os.WriteFile(privPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}), 0o600); err != nil {
		t.Fatal(err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pubPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}), 0o600); err != nil {
		t.Fatal(err)
	}

	return config.JWTConfig{
		PrivateKeyPath:  privPath,
		PublicKeyPath:   pubPath,
		AccessTokenTTL:  "15m",
		RefreshTokenTTL: "168h",
	}
}
