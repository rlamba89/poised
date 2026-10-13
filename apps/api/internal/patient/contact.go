package patient

import (
	"strings"
	"unicode/utf8"
)

// NotThisCountry refuses a mobile outside the deployment's country (PAT-11): texts go only there.
const NotThisCountry = "We can only send text messages to mobile numbers in this country. Please use your email address instead."

// Contact reads what a patient typed to be reached at (S15: asking for a sign-in link, or
// changing a contact): an email address (lower case) or a mobile (E.164; without its country code
// it's one of country, the deployment's PHONE_COUNTRY). email says which. msg is the refusal,
// "" when it's fine: a mobile must be one a text may be sent to (SMSAllowed).
func Contact(s, country string) (value string, email bool, msg string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false, "Enter your mobile number or email address."
	}
	if strings.Contains(s, "@") {
		v, ok := Email(s)
		if !ok {
			return "", true, "Enter a real email address, for example name@example.com."
		}
		return v, true, ""
	}
	v, ok := Mobile(s, country)
	if !ok || v == "" {
		return "", false, "Enter a real mobile number or email address."
	}
	if !SMSAllowed(v, country) {
		return "", false, NotThisCountry
	}
	return v, false, ""
}

// MaskMobile shows a stored mobile as only its last three digits ("•••123"), for the patient's
// own contact details on their home: enough to recognise it, nothing they didn't type in full.
func MaskMobile(m string) string {
	if m == "" {
		return ""
	}
	if len(m) > 3 {
		m = m[len(m)-3:]
	}
	return "•••" + m
}

// MaskEmail shows an email address as its first letter and its domain ("s•••@example.org").
func MaskEmail(e string) string {
	at := strings.LastIndex(e, "@")
	if at < 1 {
		return ""
	}
	_, n := utf8.DecodeRuneInString(e)
	return e[:n] + "•••" + e[at:]
}
