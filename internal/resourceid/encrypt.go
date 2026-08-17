// Package resourceid encrypts the MCP resource URL for the
// X-Encrypted-Resource-Identifier header sent to the public API.
package resourceid

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const aes256KeySize = 32

var (
	ErrInvalidKey       = errors.New("resource encryption key must be a 32-byte AES-256 key")
	ErrEncryptionFailed = errors.New("resource identifier encryption failed")
)

// EncryptResourceIdentifier encrypts plaintext with AES-256-GCM and returns a
// base64-encoded ciphertext with the nonce prepended. This matches the backend
// DecryptResourceIdentifier contract.
func EncryptResourceIdentifier(key []byte, plaintext string) (string, error) {
	if len(key) != aes256KeySize {
		return "", ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", ErrEncryptionFailed
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrEncryptionFailed
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	sealed := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	out := make([]byte, 0, len(nonce)+len(sealed))
	out = append(out, nonce...)
	out = append(out, sealed...)
	return base64.StdEncoding.EncodeToString(out), nil
}
