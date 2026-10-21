package settings

import (
	"encoding/json"
	"strings"
	"testing"
)

func raw(s string) json.RawMessage { return json.RawMessage(s) }

func TestEveryKeyHasAValidDefault(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range Defs {
		if seen[d.Key] {
			t.Errorf("%s is defined twice", d.Key)
		}
		seen[d.Key] = true
		if d.Label == "" || d.Group == "" {
			t.Errorf("%s needs a label and a group", d.Key)
		}
		if _, msg := Validate(d.Key, d.Default); msg != "" {
			t.Errorf("%s: its default %s is refused: %s", d.Key, d.Default, msg)
		}
		if d.OptIn && d.Level != Hospital {
			t.Errorf("%s: opt-ins are set per hospital too (ORG-05)", d.Key)
		}
	}
	// ORG-04's list, by level.
	for _, k := range []string{"retentionYears", "crossTrustPrefill"} {
		if d, ok := Lookup(k); !ok || d.Level != TrustOnly {
			t.Errorf("%s should be trust-only", k)
		}
	}
	for _, k := range []string{"feature.assessments", "feature.proms", "feature.files", "feature.observations",
		"instrument.eq5d5l", "instrument.oxfordHip", "instrument.oxfordKnee", "branding.logoUrl", "contact.phone",
		"contact.email", "defaultHq", "reminders.days", "reminders.notRespondingDay", "autoArchive.afterDischargeDays",
		"autoArchive.inactiveMonths", "whatHappensNext", "templates.inviteSms", "templates.inviteEmail",
		"templates.reminderSms", "templates.reminderEmail",
		// S21: the worklist's days, and each status message's switch and template (answers 3, 4).
		"worklists.soonDays", "statusMessages.readyForAdmissionSms", "statusMessages.readyForAdmissionEmail",
		"statusMessages.cancelledSms", "statusMessages.cancelledEmail", "templates.readyForAdmissionSms",
		"templates.readyForAdmissionEmail", "templates.cancelledSms", "templates.cancelledEmail"} {
		if d, ok := Lookup(k); !ok || d.Level != Hospital {
			t.Errorf("%s should be a setting a hospital can override", k)
		}
	}
}

func TestEffective(t *testing.T) {
	platform := Platform()
	t.Run("nothing set: the platform defaults", func(t *testing.T) {
		got := Effective(platform, nil, nil)
		if string(got["retentionYears"]) != "8" || string(got["reminders.days"]) != "[2,5]" || string(got["feature.proms"]) != "false" {
			t.Errorf("got %v", got)
		}
		if len(got) != len(Defs) {
			t.Errorf("every key should have a value: got %d of %d", len(got), len(Defs))
		}
	})
	t.Run("the trust's value beats the default", func(t *testing.T) {
		got := Effective(platform, Values{"retentionYears": raw("10"), "contact.phone": raw(`"0100"`)}, nil)
		if string(got["retentionYears"]) != "10" || string(got["contact.phone"]) != `"0100"` {
			t.Errorf("got %v", got)
		}
	})
	t.Run("a hospital overrides only what it may", func(t *testing.T) {
		org := Values{"retentionYears": raw("10"), "contact.phone": raw(`"0100"`), "crossTrustPrefill": raw("true")}
		hosp := Values{"retentionYears": raw("2"), "crossTrustPrefill": raw("false"), "contact.phone": raw(`"0200"`)}
		got := Effective(platform, org, hosp)
		if string(got["retentionYears"]) != "10" || string(got["crossTrustPrefill"]) != "true" {
			t.Errorf("trust-only settings must come from the trust: %v", got)
		}
		if string(got["contact.phone"]) != `"0200"` {
			t.Errorf("the hospital's phone should win: %v", got)
		}
	})
	t.Run("a hospital that sets nothing follows the trust", func(t *testing.T) {
		got := Effective(platform, Values{"contact.email": raw(`"a@t.example"`)}, Values{})
		if string(got["contact.email"]) != `"a@t.example"` {
			t.Errorf("got %v", got)
		}
	})
	t.Run("opt-in inheritance (ORG-05): trust off with a hospital on, and the reverse", func(t *testing.T) {
		got := Effective(platform, Values{"feature.proms": raw("false")}, Values{"feature.proms": raw("true")})
		if string(got["feature.proms"]) != "true" {
			t.Errorf("hospital on should win over trust off: %v", got["feature.proms"])
		}
		got = Effective(platform, Values{"instrument.eq5d5l": raw("true")}, Values{"instrument.eq5d5l": raw("false")})
		if string(got["instrument.eq5d5l"]) != "false" {
			t.Errorf("hospital off should win over trust on: %v", got["instrument.eq5d5l"])
		}
		got = Effective(platform, Values{"instrument.eq5d5l": raw("true")}, Values{})
		if string(got["instrument.eq5d5l"]) != "true" {
			t.Errorf("a hospital that follows the trust should get its on: %v", got["instrument.eq5d5l"])
		}
	})
	t.Run("unknown keys are ignored", func(t *testing.T) {
		got := Effective(platform, Values{"nope": raw("1")}, Values{"nope2": raw("1")})
		if _, ok := got["nope"]; ok {
			t.Errorf("got %v", got)
		}
		if _, ok := got["nope2"]; ok {
			t.Errorf("got %v", got)
		}
	})
	t.Run("the platform defaults are a parameter", func(t *testing.T) {
		p := Platform()
		p["retentionYears"] = raw("9")
		if got := Effective(p, nil, nil); string(got["retentionYears"]) != "9" {
			t.Errorf("got %v", got["retentionYears"])
		}
	})
}

