package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

var sam = uuid.MustParse("00000000-0000-4000-8000-0000000000b1")

func TestPatientSession(t *testing.T) {
	token, err := IssuePatient(secret, sam, now)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("works within the idle timeout", func(t *testing.T) {
		got, err := VerifyPatient(secret, token, now.Add(PatientIdle-time.Second))
		if err != nil || got != sam {
			t.Fatalf("got %v, %v; want %v", got, err, sam)
		}
	})

	t.Run("ends after 30 minutes without a request", func(t *testing.T) {
		if PatientIdle != 30*time.Minute {
			t.Fatalf("PatientIdle is %v, want 30 minutes (A-4)", PatientIdle)
		}
		if _, err := VerifyPatient(secret, token, now.Add(PatientIdle+time.Second)); err == nil {
			t.Fatal("VerifyPatient accepted an idle session")
		}
	})

	t.Run("a session renewed by each request stays open", func(t *testing.T) {
		at, tok := now, token
		for range 4 { // four requests 20 minutes apart: 80 minutes in all
			at = at.Add(20 * time.Minute)
			if _, err := VerifyPatient(secret, tok, at); err != nil {
				t.Fatalf("at +%v: %v", at.Sub(now), err)
			}
			if tok, err = IssuePatient(secret, sam, at); err != nil {
				t.Fatal(err)
			}
		}
	})

	t.Run("a patient's session isn't a staff sign-in, and a staff one isn't a patient's", func(t *testing.T) {
		if _, err := Verify(secret, token, now); err == nil {
			t.Error("Verify (staff) accepted a patient session")
		}
		staff, err := Issue(secret, alex, now)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := VerifyPatient(secret, staff, now); err == nil {
			t.Error("VerifyPatient accepted a staff token")
		}
	})

	t.Run("refuses a wrong secret, a tampered token and garbage", func(t *testing.T) {
		for name, tc := range map[string]struct {
			secret []byte
			token  string
		}{
			"wrong secret": {[]byte("other"), token},
			"tampered":     {secret, token[:len(token)-2] + "xx"},
			"garbage":      {secret, "not.a.token"},
			"empty":        {secret, ""},
		} {
			if _, err := VerifyPatient(tc.secret, tc.token, now); err == nil {
				t.Errorf("%s: accepted", name)
			}
		}
	})
}

func TestNewLinkToken(t *testing.T) {
	a, err := NewLinkToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := NewLinkToken()
	if len(a) != 43 || a == b {
		t.Errorf("tokens %q and %q: want two different 43-character tokens (32 random bytes)", a, b)
	}
}

func TestLinkTokenHash(t *testing.T) {
	a, b := HashLinkToken("abc"), HashLinkToken("abc")
	if len(a) != 32 || string(a) != string(b) {
		t.Fatalf("hash is %d bytes, or not stable", len(a))
	}
	if string(HashLinkToken("abd")) == string(a) {
		t.Fatal("two tokens hash the same")
	}
}
