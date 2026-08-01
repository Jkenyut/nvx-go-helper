package cryptoutil

import (
	"testing"
)

func TestGenerateECCKeyPair(t *testing.T) {
	priv, err := GenerateECCKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate ECC key pair: %v", err)
	}
	if priv == nil {
		t.Fatal("Expected private key, got nil")
	}
	if priv.PublicKey.Curve == nil {
		t.Fatal("Expected public key to have a valid curve")
	}
}

func TestSignAndVerifyECC(t *testing.T) {
	priv, err := GenerateECCKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate ECC key pair: %v", err)
	}

	data := []byte("hello world this is a test message for ecc")

	signature, err := SignECC(priv, data)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}
	if signature == nil {
		t.Fatal("Signature is nil")
	}

	valid := VerifyECC(&priv.PublicKey, data, signature)
	if !valid {
		t.Error("Expected signature to be valid, but verification failed")
	}
}

func TestVerifyECC_InvalidData(t *testing.T) {
	priv, err := GenerateECCKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate ECC key pair: %v", err)
	}

	data := []byte("original message")
	signature, err := SignECC(priv, data)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	tamperedData := []byte("tampered message")
	valid := VerifyECC(&priv.PublicKey, tamperedData, signature)
	if valid {
		t.Error("Expected signature to be invalid for tampered data, but verification succeeded")
	}
}

func TestVerifyECC_InvalidKey(t *testing.T) {
	priv1, _ := GenerateECCKeyPair()
	priv2, _ := GenerateECCKeyPair()

	data := []byte("message for key 1")
	signature, _ := SignECC(priv1, data)

	valid := VerifyECC(&priv2.PublicKey, data, signature)
	if valid {
		t.Error("Expected signature to be invalid when verified with a different public key")
	}
}

func TestSignECC_NilKey(t *testing.T) {
	_, err := SignECC(nil, []byte("data"))
	if err == nil {
		t.Error("Expected error when signing with nil private key, got nil")
	}
}

func TestVerifyECC_NilKey(t *testing.T) {
	valid := VerifyECC(nil, []byte("data"), []byte("sig"))
	if valid {
		t.Error("Expected verification to fail with nil public key")
	}
}

func TestEncryptAndDecryptECC(t *testing.T) {
	// Generate receiver's key pair
	recipientPriv, err := GenerateECCKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate recipient key: %v", err)
	}

	data := []byte("secret payload for ECIES encryption")

	// 1. Encrypt using recipient's public key
	ciphertext, err := EncryptECC(&recipientPriv.PublicKey, data)
	if err != nil {
		t.Fatalf("EncryptECC failed: %v", err)
	}
	if len(ciphertext) == 0 {
		t.Fatal("Ciphertext is empty")
	}

	// 2. Decrypt using recipient's private key
	plaintext, err := DecryptECC(recipientPriv, ciphertext)
	if err != nil {
		t.Fatalf("DecryptECC failed: %v", err)
	}

	// 3. Verify
	if string(plaintext) != string(data) {
		t.Errorf("Expected decrypted data %q, got %q", string(data), string(plaintext))
	}
}

func TestDecryptECC_TamperedCiphertext(t *testing.T) {
	recipientPriv, _ := GenerateECCKeyPair()
	data := []byte("secret payload")

	ciphertext, _ := EncryptECC(&recipientPriv.PublicKey, data)

	// Tamper with the ciphertext (flip a byte in the encrypted portion)
	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err := DecryptECC(recipientPriv, ciphertext)
	if err == nil {
		t.Error("Expected DecryptECC to fail on tampered ciphertext, but got nil error")
	}
}

func TestDecryptECC_WrongKey(t *testing.T) {
	recipientPriv, _ := GenerateECCKeyPair()
	wrongPriv, _ := GenerateECCKeyPair()

	data := []byte("secret payload")
	ciphertext, _ := EncryptECC(&recipientPriv.PublicKey, data)

	// Try decrypting with wrong private key
	_, err := DecryptECC(wrongPriv, ciphertext)
	if err == nil {
		t.Error("Expected DecryptECC to fail when using wrong private key, but got nil error")
	}
}

func TestExportAndParsePEM(t *testing.T) {
	// Generate key
	originalPriv, err := GenerateECCKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// 1. Export Private Key
	privPEM, err := ExportPrivateKeyAsPEM(originalPriv)
	if err != nil {
		t.Fatalf("Failed to export private key: %v", err)
	}
	if privPEM == "" {
		t.Fatal("Exported private PEM is empty")
	}

	// 2. Parse Private Key
	parsedPriv, err := ParsePrivateKeyFromPEM(privPEM)
	if err != nil {
		t.Fatalf("Failed to parse private key: %v", err)
	}
	if parsedPriv.D.Cmp(originalPriv.D) != 0 {
		t.Error("Parsed private key does not match original")
	}

	// 3. Export Public Key
	pubPEM, err := ExportPublicKeyAsPEM(&originalPriv.PublicKey)
	if err != nil {
		t.Fatalf("Failed to export public key: %v", err)
	}
	if pubPEM == "" {
		t.Fatal("Exported public PEM is empty")
	}

	// 4. Parse Public Key
	parsedPub, err := ParsePublicKeyFromPEM(pubPEM)
	if err != nil {
		t.Fatalf("Failed to parse public key: %v", err)
	}
	if parsedPub.X.Cmp(originalPriv.PublicKey.X) != 0 || parsedPub.Y.Cmp(originalPriv.PublicKey.Y) != 0 {
		t.Error("Parsed public key does not match original")
	}
}

func TestExportPEM_NilKeys(t *testing.T) {
	_, err := ExportPrivateKeyAsPEM(nil)
	if err == nil {
		t.Error("Expected error when exporting nil private key")
	}

	_, err = ExportPublicKeyAsPEM(nil)
	if err == nil {
		t.Error("Expected error when exporting nil public key")
	}
}

func TestParsePEM_Invalid(t *testing.T) {
	_, err := ParsePrivateKeyFromPEM("invalid pem string")
	if err == nil {
		t.Error("Expected error when parsing invalid private PEM")
	}

	_, err = ParsePublicKeyFromPEM("invalid pem string")
	if err == nil {
		t.Error("Expected error when parsing invalid public PEM")
	}
}
