package cognito

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// TOTP is the 6-digit authenticator-app code (RFC 6238: SHA-1, 30-second steps) for a base32
// secret at time t, as Cognito checks it. Only the QA test users' command uses it (cmd/qa-users):
// to finish their authenticator set-up, and to give testers a code.
func TOTP(secret string, t time.Time) (string, error) {
	s := strings.ToUpper(strings.ReplaceAll(secret, " ", ""))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.TrimRight(s, "="))
	if err != nil {
		return "", fmt.Errorf("totp: the secret isn't base32")
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(t.Unix()/30))
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	code := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", code%1_000_000), nil
}
