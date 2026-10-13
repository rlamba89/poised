package config_test

import (
	"strings"
	"testing"

	"github.com/rlamba89/poised/apps/api/internal/config"
)

// env is a fake environment: Load reads only through the getenv it's given.
type env map[string]string

func (e env) get(name string) string { return e[name] }

// required are the values that have no default anywhere.
func required() env {
	return env{"DATABASE_URL": "postgres://x", "TOKEN_SECRET": "s3cret"}
}

// country is a complete, made-up second country, so no test leans on the UK defaults.
func country() env {
	e := required()
	e["APP_URL"] = "https://ca.example.test"
	e["DEFAULT_TIME_ZONE"] = "America/Toronto"
	e["DEFAULT_LOCALE"] = "en-CA"
	e["PHONE_COUNTRY"] = "CA"
	e["NATIONAL_ID_TYPE"] = "nhs" // the only scheme built so far (internal/nationalid)
	return e
}

func TestLoadLocalDefaults(t *testing.T) {
	c, err := config.Load(required().get)
	if err != nil {
		t.Fatal(err)
	}
	if c.DatabaseURL != "postgres://x" || c.TokenSecret != "s3cret" {
		t.Errorf("database URL and secret = %q, %q", c.DatabaseURL, c.TokenSecret)
	}
	want := config.Country{AppURL: "http://localhost:3000", TimeZone: "Europe/London", Locale: "en-GB", PhoneCountry: "GB", NationalIDType: "nhs"}
	if c.Country != want {
		t.Errorf("country = %+v, want the local UK defaults %+v", c.Country, want)
	}
	if c.Addr != ":8080" {
		t.Errorf("addr = %q, want :8080", c.Addr)
	}
	if c.DevLogin {
		t.Error("the dev login is on by default; it must be off unless DEV_LOGIN=true")
	}
}

func TestLoadReadsEveryValue(t *testing.T) {
	e := country()
	e["API_ADDR"] = ":8081"
	e["DEV_LOGIN"] = "true"
	c, err := config.Load(e.get)
	if err != nil {
		t.Fatal(err)
	}
	want := config.Country{AppURL: "https://ca.example.test", TimeZone: "America/Toronto", Locale: "en-CA", PhoneCountry: "CA", NationalIDType: "nhs"}
	if c.Country != want {
		t.Errorf("country = %+v, want %+v", c.Country, want)
	}
	if c.Addr != ":8081" || !c.DevLogin {
		t.Errorf("addr, dev login = %q, %v", c.Addr, c.DevLogin)
	}
}

func TestLoadOneOverrideKeepsTheOtherDefaults(t *testing.T) {
	e := required()
	e["DEFAULT_LOCALE"] = "en-US"
	c, err := config.Load(e.get)
	if err != nil {
		t.Fatal(err)
	}
	if c.Country.Locale != "en-US" || c.Country.TimeZone != "Europe/London" {
		t.Errorf("country = %+v", c.Country)
	}
}

func TestLoadTrimsTheAppURLsTrailingSlash(t *testing.T) {
	e := country()
	e["APP_URL"] = "https://ca.example.test/"
	c, err := config.Load(e.get)
	if err != nil {
		t.Fatal(err)
	}
	if c.Country.AppURL != "https://ca.example.test" {
		t.Errorf("app URL = %q", c.Country.AppURL)
	}
}

func TestLoadNeedsTheDatabaseAndSecretEvenLocally(t *testing.T) {
	for _, name := range []string{"DATABASE_URL", "TOKEN_SECRET"} {
		e := required()
		delete(e, name)
		_, err := config.Load(e.get)
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("without %s: err = %v, want one naming it", name, err)
		}
	}
}

