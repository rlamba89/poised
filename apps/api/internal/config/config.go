// Package config reads the API's settings from environment variables, once, at start-up.
//
// It is the only Go package that may hold a per-country value (A-20, saas-requirements §13
// "Countries"): the web address, the default time zone and locale, the phone country and the
// patient identifier scheme.
// Each deployment sets them in its environment; local dev falls back to the UK defaults below.
// With CONFIG_STRICT=true (on AWS) there is no fallback: a missing value stops start-up, so a
// deployment can never quietly run as the UK.
//
// On AWS (S06) the database URL and the web address come from SSM Parameter Store and the token
// secret from Secrets Manager, fetched once at cold start by Resolve; the environment only names them.
package config

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // the zone database is built in, so no zone depends on the host (e.g. Lambda)

	"github.com/rlamba89/poised/apps/api/internal/nationalid"
)

// Config is everything the API reads from its environment.
type Config struct {
	DatabaseURL string // DATABASE_URL
	TokenSecret string // TOKEN_SECRET: signs the session cookie
	Addr        string // API_ADDR: where a local server listens (default :8080)
	DevLogin    bool   // DEV_LOGIN=true: the stub sign-in routes. Never on in Training or Production.
	// MESSAGING: how patient SMS and email go out (required explicitly with CONFIG_STRICT):
	//   - "capture" (the local default; QA) stores them in sent_messages and sends nothing; the
	//     testers' tools (GET /api/qa/messages…) read them;
	//   - "send" (S17; Training and Production) sends them with SES and End User Messaging SMS.
	Messaging string
	// Sending is how real messages go (S17): with MESSAGING=send, or with test recipients.
	Sending Sending
	// MESSAGING_TEST_RECIPIENTS (capture only; QA): a comma-separated list of E.164 mobiles and
	// email addresses that are also really sent their messages (messaging.TestRecipients), so a
	// real text and email can be tried on QA without anyone else receiving one.
	TestRecipients []string
	// JOBS_TIMER: the local all-in-one server (cmd/api, make dev) runs the scheduled jobs every
	// minute (S16) unless it's false. On AWS a schedule runs the jobs Lambda instead (S17).
	JobsTimer bool
	// JOBS_SCHEDULE_MINUTES (S17, set by infra on AWS): how often the jobs Lambda's schedule runs
	// the jobs there, for the testers' jobs page. 0: no schedule (local: the timer runs them).
	JobsScheduleMinutes int
	// FORWARDED_HOPS: how many trusted proxies sit in front of the API, each adding the address
	// it got the request from to X-Forwarded-For. 0 (locally): the connection's address is the
	// client's. On AWS it's set by infra (internal/serve reads the client's address from the header).
	ForwardedHops int
	// SECONDS_PER_QUESTION (S36, PX-10): the estimated time a patient is shown for a Question Set
	// counts this many seconds for each question they see, plus each video's length (S36 answers 3:
	// configurable, default 15). Not a per-country value, so it keeps its default under CONFIG_STRICT.
	SecondsPerQuestion int
	Country            Country
	// Cognito is the staff user pool (S12): staff sign in through its managed login. Required on
	// AWS (CONFIG_STRICT); locally it's usually unset and the dev login stands in.
	Cognito Cognito
}

// Cognito names the staff user pool and its app client (S12). None of it is secret: the app
// client is public (no client secret), and the code exchange is protected by PKCE. The variables
// are STAFF_POOL_*, not COGNITO_*: a developer's .env may hold another project's COGNITO_* values,
// which make exports (found in S12).
type Cognito struct {
	UserPoolID string // STAFF_POOL_ID, e.g. <region>_AbCdEf123
	ClientID   string // STAFF_POOL_CLIENT_ID: the app client the managed login signs in to
	// STAFF_POOL_DOMAIN: the managed login's address, e.g. https://<prefix>.auth.<region>.amazoncognito.com
	Domain string
	// RedirectURI is where the managed login sends the browser back with the code: the app's
	// /auth/callback page (APP_URL). The code is exchanged with the same address.
	RedirectURI string
}

