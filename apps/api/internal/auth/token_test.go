package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	secret = []byte("test-secret")
	alex   = User{ID: uuid.MustParse("00000000-0000-4000-8000-0000000000a1"), Name: "Alex Author"}
	now    = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
)

func TestIssueThenVerify(t *testing.T) {
	token, err := Issue(secret, alex, now)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Verify(secret, token, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got != alex {
		t.Errorf("got %+v, want %+v", got, alex)
	}
}

func TestVerifyRejects(t *testing.T) {
	good, err := Issue(secret, alex, now)
	if err != nil {
		t.Fatal(err)
	}
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims{
		Name:             alex.Name,
		RegisteredClaims: jwt.RegisteredClaims{Subject: alex.ID.String(), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))},
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	hs512, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: alex.ID.String(), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))},
	}).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	noExpiry, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: alex.ID.String()},
	}).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	badSubject, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "not-a-uuid", ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))},
	}).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		secret []byte
		token  string
		at     time.Time
	}{
		{"expired", secret, good, now.Add(TTL + time.Second)},
		{"wrong secret", []byte("other"), good, now},
		{"tampered", secret, good[:len(good)-2] + "xx", now},
		{"alg none", secret, unsigned, now},
		{"other algorithm", secret, hs512, now},
		{"no expiry", secret, noExpiry, now},
		{"subject not a user id", secret, badSubject, now},
		{"garbage", secret, "not.a.token", now},
		{"empty", secret, "", now},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Verify(tt.secret, tt.token, tt.at); err == nil {
				t.Error("Verify accepted the token")
			}
		})
	}
}