func TestLoadStrict(t *testing.T) {
	t.Run("a complete country starts", func(t *testing.T) {
		e := country()
		e["CONFIG_STRICT"] = "true"
		e["MESSAGING"] = "capture" // not a country value, but strict needs it too (S14 verification 1)
		if _, err := config.Load(e.get); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("every missing value is named, with no UK fallback", func(t *testing.T) {
		e := required()
		e["CONFIG_STRICT"] = "true"
		c, err := config.Load(e.get)
		if err == nil {
			t.Fatalf("no error; got %+v", c)
		}
		for _, name := range []string{"APP_URL", "DEFAULT_TIME_ZONE", "DEFAULT_LOCALE", "PHONE_COUNTRY", "NATIONAL_ID_TYPE"} {
			if !strings.Contains(err.Error(), name) {
				t.Errorf("error %q doesn't name %s", err, name)
			}
		}
	})
	t.Run("one missing value stops start-up", func(t *testing.T) {
		e := country()
		e["CONFIG_STRICT"] = "true"
		delete(e, "PHONE_COUNTRY")
		_, err := config.Load(e.get)
		if err == nil || !strings.Contains(err.Error(), "PHONE_COUNTRY") {
			t.Fatalf("err = %v, want one naming PHONE_COUNTRY", err)
		}
		if strings.Contains(err.Error(), "APP_URL") {
			t.Errorf("err = %v names a value that is set", err)
		}
	})
	t.Run("the database and secret are still needed", func(t *testing.T) {
		e := country()
		e["CONFIG_STRICT"] = "true"
		delete(e, "TOKEN_SECRET")
		if _, err := config.Load(e.get); err == nil || !strings.Contains(err.Error(), "TOKEN_SECRET") {
			t.Fatalf("err = %v, want one naming TOKEN_SECRET", err)
		}
	})
}

func TestLoadRefusesBadValues(t *testing.T) {
	cases := []struct{ name, value string }{
		{"DEFAULT_TIME_ZONE", "Mars/Olympus"},
		{"DEFAULT_TIME_ZONE", "Local"}, // the server's own zone: not a country's
		{"DEFAULT_LOCALE", "english"},
		{"DEFAULT_LOCALE", "en_GB"},
		{"PHONE_COUNTRY", "gb"},
		{"PHONE_COUNTRY", "GBR"},
		{"NATIONAL_ID_TYPE", "ssn"}, // no validator for it
		{"NATIONAL_ID_TYPE", "NHS"},
		{"APP_URL", "uk.example.test"},
		{"APP_URL", "ftp://uk.example.test"},
		{"APP_URL", "https://"},
		{"DEV_LOGIN", "yes please"},
		{"CONFIG_STRICT", "maybe"},
	}
	for _, tc := range cases {
		t.Run(tc.name+"="+tc.value, func(t *testing.T) {
			for _, strict := range []string{"false", "true"} {
				e := country()
				e["CONFIG_STRICT"] = strict
				e[tc.name] = tc.value
				_, err := config.Load(e.get)
				if err == nil || !strings.Contains(err.Error(), tc.name) {
					t.Errorf("strict=%s: err = %v, want one naming %s", strict, err, tc.name)
				}
			}
		})
	}
}

func TestPublic(t *testing.T) {
	c, err := config.Load(country().get)
	if err != nil {
		t.Fatal(err)
	}
	got := c.Public()
	want := config.Public{Locale: "en-CA", TimeZone: "America/Toronto",
		NationalID: config.PublicNationalID{Type: "nhs", Label: "NHS number"}}
	if got != want {
		t.Errorf("public = %+v", got)
	}
}

func TestLoadMessaging(t *testing.T) {
	t.Run("capture is the local default: nothing is ever sent by accident", func(t *testing.T) {
		c, err := config.Load(required().get)
		if err != nil || c.Messaging != "capture" {
			t.Fatalf("got %q, %v", c.Messaging, err)
		}
	})
	t.Run("strict (AWS): MESSAGING must be set, so the unauthenticated captured-messages route is never on by default", func(t *testing.T) {
		e := country()
		e["CONFIG_STRICT"] = "true"
		_, err := config.Load(e.get)
		if err == nil || !strings.Contains(err.Error(), "MESSAGING") {
			t.Fatalf("got %v", err)
		}
		e["MESSAGING"] = "capture" // QA opts in, as it does to DEV_LOGIN
		if c, err := config.Load(e.get); err != nil || c.Messaging != "capture" {
			t.Fatalf("got %q, %v", c.Messaging, err)
		}
	})
	t.Run("only capture exists until S17", func(t *testing.T) {
		e := required()
		e["MESSAGING"] = "aws"
		_, err := config.Load(e.get)
		if err == nil || !strings.Contains(err.Error(), "MESSAGING") {
			t.Fatalf("got %v", err)
		}
	})
}
