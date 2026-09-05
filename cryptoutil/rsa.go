package cryptoutil

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
)

// GenerateRSAKeyPair generates a new RSA private/public key pair with the specified bit size.
func GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, error) {
	if bits < 2048 {
		return nil, errors.New("key size should be at least 2048 bits for security")
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// SignRSA signs the given data using the provided RSA private key (PSS signature scheme with SHA-256).
func SignRSA(privateKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("private key cannot be nil")
	}

	hash := sha256.Sum256(data)

	// Use PSS scheme for better security margins than PKCS1v15
	signature, err := rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, hash[:], nil)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

// VerifyRSA verifies the signature of the given data using the provided RSA public key (PSS signature scheme with SHA-256).
func VerifyRSA(publicKey *rsa.PublicKey, data []byte, signature []byte) error {
	if publicKey == nil {
		return errors.New("public key cannot be nil")
	}

	hash := sha256.Sum256(data)
	return rsa.VerifyPSS(publicKey, crypto.SHA256, hash[:], signature, nil)
}

// EncryptRSA encrypts data using RSA-OAEP with SHA-256.
func EncryptRSA(publicKey *rsa.PublicKey, data []byte) ([]byte, error) {
	if publicKey == nil {
		return nil, errors.New("public key cannot be nil")
	}

	label := []byte("") // No label used
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, data, label)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

// DecryptRSA decrypts data that was encrypted with EncryptRSA.
func DecryptRSA(privateKey *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("private key cannot be nil")
	}

	label := []byte("") // No label used
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, ciphertext, label)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// ExportRSAPublicKeyAsPEM converts an RSA public key to a PEM-encoded string (SPKI format).
func ExportRSAPublicKeyAsPEM(pubKey *rsa.PublicKey) (string, error) {
	return exportPublicKeyToPEM(pubKey)
}

// ExportRSAPrivateKeyAsPEM converts an RSA private key to a PEM-encoded string (PKCS#8 format).
func ExportRSAPrivateKeyAsPEM(privKey *rsa.PrivateKey) (string, error) {
	return exportPrivateKeyToPEM(privKey)
}

// ParseRSAPublicKeyFromPEM parses a PEM-encoded string back into an RSA public key.
func ParseRSAPublicKeyFromPEM(pemStr string) (*rsa.PublicKey, error) {
	pub, err := parsePKIXPublicKeyFromPEM(pemStr)
	if err != nil {
		return nil, err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("parsed key is not an RSA public key")
	}

	return rsaPub, nil
}

// ParseRSAPrivateKeyFromPEM parses a PEM-encoded string back into an RSA private key.
func ParseRSAPrivateKeyFromPEM(pemStr string) (*rsa.PrivateKey, error) {
	priv, err := parsePKCS8PrivateKeyFromPEM(pemStr)
	if err != nil {
		return nil, err
	}

	rsaPriv, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("parsed key is not an RSA private key")
	}

	return rsaPriv, nil
}
