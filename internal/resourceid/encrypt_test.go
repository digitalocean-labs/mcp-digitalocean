package resourceid

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, aes256KeySize)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

// decryptResourceIdentifier mirrors the backend decrypt for round-trip tests only.
func decryptResourceIdentifier(key []byte, ciphertextB64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func TestDecodeKey(t *testing.T) {
	raw := testKey(t)
	b64 := base64.StdEncoding.EncodeToString(raw)

	got, err := DecodeKey(b64)
	if err != nil {
		t.Fatalf("DecodeKey() error = %v", err)
	}
	if len(got) != aes256KeySize {
		t.Fatalf("DecodeKey() len = %d, want %d", len(got), aes256KeySize)
	}
	for i := range raw {
		if got[i] != raw[i] {
			t.Fatalf("DecodeKey() mismatch at %d", i)
		}
	}
}

func TestDecodeKey_Invalid(t *testing.T) {
	cases := []string{
		"not-base64!!!",
		base64.StdEncoding.EncodeToString([]byte("too-short")),
		base64.StdEncoding.EncodeToString(make([]byte, 16)),
		"",
	}
	for _, c := range cases {
		if _, err := DecodeKey(c); err != ErrInvalidKey {
			t.Fatalf("DecodeKey(%q) error = %v, want %v", c, err, ErrInvalidKey)
		}
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := testKey(t)
	plaintext := "https://droplets.mcp.digitalocean.com/mcp"

	ciphertext, err := EncryptResourceIdentifier(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptResourceIdentifier() error = %v", err)
	}
	if ciphertext == "" {
		t.Fatal("EncryptResourceIdentifier() returned empty ciphertext")
	}
	if ciphertext == plaintext {
		t.Fatal("EncryptResourceIdentifier() returned plaintext unchanged")
	}

	got, err := decryptResourceIdentifier(key, ciphertext)
	if err != nil {
		t.Fatalf("decryptResourceIdentifier() error = %v", err)
	}
	if got != plaintext {
		t.Fatalf("decryptResourceIdentifier() = %q, want %q", got, plaintext)
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	key := testKey(t)
	plaintext := "https://apps.mcp.digitalocean.com/mcp"

	a, err := EncryptResourceIdentifier(key, plaintext)
	if err != nil {
		t.Fatalf("first encrypt: %v", err)
	}
	b, err := EncryptResourceIdentifier(key, plaintext)
	if err != nil {
		t.Fatalf("second encrypt: %v", err)
	}
	if a == b {
		t.Fatal("expected different ciphertexts due to random nonce")
	}
}

func TestEncryptResourceIdentifier_InvalidKey(t *testing.T) {
	if _, err := EncryptResourceIdentifier([]byte("short"), "https://example.com/mcp"); err != ErrEncryptionFailed {
		t.Fatalf("EncryptResourceIdentifier() error = %v, want %v", err, ErrEncryptionFailed)
	}
}
