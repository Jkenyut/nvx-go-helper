package cryptoutil

import (
	"bytes"
	"testing"
)

func TestRSA_GenerateKeyPair(t *testing.T) {
	_, err := GenerateRSAKeyPair(1024)
	if err == nil {
		t.Error("Expected error for key size < 2048, got nil")
	}

	priv, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key pair: %v", err)
	}
	if priv == nil {
		t.Fatal("Private key is nil")
	}
	if err := priv.Validate(); err != nil {
		t.Fatalf("Generated key is invalid: %v", err)
	}
}

func TestRSA_EncryptDecrypt(t *testing.T) {
	priv, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key pair: %v", err)
	}
	pub := &priv.PublicKey

	message := []byte("secret message for RSA testing")

	ciphertext, err := EncryptRSA(pub, message)
	if err != nil {
		t.Fatalf("EncryptRSA failed: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Fatal("Ciphertext is empty")
	}

	if bytes.Equal(message, ciphertext) {
		t.Fatal("Ciphertext matches plaintext")
	}

	plaintext, err := DecryptRSA(priv, ciphertext)
	if err != nil {
		t.Fatalf("DecryptRSA failed: %v", err)
	}

	if !bytes.Equal(message, plaintext) {
		t.Fatalf("Decrypted text does not match original message. Expected: %s, Got: %s", message, plaintext)
	}
}

func TestRSA_SignVerify(t *testing.T) {
	priv, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key pair: %v", err)
	}
	pub := &priv.PublicKey

	message := []byte("data to be signed")

	signature, err := SignRSA(priv, message)
	if err != nil {
		t.Fatalf("SignRSA failed: %v", err)
	}

	if len(signature) == 0 {
		t.Fatal("Signature is empty")
	}

	err = VerifyRSA(pub, message, signature)
	if err != nil {
		t.Fatalf("VerifyRSA failed: %v", err)
	}

	// Test tampering with data
	tamperedMessage := []byte("data to be signed tampered")
	err = VerifyRSA(pub, tamperedMessage, signature)
	if err == nil {
		t.Fatal("VerifyRSA should fail for tampered message")
	}
}

func TestRSA_ExportParsePEM(t *testing.T) {
	priv, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key pair: %v", err)
	}
	pub := &priv.PublicKey

	privPEM, err := ExportRSAPrivateKeyAsPEM(priv)
	if err != nil {
		t.Fatalf("ExportRSAPrivateKeyAsPEM failed: %v", err)
	}
	if privPEM == "" {
		t.Fatal("Private key PEM is empty")
	}

	pubPEM, err := ExportRSAPublicKeyAsPEM(pub)
	if err != nil {
		t.Fatalf("ExportRSAPublicKeyAsPEM failed: %v", err)
	}
	if pubPEM == "" {
		t.Fatal("Public key PEM is empty")
	}

	parsedPriv, err := ParseRSAPrivateKeyFromPEM(privPEM)
	if err != nil {
		t.Fatalf("ParseRSAPrivateKeyFromPEM failed: %v", err)
	}
	if parsedPriv == nil {
		t.Fatal("Parsed private key is nil")
	}

	parsedPub, err := ParseRSAPublicKeyFromPEM(pubPEM)
	if err != nil {
		t.Fatalf("ParseRSAPublicKeyFromPEM failed: %v", err)
	}
	if parsedPub == nil {
		t.Fatal("Parsed public key is nil")
	}
}
