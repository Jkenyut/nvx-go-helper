# Cryptoutil (`/cryptoutil`)

Production-ready cryptographic primitives: Argon2id password hashing, ECIES hybrid encryption, AES-256-GCM symmetric encryption, UUID v4/v7, and cryptographically secure random generation.

## 🔐 ECIES Hybrid Encryption Architecture

`cryptoutil.EncryptECC` and `cryptoutil.DecryptECC` implement the Elliptic Curve Integrated Encryption Scheme (ECIES) using the NIST P-256 curve, ECDH, HKDF-SHA256, and AES-256-GCM authenticated encryption.

### Hybrid Encryption Flow (Flowchart)

```mermaid
flowchart TD
    subgraph Encrypt ["1. Encryption Flow (cryptoutil.EncryptECC)"]
        Plain[Plaintext Data]
        RecvPub[Recipient Public Key]
        GenEphem[Generate Ephemeral P-256 Key Pair]
        ECDH1[ECDH: Ephemeral PrivKey + Recipient PubKey]
        HKDF1[HKDF-SHA256: Derive 32-byte AES Key]
        AESGCM1[AES-256-GCM Seal with 12-byte Nonce]
        Payload([Combined Output: EphemPubKey || Nonce || Ciphertext+Tag])

        GenEphem --> ECDH1
        RecvPub --> ECDH1
        ECDH1 --> HKDF1
        HKDF1 --> AESGCM1
        Plain --> AESGCM1
        GenEphem -. Ephem PubKey .-> Payload
        AESGCM1 -. Nonce + Ciphertext .-> Payload
    end

    subgraph Decrypt ["2. Decryption Flow (cryptoutil.DecryptECC)"]
        InPayload([Encrypted Envelope Payload])
        Split[Parse: EphemPubKey | Nonce | Ciphertext]
        RecvPriv[Recipient Private Key]
        ECDH2[ECDH: Recipient PrivKey + Ephem PubKey]
        HKDF2[HKDF-SHA256: Derive 32-byte AES Key]
        AESGCM2[AES-256-GCM Open & Verify Tag]
        DecryptedPlain[Recovered Plaintext]

        InPayload --> Split
        Split --> ECDH2
        RecvPriv --> ECDH2
        ECDH2 --> HKDF2
        HKDF2 --> AESGCM2
        Split --> AESGCM2
        AESGCM2 --> DecryptedPlain
    end
```

---

## 📖 Quickstart

```go
import "github.com/Jkenyut/nvx-go-helper/cryptoutil"

// 1. Password Hashing (Argon2id)
hash, err := cryptoutil.HashPassword("securePassword123")
valid, err := cryptoutil.VerifyPassword("securePassword123", hash)

// 2. Hybrid Asymmetric Encryption (ECIES)
privKey, pubKey, _ := cryptoutil.GenerateECCKeyPair()
ciphertext, _ := cryptoutil.EncryptECC(pubKey, []byte("sensitive payload"))
plaintext, _ := cryptoutil.DecryptECC(privKey, ciphertext)

// 3. Symmetric Encryption (AES-256-GCM)
enc, _ := cryptoutil.NewAESGCMFromHex("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
sealed, _ := enc.Encrypt(map[string]any{"user_id": 123})
```
