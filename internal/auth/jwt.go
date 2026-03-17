package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/example/mosquitto-viewer/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Type string `json:"type"`
	jwt.RegisteredClaims
}

func GenerateTokenPair(username string, cfg config.JWTConfig) (access, refresh string, err error) {
	priv, err := loadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return "", "", err
	}
	accessTTL, refreshTTL, err := parseTTLs(cfg)
	if err != nil {
		return "", "", err
	}

	now := time.Now()
	accessClaims := Claims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTTL)),
		},
	}
	refreshClaims := Claims{
		Type: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTTL)),
			ID:        randomJTI(),
		},
	}

	access, err = jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(priv)
	if err != nil {
		return "", "", err
	}
	refresh, err = jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims).SignedString(priv)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func ValidateAccessToken(tokenStr string, cfg config.JWTConfig) (string, error) {
	return validateWithType(tokenStr, cfg, "access")
}

func ValidateRefreshToken(tokenStr string, cfg config.JWTConfig) (string, error) {
	return validateWithType(tokenStr, cfg, "refresh")
}

func validateWithType(tokenStr string, cfg config.JWTConfig, expectedType string) (string, error) {
	pub, err := loadPublicKey(cfg.PublicKeyPath)
	if err != nil {
		return "", err
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, errors.New("unexpected signing method")
		}
		return pub, nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}
	if claims.Type != expectedType {
		return "", errors.New("invalid token type")
	}
	if claims.Subject == "" {
		return "", errors.New("missing subject")
	}
	return claims.Subject, nil
}

func parseTTLs(cfg config.JWTConfig) (time.Duration, time.Duration, error) {
	accessTTL, err := time.ParseDuration(cfg.AccessTokenTTL)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid access token ttl: %w", err)
	}
	refreshTTL, err := time.ParseDuration(cfg.RefreshTokenTTL)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid refresh token ttl: %w", err)
	}
	return accessTTL, refreshTTL, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("failed to decode private key PEM")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return key, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("failed to decode public key PEM")
	}
	k, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := k.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}
	return key, nil
}

func randomJTI() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("jti-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
