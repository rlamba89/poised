// Package settings is a trust's and its hospitals' settings (ORG-04, ORG-05; plan C1, S10).
//
// Settings resolve platform default → trust → hospital (Effective). Each setting is either
// trust-only, or one a hospital can override; a hospital that sets nothing follows its trust.
// The values are stored as flat JSON objects, {"contact.phone": "0100 …", "feature.proms": true},
// in orgs.settings and hospitals.settings; a key that isn't stored is "not set here".
//
// Every ORG-04 setting is defined now, though most are read only by later sessions (Rahul,
// 12 Oct, S10 answers 2): reminders (S16), auto-archive (C3), "what happens next" and the patient
// message templates (S14–S16), the default HQ (S13), retention (C3/C10), pre-fill (C4).
//
// The opt-ins (feature switches and licensed instruments) are set by the platform admin only
// (PRM-08: they can be sold as add-ons); trust admins see them but can't change them.
package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Level says where a setting can be set.
type Level int

const (
	TrustOnly Level = iota // set for the whole trust only
	Hospital               // the trust sets a default; each hospital may set its own
)

// MarshalJSON writes the level as "trust" or "hospital", for the web app.
func (l Level) MarshalJSON() ([]byte, error) {
	if l == TrustOnly {
		return []byte(`"trust"`), nil
	}
	return []byte(`"hospital"`), nil
}

// Kind is a setting's type: what Validate accepts and which field the web app shows.
type Kind string

const (
	Bool          Kind = "bool"
	Int           Kind = "int"           // a whole number from Min to Max
	DayList       Kind = "dayList"       // up to 5 different whole numbers from Min to Max, kept in order
	Text          Kind = "text"          // up to Max characters, trimmed
	URL           Kind = "url"           // "" or an http(s) address
	Email         Kind = "email"         // "" or an email address, kept in lower case
	Phone         Kind = "phone"         // "" or a phone number: digits, spaces, + ( ) -
	Questionnaire Kind = "questionnaire" // "" or the id of one of the trust's questionnaires
	Template      Kind = "template"      // patient message text with {{placeholders}}; must hold {{link}}
)

// Def is one setting.
type Def struct {
	Key     string          `json:"key"`
	Label   string          `json:"label"`
	Help    string          `json:"help,omitempty"`
	Group   string          `json:"group"` // the heading it's shown under
	Level   Level           `json:"level"`
	Kind    Kind            `json:"kind"`
	OptIn   bool            `json:"optIn"` // set by the platform admin only (ORG-05, PRM-08)
	Default json.RawMessage `json:"default"`
	Min     int             `json:"min,omitempty"`
	Max     int             `json:"max,omitempty"` // for text: the most characters
}

// Placeholders are the names a message template may use.
var Placeholders = []string{"firstName", "hospital", "link"}

const (
	gTrust     = "Whole trust"
	gOptIns    = "Features and opt-ins"
	gPatients  = "What patients see"
	gWorkflow  = "Reminders and archiving"
	gTemplates = "Patient messages"
)

const templateHelp = "Use {{firstName}}, {{hospital}} and {{link}}. {{link}} is required."