// Enabled reports whether staff sign in with Cognito here.
func (c Cognito) Enabled() bool { return c.UserPoolID != "" }

// Region is the pool's AWS region: the part of its ID before the underscore.
func (c Cognito) Region() string { return strings.SplitN(c.UserPoolID, "_", 2)[0] }

// Issuer is the iss claim of the pool's ID tokens; its keys are at Issuer + /.well-known/jwks.json.
func (c Cognito) Issuer() string {
	return "https://cognito-idp." + c.Region() + ".amazonaws.com/" + c.UserPoolID
}

// DefaultSecondsPerQuestion is SECONDS_PER_QUESTION's value when it isn't set (S36 answers 3), and
// maxSecondsPerQuestion a sanity limit (10 minutes a question is surely a typing mistake).
const (
	DefaultSecondsPerQuestion = 15
	maxSecondsPerQuestion     = 600
)

// maxForwardedHops is a sanity limit: on QA the browser's address is third from the right
// (Amplify's rewrite adds two, API Gateway one; infra/api_stack.py).
const maxForwardedHops = 4

// MessagingCapture stores patient messages instead of sending them (MESSAGING=capture).
const MessagingCapture = "capture"

// MessagingSend sends them with SES and End User Messaging SMS (MESSAGING=send, S17).
const MessagingSend = "send"

// Sending names the deployment's SES sender and the configuration sets its messages go through
// (infra's MessagingStack makes them; their event destinations report undelivered messages).
type Sending struct {
	EmailFrom           string // EMAIL_FROM: the verified address emails come from
	SESConfigurationSet string // SES_CONFIGURATION_SET
	SMSConfigurationSet string // SMS_CONFIGURATION_SET: its protect configuration allows only PHONE_COUNTRY
	SMSSenderID         string // SMS_SENDER_ID: optional; the texts' origination identity
}

// Enabled reports whether any message is really sent here.
func (s Sending) Enabled() bool { return s.EmailFrom != "" }

// Country holds the values that differ between country deployments (A-20).
type Country struct {
	AppURL       string // APP_URL: the web address, for links in emails and SMS (no trailing slash)
	TimeZone     string // DEFAULT_TIME_ZONE: an IANA zone
	Locale       string // DEFAULT_LOCALE: a BCP 47 tag, for formatting dates
	PhoneCountry string // PHONE_COUNTRY: ISO 3166-1 alpha-2, the only country SMS goes to (PAT-11)
	// NATIONAL_ID_TYPE: the patient identifier scheme, a type in internal/nationalid (S13)
	NationalIDType string
}

// NationalID is the deployment's patient identifier scheme. Load has checked it exists.
func (c Country) NationalID() nationalid.Scheme {
	s, _ := nationalid.Lookup(c.NationalIDType)
	return s
}

// Public is what GET /api/config tells the browser. Nothing secret goes in here.
type Public struct {
	Locale      string           `json:"locale"`
	TimeZone    string           `json:"timeZone"`
	NationalID  PublicNationalID `json:"nationalId"`
	StaffSignIn PublicSignIn     `json:"staffSignIn"`
	// SecondsPerQuestion is the estimated time's seconds for each question (S36): packages/clinical's
	// estimatedMinutes uses it, so the number lives only here.
	SecondsPerQuestion int `json:"secondsPerQuestion"`
}

// PublicSignIn is how staff sign in here (S12): the dev login (local only), Cognito's managed
// login, or, locally, both.
type PublicSignIn struct {
	DevLogin bool           `json:"devLogin"`
	Cognito  *PublicCognito `json:"cognito"` // null without Cognito
}

