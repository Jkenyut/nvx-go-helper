package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// GenerateECCKeyPair generates a new ECDSA private/public key pair using the NIST P-256 curve.
func GenerateECCKeyPair() (*ecdsa.PrivateKey, error) {
	// P-256 is the recommended standard curve for most general-purpose ECC implementations.
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// SignECC signs the given data using the provided ECDSA private key.
// It first hashes the data using SHA-256, then signs the hash.
// It returns the ASN.1 encoded signature.
func SignECC(privateKey *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("private key cannot be nil")
	}

	// Hash the data before signing
	hash := sha256.Sum256(data)

	// Sign the hash
	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// VerifyECC verifies the ASN.1 encoded signature of the given data using the provided ECDSA public key.
func VerifyECC(publicKey *ecdsa.PublicKey, data []byte, signature []byte) bool {
	if publicKey == nil {
		return false
	}

	// Hash the data to match what was signed
	hash := sha256.Sum256(data)

	// Verify the signature
	return ecdsa.VerifyASN1(publicKey, hash[:], signature)
}

// EncryptECC encrypts data using a hybrid encryption scheme (ECIES).
// It generates an ephemeral ECC key pair, derives a shared secret via ECDH with the recipient's public key,
// and encrypts the data using AES-256-GCM.
func EncryptECC(recipientPubKey *ecdsa.PublicKey, data []byte) ([]byte, error) {
	if recipientPubKey == nil {
		return nil, errors.New("recipient public key cannot be nil")
	}

	// 1. Convert ECDSA public key to ECDH public key
	ecdhRecipientPub, err := recipientPubKey.ECDH()
	if err != nil {
		return nil, err
	}

	// 2. Generate an ephemeral ECDH key pair on the same curve (P-256)
	ephemeralPriv, err := ecdhRecipientPub.Curve().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	ephemeralPub := ephemeralPriv.PublicKey()

	// 3. Compute shared secret
	sharedSecret, err := ephemeralPriv.ECDH(ecdhRecipientPub)
	if err != nil {
		return nil, err
	}

	// 4. Derive AES-256 key using SHA-256
	aesKey := sha256.Sum256(sharedSecret)

	// 5. Create AES-GCM cipher
	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 6. Generate random nonce
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// 7. Pack result: [EphemeralPubKey (65 bytes for P-256)] + [Nonce (12 bytes)] + [Ciphertext + Tag]
	pubBytes := ephemeralPub.Bytes()
	result := make([]byte, 0, len(pubBytes)+len(nonce)+len(data)+aesgcm.Overhead())
	result = append(result, pubBytes...)
	result = append(result, nonce...)
	result = aesgcm.Seal(result, nonce, data, nil)

	return result, nil
}

// DecryptECC decrypts data that was encrypted with EncryptECC.
func DecryptECC(privKey *ecdsa.PrivateKey, packedData []byte) ([]byte, error) {
	if privKey == nil {
		return nil, errors.New("private key cannot be nil")
	}

	// Convert our private key to ECDH
	ecdhPriv, err := privKey.ECDH()
	if err != nil {
		return nil, err
	}

	// The public key for P-256 is 65 bytes uncompressed
	// Nonce is 12 bytes. Tag is 16 bytes.
	minLen := 65 + 12 + 16
	if len(packedData) < minLen {
		return nil, errors.New("ciphertext too short or malformed")
	}

	// Extract components
	pubBytes := packedData[:65]
	nonce := packedData[65 : 65+12]
	ciphertext := packedData[65+12:]

	// 1. Reconstruct ephemeral public key
	ephemeralPub, err := ecdhPriv.Curve().NewPublicKey(pubBytes)
	if err != nil {
		return nil, errors.New("invalid ephemeral public key in ciphertext")
	}

	// 2. Compute shared secret
	sharedSecret, err := ecdhPriv.ECDH(ephemeralPub)
	if err != nil {
		return nil, err
	}

	// 3. Derive AES-256 key using SHA-256
	aesKey := sha256.Sum256(sharedSecret)

	// 4. Create AES-GCM cipher
	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 5. Decrypt
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed (wrong key or tampered data)")
	}

	return plaintext, nil
}

// ExportPublicKeyAsPEM converts an ECDSA public key to a PEM-encoded string (SPKI format).
// This is highly useful for sending the public key to a frontend application.
func ExportPublicKeyAsPEM(pubKey *ecdsa.PublicKey) (string, error) {
	if pubKey == nil {
		return "", errors.New("public key is nil")
	}

	x509EncodedPub, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return "", err
	}

	pemEncodedPub := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: x509EncodedPub,
	})

	return string(pemEncodedPub), nil
}

// ExportPrivateKeyAsPEM converts an ECDSA private key to a PEM-encoded string (PKCS#8 format).
// Store this securely! Do not send this to the frontend.
func ExportPrivateKeyAsPEM(privKey *ecdsa.PrivateKey) (string, error) {
	if privKey == nil {
		return "", errors.New("private key is nil")
	}

	x509EncodedPriv, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return "", err
	}

	pemEncodedPriv := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509EncodedPriv,
	})

	return string(pemEncodedPriv), nil
}

// ParsePublicKeyFromPEM parses a PEM-encoded string back into an ECDSA public key.
func ParsePublicKeyFromPEM(pemStr string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("parsed key is not an ECDSA public key")
	}

	return ecdsaPub, nil
}

// ParsePrivateKeyFromPEM parses a PEM-encoded string back into an ECDSA private key.
func ParsePrivateKeyFromPEM(pemStr string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the private key")
	}

	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	ecdsaPriv, ok := priv.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("parsed key is not an ECDSA private key")
	}

	return ecdsaPriv, nil
}
