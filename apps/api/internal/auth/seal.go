package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// Sealer encrypts patient link tokens (AES-256-GCM) so clinicians can see a link again, while
// a copy of the database alone can't be used to open one: the key (LINK_KEY) is kept outside it.
type Sealer struct {
	aead cipher.AEAD
}

// NewSealer returns a Sealer for a 32-byte key.
func NewSealer(key []byte) (*Sealer, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("link key: %d bytes, want 32", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("link key: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("link key: %w", err)
	}
	return &Sealer{aead: aead}, nil
}

// Seal encrypts token with a random nonce and returns the nonce followed by the ciphertext.
func (s *Sealer) Seal(token string) ([]byte, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("seal: %w", err)
	}
	return s.aead.Seal(nonce, nonce, []byte(token), nil), nil
}

// Open decrypts what Seal returned. It fails for another key or changed bytes.
func (s *Sealer) Open(sealed []byte) (string, error) {
	n := s.aead.NonceSize()
	if len(sealed) < n+s.aead.Overhead() {
		return "", errors.New("open: sealed token too short")
	}
	plain, err := s.aead.Open(nil, sealed[:n], sealed[n:], nil)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	return string(plain), nil
}