func TestValidate(t *testing.T) {
	ok := []struct{ key, in, want string }{
		{"retentionYears", "8", "8"},
		{"crossTrustPrefill", "true", "true"},
		{"feature.proms", "false", "false"},
		{"branding.logoUrl", `" https://cdn.example/logo.png "`, `"https://cdn.example/logo.png"`},
		{"branding.logoUrl", `""`, `""`},
		{"contact.phone", `" 0100 000 0000 "`, `"0100 000 0000"`},
		{"contact.email", `" POA@Hospital.Example "`, `"poa@hospital.example"`},
		{"contact.email", `""`, `""`},
		{"defaultHq", `""`, `""`},
		{"defaultHq", `"6f1c2c1e-1111-4a6b-9c1d-000000000001"`, `"6f1c2c1e-1111-4a6b-9c1d-000000000001"`},
		{"reminders.days", "[5, 2]", "[2,5]"},
		{"reminders.days", "[]", "[]"},
		{"reminders.notRespondingDay", "7", "7"},
		{"autoArchive.afterDischargeDays", "30", "30"},
		{"autoArchive.inactiveMonths", "6", "6"},
		{"whatHappensNext", `"  Thanks.  "`, `"Thanks."`},
		{"templates.inviteSms", `"Hi {{firstName}}, {{hospital}}: {{link}}"`, `"Hi {{firstName}}, {{hospital}}: {{link}}"`},
		// S21: a status message needn't hold {{link}} (answers 4), but may.
		{"worklists.soonDays", "14", "14"},
		{"statusMessages.cancelledSms", "true", "true"},
		{"templates.readyForAdmissionSms", `" Hi {{firstName}}, {{hospital}} is ready for you. "`, `"Hi {{firstName}}, {{hospital}} is ready for you."`},
		{"templates.cancelledEmail", `"Sign in: {{link}}"`, `"Sign in: {{link}}"`},
	}
	for _, c := range ok {
		got, msg := Validate(c.key, raw(c.in))
		if msg != "" || string(got) != c.want {
			t.Errorf("Validate(%s, %s) = %s, %q; want %s", c.key, c.in, got, msg, c.want)
		}
	}
	bad := []struct{ key, in string }{
		{"nope", "1"},
		{"retentionYears", "0"},
		{"retentionYears", "101"},
		{"retentionYears", "2.5"},
		{"retentionYears", `"8"`},
		{"crossTrustPrefill", `"yes"`},
		{"feature.proms", "1"},
		{"branding.logoUrl", `"javascript:alert(1)"`},
		{"branding.logoUrl", `"logo.png"`},
		{"branding.logoUrl", `"https://` + strings.Repeat("a", 2000) + `"`},
		{"contact.phone", `"call us"`},
		{"contact.phone", "123"},
		{"contact.email", `"not-an-email"`},
		{"defaultHq", `"not-a-uuid"`},
		{"reminders.days", "[0]"},
		{"reminders.days", "[61]"},
		{"reminders.days", "[2, 2]"},
		{"reminders.days", "[1,2,3,4,5,6]"},
		{"reminders.days", `"2,5"`},
		{"reminders.notRespondingDay", "0"},
		{"autoArchive.afterDischargeDays", "366"},
		{"autoArchive.inactiveMonths", "0"},
		{"whatHappensNext", `"` + strings.Repeat("a", 2001) + `"`},
		{"whatHappensNext", "null"},
		{"templates.inviteSms", `"Hi {{firstName}}"`},            // no link
		{"templates.inviteSms", `"Hi {{name}} {{link}}"`},        // unknown placeholder
		{"templates.reminderEmail", `""`},                        // empty
		{"templates.reminderEmail", `"{{link}} {{ hospital }}"`}, // a placeholder written oddly
		{"templates.reminderSms", `"Hi {{firstName}}"`},          // invites and reminders still need {{link}}
		{"templates.inviteEmail", `"Hi {{firstName}}"`},
		{"templates.readyForAdmissionSms", `""`},    // empty
		{"templates.cancelledSms", `"Hi {{name}}"`}, // unknown placeholder
		{"templates.cancelledSms", `"` + strings.Repeat("a", 601) + `"`},
		{"autoArchive.afterDischargeDays", "7.5"}, // S22 test report 2: a decimal is refused, not rounded
		{"worklists.soonDays", "0"},
		{"worklists.soonDays", "366"},
		{"statusMessages.readyForAdmissionEmail", `"on"`},
	}
	for _, c := range bad {
		if got, msg := Validate(c.key, raw(c.in)); msg == "" {
			t.Errorf("Validate(%s, %s) = %s; want a refusal", c.key, c.in, got)
		}
	}
}

