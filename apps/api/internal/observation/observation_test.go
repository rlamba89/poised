package observation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func f(v float64) *float64 { return &v }

var (
	now  = time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	born = time.Date(1970, 5, 1, 0, 0, 0, 0, time.UTC)
)

func TestKindsAndMeasures(t *testing.T) {
	// OBS-01's list: every measured kind has its measures, the notes have none, and every measure
	// has a default range.
	want := []string{"height", "weight", "bp", "heartRate", "spo2", "temperature", "respiratoryRate", "glucose", "peakFlow", "ecg", "urinalysis"}
	if len(Kinds) != len(want) {
		t.Fatalf("%d kinds, want %d", len(Kinds), len(want))
	}
	d := Defaults()
	for i, k := range Kinds {
		if k.Key != want[i] {
			t.Errorf("kind %d = %q, want %q", i, k.Key, want[i])
		}
		if k.Note != (len(k.Measures) == 0) {
			t.Errorf("%s: a note kind has no measures, and only it", k.Key)
		}
		for _, m := range k.Measures {
			if _, ok := MeasureOf(m); !ok {
				t.Errorf("%s: measure %q isn't defined", k.Key, m)
			}
			if _, ok := d[m]; !ok {
				t.Errorf("%s: no default range", m)
			}
		}
	}
	if bp, _ := KindOf("bp"); len(bp.Measures) != 2 || bp.Measures[0] != "systolic" || bp.Measures[1] != "diastolic" {
		t.Errorf("bp = %v, want systolic then diastolic", bp.Measures)
	}
	if msg := CheckRanges(d); msg != "" {
		t.Errorf("the defaults break the fixed rules: %s", msg)
	}
}