// PublicCognito is what the sign-in page needs to send the browser to the managed login (the
// OAuth code flow with PKCE) and to sign out of it.
type PublicCognito struct {
	AuthorizeURL string `json:"authorizeUrl"`
	LogoutURL    string `json:"logoutUrl"`
	ClientID     string `json:"clientId"`
	RedirectURI  string `json:"redirectUri"`  // the app's /auth/callback
	SignedOutURI string `json:"signedOutUri"` // the app's /login, after signing out of the managed login
}

// PublicNationalID names the patient identifier, so staff screens can label it ("NHS number").
type PublicNationalID struct {
	Type  string `json:"type"`
	Label string `json:"label"`
}

// Public returns the values the web app may read.
func (c Config) Public() Public {
	id := c.Country.NationalID()
	p := Public{Locale: c.Country.Locale, TimeZone: c.Country.TimeZone,
		NationalID:  PublicNationalID{Type: id.Type, Label: id.Label},
		StaffSignIn: PublicSignIn{DevLogin: c.DevLogin}, SecondsPerQuestion: c.SecondsPerQuestion}
	if c.Cognito.Enabled() {
		p.StaffSignIn.Cognito = &PublicCognito{
			AuthorizeURL: c.Cognito.Domain + "/oauth2/authorize",
			LogoutURL:    c.Cognito.Domain + "/logout",
			ClientID:     c.Cognito.ClientID,
			RedirectURI:  c.Cognito.RedirectURI,
			SignedOutURI: c.Country.AppURL + "/login",
		}
	}
	return p
}

// localDefaults are used only outside strict mode, for development on a laptop.
var localDefaults = map[string]string{
	"APP_URL":           "http://localhost:3000",
	"DEFAULT_TIME_ZONE": "Europe/London",
	"DEFAULT_LOCALE":    "en-GB",
	"PHONE_COUNTRY":     "GB",
	"NATIONAL_ID_TYPE":  "nhs",
}

