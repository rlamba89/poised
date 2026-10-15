package config_test

import (
	"reflect"
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

// cognito is a made-up staff user pool (S12), in the made-up country's region.
func cognito(e env) env {
	e["STAFF_POOL_ID"] = "ca-central-1_AbCdEf123"
	e["STAFF_POOL_CLIENT_ID"] = "1example23456789client"
	e["STAFF_POOL_DOMAIN"] = "https://poised-ca-qa.auth.ca-central-1.amazoncognito.com"
	return e
}

// strict is a complete AWS deployment of the made-up country: strict, with Cognito.
func strict() env {
	e := cognito(country())
	e["CONFIG_STRICT"] = "true"
	e["MESSAGING"] = "capture"
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
	if !c.JobsTimer {
		t.Error("the jobs timer is off by default; it must be on unless JOBS_TIMER=false")
	}
}

func TestLoadJobsTimer(t *testing.T) {
	e := required()
	e["JOBS_TIMER"] = "false"
	c, err := config.Load(e.get)
	if err != nil || c.JobsTimer {
		t.Errorf("JOBS_TIMER=false: on %v, err %v", c.JobsTimer, err)
	}
	e["JOBS_TIMER"] = "true"
	if c, err := config.Load(e.get); err != nil || !c.JobsTimer {
		t.Errorf("JOBS_TIMER=true: on %v, err %v", c.JobsTimer, err)
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
		// MESSAGING isn't a country value, but strict needs it too (S14 verification 1), and
		// Cognito, the staff sign-in on AWS (S12).
		if _, err := config.Load(strict().get); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("no dev login on AWS (S12): staff sign in with Cognito", func(t *testing.T) {
		e := strict()
		e["DEV_LOGIN"] = "true"
		_, err := config.Load(e.get)
		if err == nil || !strings.Contains(err.Error(), "DEV_LOGIN") {
			t.Fatalf("err = %v, want one naming DEV_LOGIN", err)
		}
	})
	t.Run("Cognito is needed", func(t *testing.T) {
		e := strict()
		delete(e, "STAFF_POOL_ID")
		delete(e, "STAFF_POOL_CLIENT_ID")
		delete(e, "STAFF_POOL_DOMAIN")
		_, err := config.Load(e.get)
		for _, name := range []string{"STAFF_POOL_ID", "STAFF_POOL_CLIENT_ID", "STAFF_POOL_DOMAIN"} {
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("err = %v, want one naming %s", err, name)
			}
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
		{"JOBS_TIMER", "off"},
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
	e := country()
	e["DEV_LOGIN"] = "true"
	c, err := config.Load(e.get)
	if err != nil {
		t.Fatal(err)
	}
	got := c.Public()
	want := config.Public{Locale: "en-CA", TimeZone: "America/Toronto",
		NationalID:  config.PublicNationalID{Type: "nhs", Label: "NHS number"},
		StaffSignIn: config.PublicSignIn{DevLogin: true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("public = %+v", got)
	}
}

// The browser needs Cognito's managed login addresses and the app client (S12): none of them
// is a secret (a public client, with PKCE). The redirect and sign-out addresses are the app's.
func TestPublicCognito(t *testing.T) {
	c, err := config.Load(strict().get)
	if err != nil {
		t.Fatal(err)
	}
	got := c.Public().StaffSignIn
	want := config.PublicSignIn{Cognito: &config.PublicCognito{
		AuthorizeURL: "https://poised-ca-qa.auth.ca-central-1.amazoncognito.com/oauth2/authorize",
		LogoutURL:    "https://poised-ca-qa.auth.ca-central-1.amazoncognito.com/logout",
		ClientID:     "1example23456789client",
		RedirectURI:  "https://ca.example.test/auth/callback",
		SignedOutURI: "https://ca.example.test/login",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("staff sign-in = %+v, want %+v", got, want)
	}
	if got.Cognito.RedirectURI != c.Cognito.RedirectURI {
		t.Errorf("the browser's redirect address %q isn't the one the code is exchanged with, %q", got.Cognito.RedirectURI, c.Cognito.RedirectURI)
	}
}

func TestLoadCognito(t *testing.T) {
	t.Run("locally it's optional: the dev login stands in", func(t *testing.T) {
		c, err := config.Load(required().get)
		if err != nil || c.Cognito.Enabled() {
			t.Fatalf("got %+v, %v", c.Cognito, err)
		}
	})
	t.Run("all three values", func(t *testing.T) {
		c, err := config.Load(cognito(country()).get)
		if err != nil {
			t.Fatal(err)
		}
		want := config.Cognito{UserPoolID: "ca-central-1_AbCdEf123", ClientID: "1example23456789client",
			Domain: "https://poised-ca-qa.auth.ca-central-1.amazoncognito.com", RedirectURI: "https://ca.example.test/auth/callback"}
		if c.Cognito != want || !c.Cognito.Enabled() {
			t.Errorf("cognito = %+v, want %+v", c.Cognito, want)
		}
		if c.Cognito.Issuer() != "https://cognito-idp.ca-central-1.amazonaws.com/ca-central-1_AbCdEf123" {
			t.Errorf("issuer = %q", c.Cognito.Issuer())
		}
		if c.Cognito.Region() != "ca-central-1" {
			t.Errorf("region = %q", c.Cognito.Region())
		}
	})
	t.Run("some but not all: the missing ones are named", func(t *testing.T) {
		e := cognito(country())
		delete(e, "STAFF_POOL_DOMAIN")
		_, err := config.Load(e.get)
		if err == nil || !strings.Contains(err.Error(), "STAFF_POOL_DOMAIN") {
			t.Fatalf("err = %v", err)
		}
	})
	for name, bad := range map[string]string{
		"STAFF_POOL_ID": "not-a-pool",
		"STAFF_POOL_DOMAIN":       "poised.auth.example.test",
		"STAFF_POOL_CLIENT_ID":    "has spaces",
	} {
		t.Run(name+" refused: "+bad, func(t *testing.T) {
			e := cognito(country())
			e[name] = bad
			_, err := config.Load(e.get)
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("err = %v, want one naming %s", err, name)
			}
		})
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
		e := strict()
		delete(e, "MESSAGING")
		_, err := config.Load(e.get)
		if err == nil || !strings.Contains(err.Error(), "MESSAGING") {
			t.Fatalf("got %v", err)
		}
		e["MESSAGING"] = "capture" // QA opts in
		if c, err := config.Load(e.get); err != nil || c.Messaging != "capture" {
			t.Fatalf("got %q, %v", c.Messaging, err)
		}
	})
	t.Run("MESSAGING is capture or send", func(t *testing.T) {
		e := required()
		e["MESSAGING"] = "aws"
		_, err := config.Load(e.get)
		if err == nil || !strings.Contains(err.Error(), "MESSAGING") {
			t.Fatalf("got %v", err)
		}
	})
}

// sending is the made-up country's real sending (S17): SES's sender and both configuration sets.
func sending(e env) env {
	e["EMAIL_FROM"] = "poised@ca.example.test"
	e["SES_CONFIGURATION_SET"] = "poised-ca-production"
	e["SMS_CONFIGURATION_SET"] = "poised-ca-production"
	return e
}

func TestLoadSending(t *testing.T) {
	t.Run("send: the sender and the configuration sets", func(t *testing.T) {
		e := sending(strict())
		e["MESSAGING"] = "send"
		e["SMS_SENDER_ID"] = "Poised"
		c, err := config.Load(e.get)
		if err != nil {
			t.Fatal(err)
		}
		want := config.Sending{EmailFrom: "poised@ca.example.test", SESConfigurationSet: "poised-ca-production",
			SMSConfigurationSet: "poised-ca-production", SMSSenderID: "Poised"}
		if c.Messaging != "send" || c.Sending != want || !c.Sending.Enabled() || len(c.TestRecipients) != 0 {
			t.Errorf("got %q %+v %v", c.Messaging, c.Sending, c.TestRecipients)
		}
	})
	t.Run("send without them is refused, naming each", func(t *testing.T) {
		e := strict()
		e["MESSAGING"] = "send"
		_, err := config.Load(e.get)
		for _, name := range []string{"EMAIL_FROM", "SES_CONFIGURATION_SET", "SMS_CONFIGURATION_SET"} {
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("missing %s: got %v", name, err)
			}
		}
	})
	t.Run("capture with test recipients (QA): captured, and really sent to them", func(t *testing.T) {
		e := sending(strict())
		e["MESSAGING_TEST_RECIPIENTS"] = " +447700900001 , Rahul@Example.test "
		c, err := config.Load(e.get)
		if err != nil {
			t.Fatal(err)
		}
		if c.Messaging != "capture" || !c.Sending.Enabled() || !reflect.DeepEqual(c.TestRecipients, []string{"+447700900001", "rahul@example.test"}) {
			t.Errorf("got %q %+v %q", c.Messaging, c.Sending, c.TestRecipients)
		}
	})
	t.Run("test recipients need the senders", func(t *testing.T) {
		e := strict()
		e["MESSAGING_TEST_RECIPIENTS"] = "+447700900001"
		if _, err := config.Load(e.get); err == nil || !strings.Contains(err.Error(), "EMAIL_FROM") {
			t.Errorf("got %v", err)
		}
	})
	t.Run("test recipients only with capture: with send, everyone gets messages anyway", func(t *testing.T) {
		e := sending(strict())
		e["MESSAGING"] = "send"
		e["MESSAGING_TEST_RECIPIENTS"] = "+447700900001"
		if _, err := config.Load(e.get); err == nil || !strings.Contains(err.Error(), "MESSAGING_TEST_RECIPIENTS") {
			t.Errorf("got %v", err)
		}
	})
	t.Run("capture alone sends nothing", func(t *testing.T) {
		c, err := config.Load(strict().get)
		if err != nil || c.Sending.Enabled() {
			t.Errorf("got %+v, %v", c.Sending, err)
		}
	})
	bad := map[string]string{
		"EMAIL_FROM":                "Poised <poised@ca.example.test>",
		"SMS_SENDER_ID":             "Poised Health Ltd",
		"MESSAGING_TEST_RECIPIENTS": "07700900001",
	}
	for name, v := range bad {
		t.Run("refuses a bad "+name, func(t *testing.T) {
			e := sending(strict())
			e["MESSAGING_TEST_RECIPIENTS"] = "+447700900001"
			e[name] = v
			if _, err := config.Load(e.get); err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("%s=%q: got %v", name, v, err)
			}
		})
	}
}

func TestLoadJobsSchedule(t *testing.T) {
	e := required()
	if c, err := config.Load(e.get); err != nil || c.JobsScheduleMinutes != 0 {
		t.Fatalf("local: %d, %v", c.JobsScheduleMinutes, err)
	}
	e["JOBS_SCHEDULE_MINUTES"] = "15"
	if c, err := config.Load(e.get); err != nil || c.JobsScheduleMinutes != 15 {
		t.Fatalf("got %d, %v", c.JobsScheduleMinutes, err)
	}
	for _, bad := range []string{"0", "-5", "every 5", "1441"} {
		e["JOBS_SCHEDULE_MINUTES"] = bad
		if _, err := config.Load(e.get); err == nil || !strings.Contains(err.Error(), "JOBS_SCHEDULE_MINUTES") {
			t.Errorf("%q: %v", bad, err)
		}
	}
}
