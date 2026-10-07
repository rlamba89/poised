package auth

import (
	"bytes"
	"crypto/sha256"
	"regexp"
	"testing"
)

func TestNewSessionID(t *testing.T) {
	id, hash, err := NewSessionID()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`).MatchString(id) {
		t.Errorf("id = %q, want 43 base64url characters (32 random bytes)", id)
	}
	want := sha256.Sum256([]byte(id))
	if !bytes.Equal(hash, want[:]) {
		t.Errorf("hash = %x, want sha256 of the id", hash)
	}
	if !bytes.Equal(hash, HashToken(id)) {
		t.Error("NewSessionID and HashToken disagree")
	}

	other, _, err := NewSessionID()
	if err != nil {
		t.Fatal(err)
	}
	if other == id {
		t.Error("two session ids are the same")
	}
}
