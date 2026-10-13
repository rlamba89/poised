// Package config reads the API's settings from environment variables, once, at start-up.
//
// It is the only Go package that may hold a per-country value (A-20, saas-requirements §13
// "Countries"): the web address, the default time zone and locale, the phone country and the
// patient identifier scheme.
// Each deployment sets them in its environment; local dev falls back to the UK defaults below.
// With CONFIG_STRICT=true (on AWS) there is no fallback: a missing value stops start-up, so a
// deployment can never quietly run as the UK.
//
// Fetching the database URL and token secret from SSM comes in S06 (AWS phase).
package config

import (
	"errors"
	"fmt"
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
	Country     Country
}

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
	Locale     string           `json:"locale"`
	TimeZone   string           `json:"timeZone"`
	NationalID PublicNationalID `json:"nationalId"`
}

// PublicNationalID names the patient identifier, so staff screens can label it ("NHS number").
type PublicNationalID struct {
	Type  string `json:"type"`
	Label string `json:"label"`
}

// Public returns the values the web app may read.
func (c Config) Public() Public {
	id := c.Country.NationalID()
	return Public{Locale: c.Country.Locale, TimeZone: c.Country.TimeZone,
		NationalID: PublicNationalID{Type: id.Type, Label: id.Label}}
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
	localeRe  = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
	countryRe = regexp.MustCompile(`^[A-Z]{2}$`)
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

	c := Config{
		DatabaseURL: need("DATABASE_URL"),
		TokenSecret: need("TOKEN_SECRET"),
		Addr:        getenv("API_ADDR"),
		DevLogin:    flag("DEV_LOGIN"),
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
