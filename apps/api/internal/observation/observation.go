// Package observation holds the rules for a clinician's readings (plan C7, session S46; OBS-01–03):
// what can be measured, in which unit it's stored, the platform's plausible ranges and hard
// limits, and the checks a reading must pass before it's saved.
//
//   - A kind is what's recorded (Kinds, OBS-01): most have one measure (a weight), blood pressure
//     has two (systolic, then diastolic), and ECG and urinalysis are notes (text, no measure).
//     BMI isn't a kind: the web app works it out from the latest height and weight
//     (packages/clinical, the same bmi() as the HQ's).
//   - Each measure is stored in one unit (Measure.Unit). A country shows glucose and temperature in
//     the units its config names (GLUCOSE_UNIT, TEMPERATURE_UNIT; A-20); nothing converts yet, so
//     those must be the stored units (DisplayUnitOK).
//   - Each measure has a soft range (outside it the web app warns, and the reading can still be
//     saved) and hard limits (refused). Defaults are the platform's; a trust's whole-trust admins
//     may set their own (the trust setting observations.ranges, S46 answers 1), within the fixed
//     rules CheckRanges applies whatever a trust sets.
//
// The default ranges are made up for testing until a clinician checks them (before go-live, C10).
package observation

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// Measure is one number a reading holds.
type Measure struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Unit     string `json:"unit"`     // the stored unit
	Decimals int    `json:"decimals"` // the most decimal places a value may have
}

// Kind is one thing a clinician records.
type Kind struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Measures []string `json:"measures"` // none for a note
	Note     bool     `json:"note"`
}

// Measures are every measure, in the order screens show them.
var Measures = []Measure{
	{"height", "Height", "cm", 1},
	{"weight", "Weight", "kg", 1},
	{"systolic", "Systolic", "mmHg", 0},
	{"diastolic", "Diastolic", "mmHg", 0},
	{"heartRate", "Heart rate", "beats/min", 0},
	{"spo2", "SpO₂", "%", 0},
	{"temperature", "Temperature", "°C", 1},
	{"respiratoryRate", "Respiratory rate", "breaths/min", 0},
	{"glucose", "Blood glucose", "mmol/L", 1},
	{"peakFlow", "Peak flow", "L/min", 0},
}

// Kinds are what a clinician records (OBS-01), in the order the panel offers them.
var Kinds = []Kind{
	{Key: "height", Label: "Height", Measures: []string{"height"}},
	{Key: "weight", Label: "Weight", Measures: []string{"weight"}},
	{Key: "bp", Label: "Blood pressure", Measures: []string{"systolic", "diastolic"}},
	{Key: "heartRate", Label: "Heart rate", Measures: []string{"heartRate"}},
	{Key: "spo2", Label: "SpO₂", Measures: []string{"spo2"}},
	{Key: "temperature", Label: "Temperature", Measures: []string{"temperature"}},
	{Key: "respiratoryRate", Label: "Respiratory rate", Measures: []string{"respiratoryRate"}},
	{Key: "glucose", Label: "Blood glucose", Measures: []string{"glucose"}},
	{Key: "peakFlow", Label: "Peak flow", Measures: []string{"peakFlow"}},
	{Key: "ecg", Label: "ECG", Measures: []string{}, Note: true}, // [], never null: the web app reads a list
	{Key: "urinalysis", Label: "Urinalysis", Measures: []string{}, Note: true},
}

// MaxNote is the most characters an ECG or urinalysis note may have.
const MaxNote = 2000

// Range is one measure's soft range (a warning) and hard limits (refused), in its stored unit.
type Range struct {
	SoftMin float64 `json:"softMin"`
	SoftMax float64 `json:"softMax"`
	HardMin float64 `json:"hardMin"`
	HardMax float64 `json:"hardMax"`
}

// Ranges are ranges by measure key.
type Ranges map[string]Range

// defaults are the platform's ranges. Made up for testing: a clinician checks them before go-live.
var defaults = Ranges{
	"height":          {SoftMin: 120, SoftMax: 210, HardMin: 30, HardMax: 250},
	"weight":          {SoftMin: 35, SoftMax: 200, HardMin: 1, HardMax: 400},
	"systolic":        {SoftMin: 90, SoftMax: 180, HardMin: 40, HardMax: 300},
	"diastolic":       {SoftMin: 50, SoftMax: 110, HardMin: 20, HardMax: 200},
	"heartRate":       {SoftMin: 50, SoftMax: 120, HardMin: 20, HardMax: 300},
	"spo2":            {SoftMin: 94, SoftMax: 100, HardMin: 50, HardMax: 100},
	"temperature":     {SoftMin: 35.5, SoftMax: 38, HardMin: 25, HardMax: 45},
	"respiratoryRate": {SoftMin: 10, SoftMax: 24, HardMin: 4, HardMax: 80},
	"glucose":         {SoftMin: 4, SoftMax: 11, HardMin: 0.5, HardMax: 50},
	"peakFlow":        {SoftMin: 200, SoftMax: 700, HardMin: 50, HardMax: 1000},
}

// Defaults is a copy of the platform's ranges.
func Defaults() Ranges {
	out := Ranges{}
	for k, v := range defaults {
		out[k] = v
	}
	return out
}

// Effective is the ranges that apply: the trust's own for the measures it set, else the platform's.
func Effective(trust Ranges) Ranges {
	out := Defaults()
	for k, v := range trust {
		if _, ok := out[k]; ok {
			out[k] = v
		}
	}
	return out
}

// MeasureOf finds a measure by key.
func MeasureOf(key string) (Measure, bool) {
	i := slices.IndexFunc(Measures, func(m Measure) bool { return m.Key == key })
	if i < 0 {
		return Measure{}, false
	}
	return Measures[i], true
}

