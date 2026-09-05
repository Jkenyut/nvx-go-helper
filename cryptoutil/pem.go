package cryptoutil

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"reflect"
)

func isNilKey(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

// exportPublicKeyToPEM encodes any PKIX-compatible public key to PEM format.
func exportPublicKeyToPEM(pubKey any) (string, error) {
	if isNilKey(pubKey) {
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

// exportPrivateKeyToPEM encodes any PKCS#8-compatible private key to PEM format.
func exportPrivateKeyToPEM(privKey any) (string, error) {
	if isNilKey(privKey) {
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

// parsePKIXPublicKeyFromPEM decodes and parses a PEM block into a PKIX public key.
func parsePKIXPublicKeyFromPEM(pemStr string) (any, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the public key")
	}

	return x509.ParsePKIXPublicKey(block.Bytes)
}

// parsePKCS8PrivateKeyFromPEM decodes and parses a PEM block into a PKCS#8 private key.
func parsePKCS8PrivateKeyFromPEM(pemStr string) (any, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the private key")
	}

	return x509.ParsePKCS8PrivateKey(block.Bytes)
}