func TestCheckRanges(t *testing.T) {
	ok := Range{SoftMin: 50, SoftMax: 120, HardMin: 20, HardMax: 300}
	for _, c := range []struct {
		name string
		r    Ranges
		want string // "" or a part of the refusal
	}{
		{"a trust's own", Ranges{"heartRate": ok}, ""},
		{"only some measures", Ranges{"weight": {SoftMin: 40, SoftMax: 150, HardMin: 2, HardMax: 350}}, ""},
		{"unknown measure", Ranges{"pulse": ok}, "isn't a measurement"},
		{"negative", Ranges{"heartRate": {SoftMin: 50, SoftMax: 120, HardMin: -1, HardMax: 300}}, "can't be below 0"},
		{"soft below hard", Ranges{"heartRate": {SoftMin: 10, SoftMax: 120, HardMin: 20, HardMax: 300}}, "inside its hard limits"},
		{"soft above hard", Ranges{"heartRate": {SoftMin: 50, SoftMax: 320, HardMin: 20, HardMax: 300}}, "inside its hard limits"},
		{"soft upside down", Ranges{"heartRate": {SoftMin: 120, SoftMax: 50, HardMin: 20, HardMax: 300}}, "inside its hard limits"},
		{"SpO2 over 100", Ranges{"spo2": {SoftMin: 94, SoftMax: 100, HardMin: 50, HardMax: 101}}, "100"},
		{"a fraction on a whole-number measure", Ranges{"heartRate": {SoftMin: 50.5, SoftMax: 120, HardMin: 20, HardMax: 300}}, "whole number"},
		{"too many decimals", Ranges{"weight": {SoftMin: 35.25, SoftMax: 200, HardMin: 1, HardMax: 400}}, "decimal"},
	} {
		got := CheckRanges(c.r)
		if (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestEffective(t *testing.T) {
	mine := Range{SoftMin: 40, SoftMax: 150, HardMin: 2, HardMax: 350}
	e := Effective(Ranges{"weight": mine})
	if e["weight"] != mine {
		t.Errorf("weight = %+v, want the trust's", e["weight"])
	}
	if e["height"] != Defaults()["height"] || len(e) != len(Defaults()) {
		t.Errorf("the rest should be the platform's: %+v", e)
	}
	if len(Effective(nil)) != len(Defaults()) {
		t.Error("no trust value: the platform's")
	}
}

func TestCheckReading(t *testing.T) {
	rg := Defaults()
	hr := rg["heartRate"]
	for _, c := range []struct {
		name string
		in   Reading
		want string
	}{
		{"weight", Reading{Kind: "weight", Value: f(82.5), TakenAt: now}, ""},
		{"blood pressure", Reading{Kind: "bp", Value: f(128), Value2: f(82), TakenAt: now}, ""},
		{"an implausible heart rate is only a warning", Reading{Kind: "heartRate", Value: f(hr.SoftMax + 30), TakenAt: now}, ""},
		{"at the hard maximum", Reading{Kind: "heartRate", Value: f(hr.HardMax), TakenAt: now}, ""},
		{"an ECG note", Reading{Kind: "ecg", Note: "  Sinus rhythm  ", TakenAt: now}, ""},
		{"earlier today", Reading{Kind: "weight", Value: f(80), TakenAt: now.Add(-3 * time.Hour)}, ""},
		{"a few seconds ahead (clock skew)", Reading{Kind: "weight", Value: f(80), TakenAt: now.Add(time.Minute)}, ""},
		{"unknown kind", Reading{Kind: "pulse", Value: f(70), TakenAt: now}, "Choose what was measured"},
		{"negative", Reading{Kind: "weight", Value: f(-3), TakenAt: now}, "Weight"},
		{"above the hard maximum", Reading{Kind: "heartRate", Value: f(hr.HardMax + 1), TakenAt: now}, "Heart rate"},
		{"no value", Reading{Kind: "weight", TakenAt: now}, "Enter the weight"},
		{"bp without the diastolic", Reading{Kind: "bp", Value: f(128), TakenAt: now}, "Enter the diastolic"},
		{"diastolic above systolic", Reading{Kind: "bp", Value: f(80), Value2: f(120), TakenAt: now}, "lower than the systolic"},
		{"diastolic equal to systolic", Reading{Kind: "bp", Value: f(90), Value2: f(90), TakenAt: now}, "lower than the systolic"},
		{"a second value on a weight", Reading{Kind: "weight", Value: f(80), Value2: f(1), TakenAt: now}, "could not be read"},
		{"a note on a weight", Reading{Kind: "weight", Value: f(80), Note: "x", TakenAt: now}, "could not be read"},
		{"a value on an ECG", Reading{Kind: "ecg", Value: f(1), Note: "x", TakenAt: now}, "could not be read"},
		{"an empty ECG note", Reading{Kind: "ecg", Note: "   ", TakenAt: now}, "Enter the ECG"},
		{"a note too long", Reading{Kind: "urinalysis", Note: strings.Repeat("x", MaxNote+1), TakenAt: now}, "characters"},
		{"too many decimals", Reading{Kind: "weight", Value: f(82.25), TakenAt: now}, "decimal"},
		{"a fraction of a beat", Reading{Kind: "heartRate", Value: f(72.5), TakenAt: now}, "whole number"},
		{"in the future", Reading{Kind: "weight", Value: f(80), TakenAt: now.Add(time.Hour)}, "future"},
		{"no time", Reading{Kind: "weight", Value: f(80)}, "When was it measured"},
	} {
		got, msg := Check(c.in, rg, now, born)
		if (c.want == "") != (msg == "") || !strings.Contains(msg, c.want) {
			t.Errorf("%s: %q, want %q", c.name, msg, c.want)
			continue
		}
		if msg == "" && c.in.Kind == "ecg" && got.Note != "Sinus rhythm" {
			t.Errorf("%s: note %q, want it trimmed", c.name, got.Note)
		}
	}
	// A trust's own limits are the ones applied.
	tight := Effective(Ranges{"heartRate": {SoftMin: 50, SoftMax: 100, HardMin: 30, HardMax: 150}})
	if _, msg := Check(Reading{Kind: "heartRate", Value: f(160), TakenAt: now}, tight, now, born); !strings.Contains(msg, "150") {
		t.Errorf("trust's hard maximum: %q", msg)
	}
}

func TestMeasuredNoEarlierThanBirth(t *testing.T) {
	// S46 verification: a mistyped year (0026, 1926) would sit at the bottom of the list, and the
	// panel's Correct can't change a time. The date of birth is a date, so the limit is that day's
	// start anywhere: 14 hours ahead of UTC.
	for _, c := range []struct {
		name    string
		at      time.Time
		refused bool
	}{
		{"year 26", time.Date(26, 10, 30, 9, 0, 0, 0, time.UTC), true},
		{"before they were born", time.Date(1926, 10, 30, 9, 0, 0, 0, time.UTC), true},
		{"the day before, past any zone", born.Add(-15 * time.Hour), true},
		{"early on the day, in a zone ahead of UTC", born.Add(-13 * time.Hour), false},
		{"the day they were born", born.Add(9 * time.Hour), false},
	} {
		_, msg := Check(Reading{Kind: "weight", Value: f(80), TakenAt: c.at}, Defaults(), now, born)
		if c.refused != (msg == "The time it was measured can't be before the patient's date of birth.") {
			t.Errorf("%s: %q", c.name, msg)
		}
	}
}

func TestDisplayUnits(t *testing.T) {
	// A-20: each country's display units are config (GLUCOSE_UNIT, TEMPERATURE_UNIT). Nothing
	// converts yet, so only the stored units are accepted.
	if !DisplayUnitOK("glucose", "mmol/L") || !DisplayUnitOK("temperature", "°C") {
		t.Error("the stored units must be accepted")
	}
	if DisplayUnitOK("glucose", "mg/dL") || DisplayUnitOK("temperature", "°F") || DisplayUnitOK("weight", "kg") {
		t.Error("nothing converts yet, and only glucose and temperature are configurable")
	}
}

// The web app reads every kind's measures as a list: a note's is empty, never null (S46 test report 1, F-1).
func TestKindsJSONHasNoNullMeasures(t *testing.T) {
	b, err := json.Marshal(Kinds)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"measures":null`) {
		t.Errorf("kinds = %s", b)
	}
}
