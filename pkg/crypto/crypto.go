// Package crypto provides AES-256-GCM envelope encryption for secrets at rest
// (webhook secrets, provider credentials). The master key is a base64-encoded
// 32-byte value from config.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

type Cipher struct {
	aead cipher.AEAD
}

// New builds a Cipher from a base64-encoded 32-byte key.
func New(base64Key string) (*Cipher, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt returns nonce||ciphertext.
func (c *Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt reverses Encrypt.
func (c *Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	n := c.aead.NonceSize()
	if len(ciphertext) < n {
		return nil, errors.New("ciphertext too short")
	}
	return c.aead.Open(nil, ciphertext[:n], ciphertext[n:], nil)
}