var (
	localeRe   = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
	countryRe  = regexp.MustCompile(`^[A-Z]{2}$`)
	poolIDRe   = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-\d_[A-Za-z0-9]+$`)
	e164Re     = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)
	senderIDRe = regexp.MustCompile(`^[A-Za-z0-9-]{1,11}$`)
	clientIDRe = regexp.MustCompile(`^[A-Za-z0-9]+$`)
)

// Load reads the configuration through getenv (os.Getenv in a real server, a map in tests).
// The error names every missing or invalid variable at once.
func Load(getenv func(string) string) (Config, error) {
	var errs []error
	flag := func(name string) bool {
		v := getenv(name)
		if v == "" {
			return false
		}
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s must be true or false, not %q", name, v))
		}
		return b
	}
	strict := flag("CONFIG_STRICT")
	if getenv("AWS_LAMBDA_FUNCTION_NAME") != "" && !strict {
		errs = append(errs, errors.New("CONFIG_STRICT must be true on Lambda, so a deployment never falls back to the local defaults"))
	}

	var missing []string
	need := func(name string) string {
		v := getenv(name)
		if v == "" {
			missing = append(missing, name)
		}
		return v
	}
	country := func(name string) string {
		if v := getenv(name); v != "" || strict {
			return need(name)
		}
		return localDefaults[name]
	}

	// MESSAGING: locally "capture" by default. Under strict (AWS) it must be set: capture mode
	// serves the unauthenticated GET /api/qa/messages, so a stack must opt in to it, as to DEV_LOGIN.
	messaging := func() string {
		if strict {
			return need("MESSAGING")
		}
		return getenv("MESSAGING")
	}
	jobsTimer := getenv("JOBS_TIMER") == "" || flag("JOBS_TIMER")
	c := Config{
		JobsTimer:   jobsTimer,
		DatabaseURL: need("DATABASE_URL"),
		TokenSecret: need("TOKEN_SECRET"),
		Addr:        getenv("API_ADDR"),
		DevLogin:    flag("DEV_LOGIN"),
		Messaging:   messaging(),
		Country: Country{
			AppURL:         strings.TrimSuffix(country("APP_URL"), "/"),
			TimeZone:       country("DEFAULT_TIME_ZONE"),
			Locale:         country("DEFAULT_LOCALE"),
			PhoneCountry:   country("PHONE_COUNTRY"),
			NationalIDType: country("NATIONAL_ID_TYPE"),
		},
	}
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	errs = append(errs, c.loadCognito(getenv, strict, need)...)
	if v := getenv("JOBS_SCHEDULE_MINUTES"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 24*60 {
			errs = append(errs, fmt.Errorf("JOBS_SCHEDULE_MINUTES must be a number of minutes from 1 to 1440, not %q", v))
		}
		c.JobsScheduleMinutes = n
	}
	c.SecondsPerQuestion = DefaultSecondsPerQuestion
	if v := getenv("SECONDS_PER_QUESTION"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxSecondsPerQuestion {
			errs = append(errs, fmt.Errorf("SECONDS_PER_QUESTION must be a whole number of seconds from 1 to %d, not %q", maxSecondsPerQuestion, v))
		}
		c.SecondsPerQuestion = n
	}
	if v := getenv("FORWARDED_HOPS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > maxForwardedHops {
			errs = append(errs, fmt.Errorf("FORWARDED_HOPS must be a number from 0 to %d, not %q", maxForwardedHops, v))
		}
		c.ForwardedHops = n
	}
	switch c.Messaging {
	case "":
		if !strict { // strict: reported as missing above
			c.Messaging = MessagingCapture
		}
	case MessagingCapture, MessagingSend:
	default:
		errs = append(errs, fmt.Errorf("MESSAGING must be %q or %q, not %q", MessagingCapture, MessagingSend, c.Messaging))
	}
	errs = append(errs, c.loadSending(getenv, need)...)
	if len(missing) > 0 {
		hint := " (copy .env.example to .env)"
		if strict {
			hint = " (CONFIG_STRICT is on, so there are no local defaults)"
		}
		errs = append(errs, fmt.Errorf("not set: %s%s", strings.Join(missing, ", "), hint))
	}
	errs = append(errs, c.Country.validate()...)
	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return c, nil
}

// loadSending reads how real messages go (S17): needed for MESSAGING=send, and for capture with
// test recipients (QA), which sends to them only.
func (c *Config) loadSending(getenv func(string) string, need func(string) string) []error {
	var errs []error
	if v := getenv("MESSAGING_TEST_RECIPIENTS"); v != "" {
		if c.Messaging != MessagingCapture {
			errs = append(errs, errors.New("MESSAGING_TEST_RECIPIENTS goes only with MESSAGING=capture (QA): with send, everyone's messages go"))
		}
		for _, r := range strings.Split(v, ",") {
			r = strings.ToLower(strings.TrimSpace(r))
			if !e164Re.MatchString(r) && !isAddress(r) {
				errs = append(errs, errors.New("MESSAGING_TEST_RECIPIENTS must be E.164 mobiles and email addresses, comma-separated (one isn't)")) // never print a contact
				break
			}
			c.TestRecipients = append(c.TestRecipients, r)
		}
	}
	if c.Messaging != MessagingSend && len(c.TestRecipients) == 0 {
		return errs
	}
	c.Sending = Sending{
		EmailFrom:           need("EMAIL_FROM"),
		SESConfigurationSet: need("SES_CONFIGURATION_SET"),
		SMSConfigurationSet: need("SMS_CONFIGURATION_SET"),
		SMSSenderID:         getenv("SMS_SENDER_ID"),
	}
	if v := c.Sending.EmailFrom; v != "" && !isAddress(v) {
		errs = append(errs, fmt.Errorf("EMAIL_FROM must be a plain email address, such as preopsis@example.org, not %q", v))
	}
	if v := c.Sending.SMSSenderID; v != "" && !senderIDRe.MatchString(v) {
		errs = append(errs, fmt.Errorf("SMS_SENDER_ID must be 1 to 11 letters, digits or hyphens, not %q", v))
	}
	return errs
}

// isAddress reports whether v is a bare email address (no name, no angle brackets).
func isAddress(v string) bool {
	a, err := mail.ParseAddress(v)
	return err == nil && a.Name == "" && a.Address == v
}

// loadCognito reads the staff user pool (S12). On AWS (strict) it's required and the dev login
// is refused: the stub would let anyone sign in as anyone. Locally it's optional, but it's all
// three values or none.
func (c *Config) loadCognito(getenv func(string) string, strict bool, need func(string) string) []error {
	names := []string{"STAFF_POOL_ID", "STAFF_POOL_CLIENT_ID", "STAFF_POOL_DOMAIN"}
	var errs []error
	if strict && c.DevLogin {
		errs = append(errs, errors.New("DEV_LOGIN must be off with CONFIG_STRICT (AWS): staff sign in with Cognito there (S12)"))
	}
	set := false
	for _, n := range names {
		set = set || getenv(n) != ""
	}
	if !strict && !set {
		return errs
	}
	c.Cognito = Cognito{
		UserPoolID: need("STAFF_POOL_ID"),
		ClientID:   need("STAFF_POOL_CLIENT_ID"),
		Domain:     strings.TrimSuffix(need("STAFF_POOL_DOMAIN"), "/"),
	}
	if c.Country.AppURL != "" {
		c.Cognito.RedirectURI = c.Country.AppURL + "/auth/callback"
	}
	if v := c.Cognito.UserPoolID; v != "" && !poolIDRe.MatchString(v) {
		errs = append(errs, fmt.Errorf("STAFF_POOL_ID must be a user pool ID (<region>_<id>), not %q", v))
	}
	if v := c.Cognito.ClientID; v != "" && !clientIDRe.MatchString(v) {
		errs = append(errs, fmt.Errorf("STAFF_POOL_CLIENT_ID must be an app client ID (letters and digits), not %q", v))
	}
	if v := c.Cognito.Domain; v != "" {
		if u, err := url.Parse(v); err != nil || u.Scheme != "https" || u.Host == "" || u.Path != "" {
			errs = append(errs, fmt.Errorf("STAFF_POOL_DOMAIN must be the managed login's https address with no path, not %q", v))
		}
	}
	return errs
}

// validate checks the values that are set; missing ones are reported by Load.
func (c Country) validate() []error {
	var errs []error
	if c.AppURL != "" {
		u, err := url.Parse(c.AppURL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			errs = append(errs, fmt.Errorf("APP_URL must be a web address such as https://uk.example.com, not %q", c.AppURL))
		}
	}
	if c.TimeZone != "" {
		if _, err := time.LoadLocation(c.TimeZone); err != nil || c.TimeZone == "Local" {
			errs = append(errs, fmt.Errorf("DEFAULT_TIME_ZONE must be an IANA time zone such as America/Toronto, not %q", c.TimeZone))
		}
	}
	if c.Locale != "" && !localeRe.MatchString(c.Locale) {
		errs = append(errs, fmt.Errorf("DEFAULT_LOCALE must be a language tag such as en-CA, not %q", c.Locale))
	}
	if c.PhoneCountry != "" && !countryRe.MatchString(c.PhoneCountry) {
		errs = append(errs, fmt.Errorf("PHONE_COUNTRY must be a two-letter country code such as CA, not %q", c.PhoneCountry))
	}
	if c.NationalIDType != "" {
		if _, ok := nationalid.Lookup(c.NationalIDType); !ok {
			errs = append(errs, fmt.Errorf("NATIONAL_ID_TYPE must be a patient identifier scheme the API has a validator for (internal/nationalid), not %q", c.NationalIDType))
		}
	}
	return errs
}
