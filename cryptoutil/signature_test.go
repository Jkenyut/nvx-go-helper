package cryptoutil

import (
	"encoding/hex"
	"testing"
)

func TestSignature(t *testing.T) {
	secret := "mysecretkey"
	data1 := "hello"
	data2 := "world"

	// Test 1: Validity of Hex Output
	sig := Signature(secret, data1, data2)
	if _, err := hex.DecodeString(sig); err != nil {
		t.Errorf("Signature returned invalid hex string: %v", err)
	}

	// Test 2: Deterministic
	sig2 := Signature(secret, data1, data2)
	if sig != sig2 {
		t.Errorf("Signature is not deterministic")
	}

	// Test 3: Secret sensitivity
	sigDifferentSecret := Signature("othersecret", data1, data2)
	if sig == sigDifferentSecret {
		t.Errorf("Signature should change with different secret")
	}

	// Test 4: Data sensitivity
	sigDifferentData := Signature(secret, data1, "world2")
	if sig == sigDifferentData {
		t.Errorf("Signature should change with different data")
	}
}
