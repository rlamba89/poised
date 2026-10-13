// Package patient holds the rules for the trust's patient records (plan C2, S13) that don't need
// the database: cleaning up what staff type, reading a search term, and deciding whether typed
// details are a patient the trust already has. The handlers in internal/httpapi run the queries.
package patient

import (
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/nyaruka/phonenumbers"

	"github.com/rlamba89/poised/apps/api/internal/nationalid"
)

// Mobile returns a phone number in E.164 (+447700900123), or "" for an empty one. A number
// without its country code is read as one of country, the deployment's PHONE_COUNTRY. ok is
// false for something that can't be a phone number.
//
// It checks the number's shape (its length for its country), not that it's a mobile in use:
// made-up test numbers (Ofcom's 07700 900xxx) must pass. Whether a text may be sent to it
// (PAT-11: mobiles in the deployment's country only) is checked when it's sent (S14).
func Mobile(s, country string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", true
	}
	n, err := phonenumbers.Parse(s, country)
	if err != nil || phonenumbers.IsPossibleNumberWithReason(n) != phonenumbers.IS_POSSIBLE {
		return "", false
	}
	return phonenumbers.Format(n, phonenumbers.E164), true
}

// Email returns the address in lower case, or "" for an empty one. ok is false unless it's a
// single plain address (no display name, no list).
func Email(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", true
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s || a.Name != "" || !strings.Contains(s[strings.LastIndex(s, "@")+1:], ".") {
		return "", false
	}
	return strings.ToLower(s), true
}

// NameKey is a name for comparing: lower case, without spaces ("Mary Ann" = "maryann").
func NameKey(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, s)
}

// Details are what staff typed about a new patient, already cleaned up: the national ID
// normalised ("" if none), the mobile in E.164 and the email in lower case ("" if none).
type Details struct {
	FirstName, LastName string
	DateOfBirth         time.Time
	NationalID          string
	Mobile, Email       string
}

// Record is one of the trust's patient records, as matching needs it.
type Record struct {
	ID                  uuid.UUID
	FirstName, LastName string
	DateOfBirth         time.Time
	NationalID          string // "" if none
	Mobile, Email       string
}

// Outcome is what creating an episode for typed details does.
type Outcome int

const (
	Create            Outcome = iota // a new trust record
	Link                             // the trust's existing Record, silently: nothing about it is shown
	Mismatch                         // the national ID is the trust's, but the date of birth or surname differ: refused
	PossibleDuplicate                // Matches have the same name and date of birth: warn, and allow Create anyway
)

// Decision is Decide's answer.
type Decision struct {
	Outcome Outcome
	Record  Record   // for Link
	Matches []Record // for PossibleDuplicate
}

// Decide applies the matching rules (decisions.md, 11 Oct; S13 answers 2) to typed details:
//   - byID is the trust's record with the typed national ID, if there is one. Its date of
//     birth and surname must agree: then link to it, else refuse (Mismatch).
//   - sameBirth are the trust's records born on the same day. Those with the same first name
//     and surname (ignoring case and spaces), and no national ID that differs from the typed
//     one, are candidates. A candidate with the same email or mobile is linked to; otherwise
//     any candidate is a possible duplicate, unless staff chose Create anyway.
func Decide(typed Details, byID *Record, sameBirth []Record, createAnyway bool) Decision {
	if typed.NationalID != "" && byID != nil {
		if sameDay(byID.DateOfBirth, typed.DateOfBirth) && NameKey(byID.LastName) == NameKey(typed.LastName) {
			return Decision{Outcome: Link, Record: *byID}
		}
		return Decision{Outcome: Mismatch}
	}
	var named []Record
	for _, r := range sameBirth {
		if !sameDay(r.DateOfBirth, typed.DateOfBirth) || NameKey(r.FirstName) != NameKey(typed.FirstName) ||
			NameKey(r.LastName) != NameKey(typed.LastName) {
			continue
		}
		if typed.NationalID != "" && r.NationalID != "" { // a different national ID: someone else
			continue
		}
		named = append(named, r)
	}
	for _, r := range named {
		if (typed.Email != "" && r.Email == typed.Email) || (typed.Mobile != "" && r.Mobile == typed.Mobile) {
			return Decision{Outcome: Link, Record: r}
		}
	}
	if len(named) > 0 && !createAnyway {
		return Decision{Outcome: PossibleDuplicate, Matches: named}
	}
	return Decision{Outcome: Create}
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// Search is a patient search term, read by ParseSearch. The query matches this hospital's
// patients whose national ID is NationalID (when the term is a valid one), whose hospital
// number here is Term, or whose surname starts with Term; born on Birth when it's set.
type Search struct {
	Term       string
	NationalID string // the term as a national ID, if it's a valid one
	Birth      *time.Time
}

// TooBroad is the message for a term that would list too many patients.
func TooBroad(scheme nationalid.Scheme) string {
	return "Type at least 3 letters of the surname, an " + scheme.Label + " or a hospital number. " +
		"Add the date of birth to search a shorter surname."
}

// ParseSearch reads a search: the term staff typed and an optional date of birth (YYYY-MM-DD).
// Nobody can list everyone: the term needs 3 characters, or 1 with a date of birth.
func ParseSearch(q, dob string, scheme nationalid.Scheme) (Search, string) {
	s := Search{Term: strings.TrimSpace(q)}
	if utf8.RuneCountInString(s.Term) > 100 {
		return s, "The search is too long."
	}
	if dob = strings.TrimSpace(dob); dob != "" {
		d, err := time.Parse(time.DateOnly, dob)
		if err != nil {
			return s, "Enter a valid date of birth."
		}
		s.Birth = &d
	}
	if n := utf8.RuneCountInString(s.Term); n == 0 || (n < 3 && s.Birth == nil) {
		return s, TooBroad(scheme)
	}
	if id := scheme.Normalise(s.Term); scheme.Valid(id) {
		s.NationalID = id
	}
	return s, ""
}

// SMSAllowed reports whether a text may be sent to a stored (E.164) mobile: only to numbers with
// the deployment's country calling code (PAT-11; country is PHONE_COUNTRY). That blocks
// SMS-pumping fraud, where someone makes the service text expensive foreign numbers. Countries
// that share a calling code (+44 with the Crown Dependencies, +1 across North America) pass
// together: the code is what the SMS price depends on.
func SMSAllowed(mobile, country string) bool {
	if mobile == "" {
		return false
	}
	n, err := phonenumbers.Parse(mobile, country)
	if err != nil {
		return false
	}
	return n.GetCountryCode() == int32(phonenumbers.GetCountryCodeForRegion(country))
}
