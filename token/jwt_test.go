package token

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"testing"
	"time"
)

type AppUserData struct {
	Roles       []string `json:"roles"`
	Email       string   `json:"email"`
	Permissions []string `json:"permissions"`
}

func TestES256JWT_GenericClaims(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	pubKey := &privKey.PublicKey

	// 1. Generate token with concrete generic custom data
	claims := JWTClaims[AppUserData]{
		Subject:   "user-123",
		TokenID:   "session-456",
		Issuer:    "be-iam-svc",
		Audience:  "web-client",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
		Data: AppUserData{
			Roles:       []string{"admin", "user"},
			Email:       "user@example.com",
			Permissions: []string{"read", "write"},
		},
	}

	tokenStr, err := GenerateES256JWT(privKey, claims)
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	// 2. Verify token with compile-time type safety
	verifiedClaims, err := VerifyES256JWT[AppUserData](pubKey, tokenStr)
	if err != nil {
		t.Fatalf("failed to verify valid token: %v", err)
	}

	if verifiedClaims.Subject != "user-123" {
		t.Errorf("expected subject 'user-123', got '%s'", verifiedClaims.Subject)
	}
	if verifiedClaims.TokenID != "session-456" {
		t.Errorf("expected token id 'session-456', got '%s'", verifiedClaims.TokenID)
	}
	if verifiedClaims.Issuer != "be-iam-svc" {
		t.Errorf("expected issuer 'be-iam-svc', got '%s'", verifiedClaims.Issuer)
	}
	if verifiedClaims.Audience != "web-client" {
		t.Errorf("expected audience 'web-client', got '%s'", verifiedClaims.Audience)
	}

	// Direct access to typed fields without extra json unmarshaling
	if len(verifiedClaims.Data.Roles) != 2 || verifiedClaims.Data.Roles[0] != "admin" {
		t.Errorf("expected roles ['admin', 'user'], got %v", verifiedClaims.Data.Roles)
	}
	if verifiedClaims.Data.Email != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got '%s'", verifiedClaims.Data.Email)
	}
	if len(verifiedClaims.Data.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(verifiedClaims.Data.Permissions))
	}

	// 3. Verify expired token
	expiredClaims := JWTClaims[AppUserData]{
		Subject:   "user-123",
		IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	expiredToken, err := GenerateES256JWT(privKey, expiredClaims)
	if err != nil {
		t.Fatalf("failed to generate expired JWT: %v", err)
	}

	_, err = VerifyES256JWT[AppUserData](pubKey, expiredToken)
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}

	// 4. Verify not before token
	futureClaims := JWTClaims[AppUserData]{
		Subject:   "user-123",
		NotBefore: time.Now().Add(1 * time.Hour).Unix(),
	}
	futureToken, err := GenerateES256JWT(privKey, futureClaims)
	if err != nil {
		t.Fatalf("failed to generate future JWT: %v", err)
	}
	_, err = VerifyES256JWT[AppUserData](pubKey, futureToken)
	if !errors.Is(err, ErrTokenNotValidYet) {
		t.Errorf("expected ErrTokenNotValidYet, got %v", err)
	}

	// 5. Verify with wrong key
	wrongPrivKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	_, err = VerifyES256JWT[AppUserData](&wrongPrivKey.PublicKey, tokenStr)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}

	// 6. Nil key safety checks
	_, err = GenerateES256JWT[AppUserData](nil, claims)
	if !errors.Is(err, ErrNilKey) {
		t.Errorf("expected ErrNilKey for nil private key, got %v", err)
	}
	_, err = VerifyES256JWT[AppUserData](nil, tokenStr)
	if !errors.Is(err, ErrNilKey) {
		t.Errorf("expected ErrNilKey for nil public key, got %v", err)
	}
}

func TestES256JWT_StandardClaims(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	pubKey := &privKey.PublicKey

	// Using StandardClaims (alias to JWTClaims[any]) without custom payload
	claims := StandardClaims{
		Subject:   "user-plain-456",
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
	}

	tokenStr, err := GenerateES256JWT(privKey, claims)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	verified, err := VerifyES256JWT[any](pubKey, tokenStr)
	if err != nil {
		t.Fatalf("failed to verify token: %v", err)
	}

	if verified.Subject != "user-plain-456" {
		t.Errorf("expected subject user-plain-456, got %s", verified.Subject)
	}
}

func TestHashToken(t *testing.T) {
	rawToken := "sample-refresh-token-xyz"
	hash1 := HashToken(rawToken)
	hash2 := HashToken(rawToken)

	if hash1 != hash2 {
		t.Errorf("expected deterministic hash output, got %s and %s", hash1, hash2)
	}

	if len(hash1) != 64 { // SHA-256 hex is 64 characters
		t.Errorf("expected 64 character hex string, got length %d", len(hash1))
	}
}