// KindOf finds a kind by key.
func KindOf(key string) (Kind, bool) {
	i := slices.IndexFunc(Kinds, func(k Kind) bool { return k.Key == key })
	if i < 0 {
		return Kind{}, false
	}
	return Kinds[i], true
}

// CheckRanges applies the rules no trust can change to a trust's ranges (S46 answers 1): only known
// measures; no limit below 0; the soft range in order and inside the hard limits; SpO₂ at most
// 100; no more decimals than the measure takes. "" means they're fine.
func CheckRanges(r Ranges) string {
	for _, k := range slices.Sorted(maps.Keys(r)) {
		m, ok := MeasureOf(k)
		if !ok {
			return fmt.Sprintf("%q isn't a measurement.", k)
		}
		v := r[k]
		for _, n := range []float64{v.SoftMin, v.SoftMax, v.HardMin, v.HardMax} {
			switch {
			case math.IsNaN(n) || math.IsInf(n, 0):
				return m.Label + ": enter numbers."
			case n < 0:
				return m.Label + ": a limit can't be below 0."
			case !decimalsOK(n, m.Decimals):
				return m.Label + ": " + decimalsText(m.Decimals) + "."
			}
		}
		if !(v.HardMin <= v.SoftMin && v.SoftMin <= v.SoftMax && v.SoftMax <= v.HardMax) {
			return m.Label + ": the usual range must be inside its hard limits, lowest first."
		}
		if m.Key == "spo2" && v.HardMax > 100 {
			return m.Label + ": a limit can't be above 100%."
		}
	}
	return ""
}

// Reading is what a clinician enters: Value (and Value2, the diastolic) for a measured kind, Note
// for ECG and urinalysis.
type Reading struct {
	Kind    string
	Value   *float64
	Value2  *float64
	Note    string
	TakenAt time.Time
}

// skew is how far ahead of the server's clock a reading's time may be (a browser's clock).
const skew = 5 * time.Minute

// aheadOfUTC is the most any time zone is ahead of UTC. A date of birth is a date, so that day
// starts this long before its midnight in UTC at the earliest, wherever the patient was measured.
const aheadOfUTC = 14 * time.Hour

// Check refuses a reading outside the hard limits in rg (the effective ranges), or otherwise
// malformed, with a plain reason; it returns the reading as stored (a note trimmed). born is the
// patient's date of birth, the earliest a reading can be: the panel can't change a saved time, so
// a mistyped year would otherwise stay (S46 verification).
func Check(rd Reading, rg Ranges, now, born time.Time) (Reading, string) {
	k, ok := KindOf(rd.Kind)
	if !ok {
		return rd, "Choose what was measured."
	}
	switch {
	case rd.TakenAt.IsZero():
		return rd, "When was it measured? Enter the date and time."
	case rd.TakenAt.After(now.Add(skew)):
		return rd, "The time it was measured can't be in the future."
	case rd.TakenAt.Before(born.Add(-aheadOfUTC)):
		return rd, "The time it was measured can't be before the patient's date of birth."
	}
	if k.Note {
		if rd.Value != nil || rd.Value2 != nil {
			return rd, "The request could not be read."
		}
		rd.Note = strings.TrimSpace(rd.Note)
		switch {
		case rd.Note == "":
			return rd, "Enter the " + k.Label + " note."
		case utf8.RuneCountInString(rd.Note) > MaxNote:
			return rd, fmt.Sprintf("The note is too long (%d characters at most).", MaxNote)
		}
		return rd, ""
	}
	if rd.Note != "" || (len(k.Measures) == 1 && rd.Value2 != nil) {
		return rd, "The request could not be read."
	}
	values := []*float64{rd.Value, rd.Value2}
	for i, key := range k.Measures {
		m, _ := MeasureOf(key)
		v := values[i]
		if v == nil {
			return rd, "Enter the " + strings.ToLower(m.Label) + "."
		}
		if msg := within(m, *v, rg[key]); msg != "" {
			return rd, msg
		}
	}
	if len(k.Measures) == 2 && *rd.Value2 >= *rd.Value {
		return rd, "The diastolic must be lower than the systolic."
	}
	return rd, ""
}

// within checks one value against its measure's hard limits and decimals.
func within(m Measure, v float64, r Range) string {
	switch {
	case math.IsNaN(v) || math.IsInf(v, 0):
		return m.Label + ": enter a number."
	case !decimalsOK(v, m.Decimals):
		return m.Label + ": " + decimalsText(m.Decimals) + "."
	case v < r.HardMin || v > r.HardMax || v < 0:
		return fmt.Sprintf("%s must be from %s to %s %s.", m.Label, format(r.HardMin), format(r.HardMax), m.Unit)
	}
	return ""
}

func decimalsOK(v float64, decimals int) bool {
	scaled := v * math.Pow10(decimals)
	return math.Abs(scaled-math.Round(scaled)) < 1e-6
}

func decimalsText(decimals int) string {
	if decimals == 0 {
		return "enter a whole number"
	}
	return fmt.Sprintf("use at most %d decimal place", decimals)
}

func format(v float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
}

// displayUnits are the units a country's config may name for each configurable measure (A-20).
// Only the stored units for now: a second country's units come with their conversion.
var displayUnits = map[string][]string{
	"glucose":     {"mmol/L"},
	"temperature": {"°C"},
}

// DisplayUnitOK reports whether a country may show measure in unit.
func DisplayUnitOK(measure, unit string) bool {
	return slices.Contains(displayUnits[measure], unit)
}