// Defs is every setting, in the order screens show them.
var Defs = []Def{
	{Key: "retentionYears", Label: "Keep episode data for (years)", Group: gTrust, Level: TrustOnly, Kind: Int,
		Help:    "After this many years from the end of an episode, its data is deleted (EPI-06). The trust, as controller, confirms it.",
		Default: j(8), Min: 1, Max: 100},
	{Key: "crossTrustPrefill", Label: "Offer answers patients gave other trusts", Group: gTrust, Level: TrustOnly, Kind: Bool,
		Help: "The patient is always asked first.", Default: j(false)},

	{Key: "feature.assessments", Label: "Assessments", Group: gOptIns, Level: Hospital, Kind: Bool, OptIn: true, Default: j(true)},
	{Key: "feature.proms", Label: "PROMs", Group: gOptIns, Level: Hospital, Kind: Bool, OptIn: true, Default: j(false)},
	{Key: "feature.files", Label: "Files", Group: gOptIns, Level: Hospital, Kind: Bool, OptIn: true, Default: j(true)},
	{Key: "feature.observations", Label: "Observations", Group: gOptIns, Level: Hospital, Kind: Bool, OptIn: true, Default: j(true)},
	{Key: "instrument.eq5d5l", Label: "EQ-5D-5L (licensed)", Group: gOptIns, Level: Hospital, Kind: Bool, OptIn: true, Default: j(false)},
	{Key: "instrument.oxfordHip", Label: "Oxford Hip Score (licensed)", Group: gOptIns, Level: Hospital, Kind: Bool, OptIn: true, Default: j(false)},
	{Key: "instrument.oxfordKnee", Label: "Oxford Knee Score (licensed)", Group: gOptIns, Level: Hospital, Kind: Bool, OptIn: true, Default: j(false)},

	{Key: "branding.logoUrl", Label: "Logo web address", Group: gPatients, Level: Hospital, Kind: URL,
		Help: "A link to the logo image. Uploading a file comes later.", Default: j(""), Max: 2000},
	{Key: "contact.phone", Label: "Phone number for patients", Group: gPatients, Level: Hospital, Kind: Phone, Default: j("")},
	{Key: "contact.email", Label: "Email address for patients", Group: gPatients, Level: Hospital, Kind: Email, Default: j("")},
	{Key: "defaultHq", Label: "Default health questionnaire", Group: gPatients, Level: Hospital, Kind: Questionnaire,
		Help: "Chosen first when a new episode is created.", Default: j("")},
	{Key: "whatHappensNext", Label: "\"What happens next\" text", Group: gPatients, Level: Hospital, Kind: Text,
		Help:    "Shown to the patient after they send their answers.",
		Default: j("Thank you. Your answers have been sent to the pre-operative assessment team. They will contact you if they need anything else."),
		Max:     2000},

	{Key: "reminders.days", Label: "Send reminders on these days after the invite", Group: gWorkflow, Level: Hospital, Kind: DayList,
		Default: j([]int{2, 5}), Min: 1, Max: 60},
	{Key: "reminders.notRespondingDay", Label: "Mark \"not responding\" on day", Group: gWorkflow, Level: Hospital, Kind: Int,
		Default: j(7), Min: 1, Max: 90},
	{Key: "autoArchive.afterDischargeDays", Label: "Archive episodes this many days after discharge", Group: gWorkflow, Level: Hospital, Kind: Int,
		Default: j(30), Min: 1, Max: 365},
	{Key: "autoArchive.inactiveMonths", Label: "Archive pre-op episodes with no activity for (months)", Group: gWorkflow, Level: Hospital, Kind: Int,
		Default: j(6), Min: 1, Max: 36},

	{Key: "templates.inviteSms", Label: "Invitation text message", Group: gTemplates, Level: Hospital, Kind: Template, Help: templateHelp,
		Default: j("Hello {{firstName}}, {{hospital}} needs some health information before your operation. Please fill in your questionnaire: {{link}}"),
		Max:     600},
	{Key: "templates.inviteEmail", Label: "Invitation email", Group: gTemplates, Level: Hospital, Kind: Template, Help: templateHelp,
		Default: j("Hello {{firstName}},\n\n{{hospital}} needs some health information before your operation. Please fill in your health questionnaire:\n\n{{link}}"),
		Max:     5000},
	{Key: "templates.reminderSms", Label: "Reminder text message", Group: gTemplates, Level: Hospital, Kind: Template, Help: templateHelp,
		Default: j("Hello {{firstName}}, a reminder from {{hospital}}: please fill in your health questionnaire: {{link}}"),
		Max:     600},
	{Key: "templates.reminderEmail", Label: "Reminder email", Group: gTemplates, Level: Hospital, Kind: Template, Help: templateHelp,
		Default: j("Hello {{firstName}},\n\nThis is a reminder from {{hospital}}: please fill in your health questionnaire before your operation:\n\n{{link}}"),
		Max:     5000},
}

