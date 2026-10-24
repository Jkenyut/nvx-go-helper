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

	// Test 5: Delimiter Collision Resistance (length prefix protection)
	sigCollisionA := Signature(secret, "user", "admin")
	sigCollisionB := Signature(secret, "userad", "min")
	if sigCollisionA == sigCollisionB {
		t.Errorf("Canonicalization collision detected: signature parts must not collide")
	}

	// Test 6: VerifySignature Constant-Time Verification
	if !VerifySignature(secret, sig, data1, data2) {
		t.Errorf("VerifySignature should succeed for valid signature")
	}
	if VerifySignature(secret, "invalidhexsig", data1, data2) {
		t.Errorf("VerifySignature should fail for invalid signature")
	}
	if VerifySignature("wrongsecret", sig, data1, data2) {
		t.Errorf("VerifySignature should fail for wrong secret")
	}
}
