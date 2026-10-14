package cognito_test

import (
	"encoding/base32"
	"testing"
	"time"

	"github.com/rlamba89/poised/apps/api/internal/cognito"
)

// RFC 6238 appendix B, SHA-1, cut to the 6 digits authenticator apps (and Cognito) use.
func TestTOTPMatchesTheRFC(t *testing.T) {
	secret := base32.StdEncoding.EncodeToString([]byte("12345678901234567890"))
	cases := map[int64]string{59: "287082", 1111111109: "081804", 1111111111: "050471", 1234567890: "005924", 2000000000: "279037"}
	for unix, want := range cases {
		got, err := cognito.TOTP(secret, time.Unix(unix, 0))
		if err != nil || got != want {
			t.Errorf("at %d: %q, %v; want %s", unix, got, err, want)
		}
	}
}

func TestTOTPTakesCognitosSecretAsGiven(t *testing.T) {
	// Cognito's secret is base32 without padding; apps accept lower case and spaces too.
	if _, err := cognito.TOTP("gezdgnbvgy3tqojq gezdgnbvgy3tqojq", time.Unix(59, 0)); err != nil {
		t.Error(err)
	}
	if _, err := cognito.TOTP("not base32!", time.Unix(59, 0)); err == nil {
		t.Error("accepted a bad secret")
	}
}