func j(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// Values are settings as stored: a key that's missing isn't set at that level.
type Values map[string]json.RawMessage

// Lookup finds a setting by key.
func Lookup(key string) (Def, bool) {
	i := slices.IndexFunc(Defs, func(d Def) bool { return d.Key == key })
	if i < 0 {
		return Def{}, false
	}
	return Defs[i], true
}

// IsOptIn reports whether key is a feature switch or licensed instrument (ORG-05).
func IsOptIn(key string) bool {
	d, ok := Lookup(key)
	return ok && d.OptIn
}

// Platform is the platform's defaults: every setting's value when nobody has set it.
func Platform() Values {
	v := Values{}
	for _, d := range Defs {
		v[d.Key] = d.Default
	}
	return v
}

// Effective is every setting's value at a hospital (or, with hospital nil, for the trust
// itself): the hospital's own value where it may override and has set one, else the trust's,
// else the platform's. Keys that aren't settings are ignored.
func Effective(platform, org, hospital Values) Values {
	out := Values{}
	for _, d := range Defs {
		if v, ok := hospital[d.Key]; ok && d.Level == Hospital {
			out[d.Key] = v
		} else if v, ok := org[d.Key]; ok {
			out[d.Key] = v
		} else {
			out[d.Key] = platform[d.Key]
		}
	}
	return out
}

var (
	phoneFormat = regexp.MustCompile(`^\+?[0-9 ()\-]{6,30}$`)
	placeholder = regexp.MustCompile(`\{\{([^{}]*)\}\}`)
)

// Validate checks a value for key and returns it in its stored form (trimmed, sorted, lower
// case where that applies), or a plain-language refusal.
func Validate(key string, raw json.RawMessage) (json.RawMessage, string) {
	d, ok := Lookup(key)
	if !ok {
		return nil, fmt.Sprintf("%q isn't a setting.", key)
	}
	raw = bytes.TrimSpace(raw)
	switch d.Kind {
	case Bool:
		var b bool
		if !strict(raw, &b) {
			return nil, d.Label + ": choose on or off."
		}
		return j(b), ""
	case Int:
		var n int
		if !strict(raw, &n) || n < d.Min || n > d.Max {
			return nil, fmt.Sprintf("%s: enter a whole number from %d to %d.", d.Label, d.Min, d.Max)
		}
		return j(n), ""
	case DayList:
		var days []int
		bad := fmt.Sprintf("%s: enter up to 5 different days, each from %d to %d.", d.Label, d.Min, d.Max)
		if !strict(raw, &days) || days == nil || len(days) > 5 {
			return nil, bad
		}
		slices.Sort(days)
		for i, n := range days {
			if n < d.Min || n > d.Max || (i > 0 && days[i-1] == n) {
				return nil, bad
			}
		}
		return j(days), ""
	}

	var s string
	if !strict(raw, &s) {
		return nil, d.Label + ": enter some text."
	}
	s = strings.TrimSpace(s)
	switch d.Kind {
	case Text:
		if utf8.RuneCountInString(s) > d.Max {
			return nil, fmt.Sprintf("%s: use %d characters or fewer.", d.Label, d.Max)
		}
	case URL:
		if s != "" {
			u, err := url.Parse(s)
			if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || len(s) > d.Max {
				return nil, d.Label + ": enter a web address starting with https://, or leave it empty."
			}
		}
	case Email:
		s = strings.ToLower(s)
		if s != "" {
			a, err := mail.ParseAddress(s)
			if err != nil || a.Address != s || !strings.Contains(s[strings.LastIndex(s, "@"):], ".") {
				return nil, d.Label + ": enter a valid email address, or leave it empty."
			}
		}
	case Phone:
		if s != "" && !phoneFormat.MatchString(s) {
			return nil, d.Label + ": enter a phone number (digits, spaces and + ( ) - only), or leave it empty."
		}
	case Questionnaire:
		if s != "" {
			if _, err := uuid.Parse(s); err != nil {
				return nil, d.Label + ": choose one of the trust's questionnaires."
			}
		}
	case Template:
		if s == "" || utf8.RuneCountInString(s) > d.Max {
			return nil, fmt.Sprintf("%s: enter the message, in %d characters or fewer.", d.Label, d.Max)
		}
		hasLink := false
		for _, m := range placeholder.FindAllStringSubmatch(s, -1) {
			if !slices.Contains(Placeholders, m[1]) {
				return nil, fmt.Sprintf("%s: {{%s}} isn't a placeholder. %s", d.Label, m[1], templateHelp)
			}
			hasLink = hasLink || m[1] == "link"
		}
		if !hasLink {
			return nil, d.Label + ": the message must include {{link}}, so the patient can open their questionnaire."
		}
	}
	return j(s), ""
}

// strict decodes raw into v, refusing null and anything of another type.
func strict(raw json.RawMessage, v any) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	return json.Unmarshal(raw, v) == nil
}

// PhoneOK reports whether s looks like a phone number of any country: 6 to 30 digits, spaces and
// + ( ) -. It isn't checked against a numbering plan (A-20: nothing country-specific here).
func PhoneOK(s string) bool { return phoneFormat.MatchString(s) }

// Decode reads stored settings (a JSON object), keeping only known keys, each in compact form
// (Postgres writes jsonb arrays as [1, 5]).
func Decode(raw json.RawMessage) (Values, error) {
	var all map[string]json.RawMessage
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, err
	}
	out := Values{}
	for k, v := range all {
		if _, ok := Lookup(k); !ok {
			continue
		}
		var b bytes.Buffer
		if err := json.Compact(&b, v); err != nil {
			return nil, err
		}
		out[k] = b.Bytes()
	}
	return out, nil
}

// Apply returns stored with changes made: a nil value removes the key ("not set here"). Changes
// must have been checked with Validate.
func Apply(stored Values, changes map[string]json.RawMessage) Values {
	out := Values{}
	for k, v := range stored {
		out[k] = v
	}
	for k, v := range changes {
		if v == nil {
			delete(out, k)
		} else {
			out[k] = v
		}
	}
	return out
}
