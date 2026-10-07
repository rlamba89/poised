package auth

import (
	"bytes"
	"strings"
	"testing"
)

func TestSealerRoundTrip(t *testing.T) {
	s, err := NewSealer(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := s.Seal("a-patient-link-token")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("a-patient-link-token")) {
		t.Error("the sealed bytes contain the token in plain text")
	}
	got, err := s.Open(sealed)
	if err != nil || got != "a-patient-link-token" {
		t.Errorf("Open = %q, %v; want the token back", got, err)
	}

	again, _ := s.Seal("a-patient-link-token")
	if bytes.Equal(again, sealed) {
		t.Error("sealing twice gave the same bytes: the nonce must be random")
	}
}

func TestSealerRefuses(t *testing.T) {
	if _, err := NewSealer([]byte("too short")); err == nil {
		t.Error("NewSealer accepted a key that isn't 32 bytes")
	}
	a, _ := NewSealer(bytes.Repeat([]byte{1}, 32))
	b, _ := NewSealer(bytes.Repeat([]byte{2}, 32))
	sealed, _ := a.Seal("token")
	if _, err := b.Open(sealed); err == nil {
		t.Error("another key opened the sealed token")
	}
	tampered := append([]byte{}, sealed...)
	tampered[len(tampered)-1] ^= 1
	if _, err := a.Open(tampered); err == nil {
		t.Error("a tampered token opened")
	}
	if _, err := a.Open([]byte("x")); err == nil || !strings.Contains(err.Error(), "too short") {
		t.Errorf("Open of 1 byte: err = %v, want too short", err)
	}
}
