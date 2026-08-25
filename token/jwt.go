package token

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

var (
	ErrInvalidTokenFormat = errors.New("invalid token format")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenNotValidYet   = errors.New("token is not valid yet")
	ErrInvalidSignature   = errors.New("invalid token signature")
	ErrNilKey             = errors.New("cryptographic key cannot be nil")
)

// StandardClaims is a type alias for standard RFC 7519 registered claims without custom data payload.
type StandardClaims = JWTClaims[any]

// JWTClaims defines standard RFC 7519 registered claims with a generic Data field for type-safe custom payloads.
type JWTClaims[T any] struct {
	Subject   string `json:"sub,omitempty"`  // Subject (e.g., User UUID / Identifier)
	TokenID   string `json:"jti,omitempty"`  // JWT ID (e.g., Session UUID / Token Identifier)
	Issuer    string `json:"iss,omitempty"`  // Service Issuer
	Audience  string `json:"aud,omitempty"`  // Audience
	IssuedAt  int64  `json:"iat,omitempty"`  // Issued At (Unix Timestamp)
	ExpiresAt int64  `json:"exp,omitempty"`  // Expiration Time (Unix Timestamp)
	NotBefore int64  `json:"nbf,omitempty"`  // Not Before (Unix Timestamp)
	Data      T      `json:"data,omitempty"` // Type-safe custom application data / payload
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// GenerateES256JWT creates a signed JWT string using an ECDSA P-256 private key and generic JWTClaims.
func GenerateES256JWT[T any](privKey *ecdsa.PrivateKey, claims JWTClaims[T]) (string, error) {
	if privKey == nil {
		return "", ErrNilKey
	}

	header := jwtHeader{
		Alg: "ES256",
		Typ: "JWT",
	}

	headerJSON, err := sonic.Marshal(header)
	if err != nil {
		return "", err
	}

	claimsJSON, err := sonic.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := headerB64 + "." + claimsB64

	hash := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		return "", err
	}

	// Format signature as IEEE P1363 (r || s, 32 bytes each for P-256)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	sig := make([]byte, 64)
	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):64], sBytes)

	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return signingInput + "." + sigB64, nil
}

// VerifyES256JWT validates the signature, expiration, and format of an ES256 JWT using an ECDSA P-256 public key,
// and unmarshals the payload directly into a type-safe *JWTClaims[T].
func VerifyES256JWT[T any](pubKey *ecdsa.PublicKey, tokenString string) (*JWTClaims[T], error) {
	if pubKey == nil {
		return nil, ErrNilKey
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidTokenFormat
	}

	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(sig) != 64 {
		return nil, ErrInvalidSignature
	}

	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])

	hash := sha256.Sum256([]byte(signingInput))
	if !ecdsa.Verify(pubKey, hash[:], r, s) {
		return nil, ErrInvalidSignature
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidTokenFormat
	}

	var claims JWTClaims[T]
	if err := sonic.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	if claims.ExpiresAt > 0 && now > claims.ExpiresAt {
		return nil, ErrTokenExpired
	}
	if claims.NotBefore > 0 && now < claims.NotBefore {
		return nil, ErrTokenNotValidYet
	}

	return &claims, nil
}

// HashToken computes a SHA-256 hex string for storing Refresh Tokens safely in DB.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