// Each status message is off by default (S21 answers 4), and its help says {{link}} is optional.
func TestStatusMessagesAreOffByDefault(t *testing.T) {
	for _, stem := range []string{"readyForAdmission", "cancelled"} {
		for _, ch := range []string{"Sms", "Email"} {
			sw, ok := Lookup("statusMessages." + stem + ch)
			if !ok || sw.Kind != Bool || string(sw.Default) != "false" {
				t.Errorf("statusMessages.%s%s = %+v, want a switch, off by default", stem, ch, sw)
			}
			tmpl, ok := Lookup("templates." + stem + ch)
			if !ok || tmpl.Kind != Template || !strings.Contains(tmpl.Help, "optional") || tmpl.Group != sw.Group {
				t.Errorf("templates.%s%s = %+v, want a template beside its switch, with {{link}} optional", stem, ch, tmpl)
			}
		}
	}
	if d, _ := Lookup("worklists.soonDays"); string(d.Default) != "14" {
		t.Errorf("worklists.soonDays defaults to %s, want 14 (S21 answers 3)", d.Default)
	}
}

func TestOptInKeys(t *testing.T) {
	for _, d := range Defs {
		want := strings.HasPrefix(d.Key, "feature.") || strings.HasPrefix(d.Key, "instrument.")
		if d.OptIn != want {
			t.Errorf("%s: OptIn = %v", d.Key, d.OptIn)
		}
	}
	if !IsOptIn("feature.proms") || IsOptIn("contact.phone") || IsOptIn("nope") {
		t.Error("IsOptIn is wrong")
	}
}

func TestDecodeAndApply(t *testing.T) {
	v, err := Decode(raw(`{"reminders.days": [1, 5], "nope": 1, "contact.phone": "0100"}`))
	if err != nil || len(v) != 2 || string(v["reminders.days"]) != "[1,5]" || string(v["contact.phone"]) != `"0100"` {
		t.Fatalf("Decode = %v, %v", v, err)
	}
	if _, err := Decode(raw(`[]`)); err == nil {
		t.Error("an array isn't settings")
	}
	got := Apply(v, map[string]json.RawMessage{"contact.phone": nil, "retentionYears": raw("9")})
	if len(got) != 2 || string(got["retentionYears"]) != "9" || got["contact.phone"] != nil {
		t.Errorf("Apply = %v", got)
	}
	if len(v) != 2 || string(v["contact.phone"]) != `"0100"` {
		t.Error("Apply changed what it was given")
	}
}

func TestFill(t *testing.T) {
	vars := map[string]string{"firstName": "Sam", "hospital": "St Mary's Hospital", "link": "https://app.example/i/tok"}
	t.Run("the default invite text follows PAT-03's wording", func(t *testing.T) {
		v := Effective(Platform(), nil, nil)
		got, err := Fill(v, "templates.inviteSms", vars)
		want := "St Mary's Hospital: please complete your pre-op health questionnaire: https://app.example/i/tok"
		if err != nil || got != want {
			t.Fatalf("got %q, %v; want %q", got, err, want)
		}
	})
	t.Run("a hospital's own template, every placeholder filled once each", func(t *testing.T) {
		v := Values{"templates.inviteEmail": raw(`"Hi {{firstName}} from {{hospital}}: {{link}} ({{link}})"`)}
		got, err := Fill(v, "templates.inviteEmail", vars)
		if err != nil || got != "Hi Sam from St Mary's Hospital: https://app.example/i/tok (https://app.example/i/tok)" {
			t.Fatalf("got %q, %v", got, err)
		}
	})
	t.Run("a value can't add a placeholder of its own", func(t *testing.T) {
		v := Values{"templates.inviteSms": raw(`"{{firstName}}: {{link}}"`)}
		got, _ := Fill(v, "templates.inviteSms", map[string]string{"firstName": "{{link}}", "link": "L"})
		if got != "{{link}}: L" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("a missing or broken template is an error, not an empty message", func(t *testing.T) {
		if _, err := Fill(Values{}, "templates.inviteSms", vars); err == nil {
			t.Error("missing: no error")
		}
		if _, err := Fill(Values{"templates.inviteSms": raw(`42`)}, "templates.inviteSms", vars); err == nil {
			t.Error("not a string: no error")
		}
	})
}

func TestGet(t *testing.T) {
	v := Platform()
	var days []int
	if err := Get(v, "reminders.days", &days); err != nil || len(days) != 2 || days[0] != 2 || days[1] != 5 {
		t.Errorf("reminders.days = %v, %v", days, err)
	}
	var n int
	if err := Get(v, "reminders.notRespondingDay", &n); err != nil || n != 7 {
		t.Errorf("reminders.notRespondingDay = %v, %v", n, err)
	}
	if err := Get(Values{"reminders.days": raw(`"2,5"`)}, "reminders.days", &days); err == nil {
		t.Error("a string read as a day list")
	}
	if err := Get(Values{}, "reminders.days", &days); err == nil {
		t.Error("a missing value read")
	}
}
