// Package nationalid holds the national patient identifier schemes (A-20): one validator per
// type. A deployment names its scheme in NATIONAL_ID_TYPE (internal/config); patients store
// national_id + national_id_type. Only the NHS number is built; another country's identifier
// (a Canadian health card, Australia's IHI, New Zealand's NHI) is a new entry in schemes.
package nationalid

import "strings"

// Scheme is one kind of national identifier.
type Scheme struct {
	Type  string // stored in patients.national_id_type
	Label string // what staff call it, e.g. "NHS number"
	// Normalise strips what people type around the identifier (spaces, hyphens). Store and
	// compare only normalised values.
	Normalise func(string) string
	// Valid reports whether a normalised value is a well-formed identifier (its check digit).
	Valid func(string) bool
}

var schemes = map[string]Scheme{
	"nhs": {Type: "nhs", Label: "NHS number", Normalise: digitsOnly, Valid: nhsValid},
}

// Lookup returns the scheme of a type.
func Lookup(typ string) (Scheme, bool) {
	s, ok := schemes[typ]
	return s, ok
}

// digitsOnly removes spaces and hyphens, the separators people type ("943 476 5919").
func digitsOnly(s string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '\t' {
			return -1
		}
		return r
	}, s)
}

// nhsValid is the NHS number's Modulus 11 check: ten digits, where the first nine weighted
// 10, 9, … 2 and summed give 11 − (sum mod 11) as the tenth (11 means 0; 10 is never issued).
func nhsValid(s string) bool {
	if len(s) != 10 {
		return false
	}
	sum := 0
	for i := range 10 {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
		if i < 9 {
			sum += int(s[i]-'0') * (10 - i)
		}
	}
	check := 11 - sum%11
	if check == 11 {
		check = 0
	}
	return check != 10 && check == int(s[9]-'0')
}
