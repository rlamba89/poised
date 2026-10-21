package episode

import (
	"testing"
	"time"
)

// ArchiveDue is the daily auto-archive's decision for one episode (EPI-05, S22 answers 3 and 4):
// N days after discharge, and M months without activity for a pre-op episode, with the hospital's
// own N and M; written test-first.
func TestArchiveDue(t *testing.T) {
	utc := time.UTC
	kiritimati, err := time.LoadLocation("Pacific/Kiritimati") // UTC+14: its date runs ahead of UTC's
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 15, 2, 0, 0, 0, utc) // the run: 02:00 at the hospital
	date := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, utc) }
	defaults := ArchivePolicy{DaysAfterDischarge: 30, InactiveMonths: 6, Zone: utc}
	sevenDays := ArchivePolicy{DaysAfterDischarge: 7, InactiveMonths: 2, Zone: utc} // a hospital's override of both
	recent := now.AddDate(0, 0, -7)                                                 // a note added last week

	for _, tt := range []struct {
		name   string
		facts  ArchiveFacts
		policy ArchivePolicy
		at     time.Time
		want   string // the rule; "" not due
	}{
		// N days after discharge.
		{"discharged 30 days ago", ArchiveFacts{Status: Discharged, DischargedAt: date(2026, 5, 16), LastActivity: recent},
			defaults, now, "Auto-archive: 30 days after discharge"},
		{"discharged 31 days ago", ArchiveFacts{Status: Discharged, DischargedAt: date(2026, 5, 15), LastActivity: recent},
			defaults, now, "Auto-archive: 30 days after discharge"},
		{"just not due: discharged 29 days ago", ArchiveFacts{Status: Discharged, DischargedAt: date(2026, 5, 17), LastActivity: recent},
			defaults, now, ""},
		{"the hospital's 7 days: 8 days after", ArchiveFacts{Status: Discharged, DischargedAt: date(2026, 6, 7), LastActivity: recent},
			sevenDays, now, "Auto-archive: 7 days after discharge"},
		{"the hospital's 7 days: 6 days after", ArchiveFacts{Status: Discharged, DischargedAt: date(2026, 6, 9), LastActivity: recent},
			sevenDays, now, ""},
		{"1 day: singular", ArchiveFacts{Status: Discharged, DischargedAt: date(2026, 6, 14)},
			ArchivePolicy{DaysAfterDischarge: 1, InactiveMonths: 6, Zone: utc}, now, "Auto-archive: 1 day after discharge"},
		{"the hospital's date, not UTC's: due there", ArchiveFacts{Status: Discharged, DischargedAt: date(2026, 5, 16)},
			ArchivePolicy{DaysAfterDischarge: 30, InactiveMonths: 6, Zone: kiritimati}, date(2026, 6, 14).Add(11 * time.Hour),
			"Auto-archive: 30 days after discharge"},
		{"…and not yet in UTC", ArchiveFacts{Status: Discharged, DischargedAt: date(2026, 5, 16)},
			defaults, date(2026, 6, 14).Add(11 * time.Hour), ""},
		{"un-archived since: the days count from the un-archive", ArchiveFacts{Status: Discharged, DischargedAt: date(2026, 1, 1),
			UnarchivedAt: now.AddDate(0, 0, -5)}, defaults, now, ""},
		{"un-archived 30 days ago", ArchiveFacts{Status: Discharged, DischargedAt: date(2026, 1, 1),
			UnarchivedAt: now.AddDate(0, 0, -30)}, defaults, now, "Auto-archive: 30 days after discharge"},
		{"discharged with no date", ArchiveFacts{Status: Discharged, LastActivity: now.AddDate(-2, 0, 0)}, defaults, now, ""},
		{"a discharge long ago doesn't count once admitted again", ArchiveFacts{Status: Admitted, DischargedAt: date(2025, 1, 1),
			LastActivity: now.AddDate(-2, 0, 0)}, defaults, now, ""},

		// M months without activity, for pre-op episodes.
		{"untouched for 6 months", ArchiveFacts{Status: OnHold, LastActivity: now.AddDate(0, -6, 0)},
			defaults, now, "Auto-archive: 6 months without activity"},
		{"just not due: 6 months less a minute", ArchiveFacts{Status: OnHold, LastActivity: now.AddDate(0, -6, 0).Add(time.Minute)},
			defaults, now, ""},
		{"a note added last week", ArchiveFacts{Status: HQNotComplete, LastActivity: recent}, defaults, now, ""},
		{"the hospital's 2 months", ArchiveFacts{Status: ReadyForAdmission, LastActivity: now.AddDate(0, -2, 0)},
			sevenDays, now, "Auto-archive: 2 months without activity"},
		{"the hospital's 2 months: not yet", ArchiveFacts{Status: ReadyForAdmission, LastActivity: now.AddDate(0, -2, 1)},
			sevenDays, now, ""},
		{"1 month: singular", ArchiveFacts{Status: ReadyForReview, LastActivity: now.AddDate(0, -1, 0)},
			ArchivePolicy{DaysAfterDischarge: 30, InactiveMonths: 1, Zone: utc}, now, "Auto-archive: 1 month without activity"},
		{"every pre-op status", ArchiveFacts{Status: ReadyForPOA, LastActivity: now.AddDate(-1, 0, 0)},
			defaults, now, "Auto-archive: 6 months without activity"},
		{"a cancelled episode, untouched", ArchiveFacts{Status: Cancelled, LastActivity: now.AddDate(0, -6, 0)},
			defaults, now, "Auto-archive: 6 months without activity"},
		{"a cancelled episode, touched", ArchiveFacts{Status: Cancelled, LastActivity: recent}, defaults, now, ""},

		// Never.
		{"admitted: neither rule", ArchiveFacts{Status: Admitted, LastActivity: now.AddDate(-1, 0, 0)}, defaults, now, ""},
		{"discharged recently, untouched long ago: only the discharge rule", ArchiveFacts{Status: Discharged,
			DischargedAt: date(2026, 6, 1), LastActivity: now.AddDate(-1, 0, 0)}, defaults, now, ""},
		{"already archived", ArchiveFacts{Status: Archived, DischargedAt: date(2025, 1, 1), LastActivity: now.AddDate(-2, 0, 0)},
			defaults, now, ""},
		{"an unknown status", ArchiveFacts{Status: "recovery", LastActivity: now.AddDate(-2, 0, 0)}, defaults, now, ""},
		{"no settings: nothing is archived", ArchiveFacts{Status: OnHold, LastActivity: now.AddDate(-2, 0, 0)}, ArchivePolicy{}, now, ""},
	} {
		got, due := ArchiveDue(tt.facts, tt.policy, tt.at)
		if got != tt.want || due != (tt.want != "") {
			t.Errorf("%s: ArchiveDue = %q, %v; want %q", tt.name, got, due, tt.want)
		}
	}
}

// NextArchiveRun is the auto-archive job's next run: 02:00 at the hospital, after now.
func TestNextArchiveRun(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		now  time.Time
		zone *time.Location
		want time.Time
	}{
		{time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC), time.UTC, time.Date(2026, 6, 15, 2, 0, 0, 0, time.UTC)},
		{time.Date(2026, 6, 15, 2, 0, 0, 0, time.UTC), time.UTC, time.Date(2026, 6, 16, 2, 0, 0, 0, time.UTC)}, // strictly after
		{time.Date(2026, 6, 15, 23, 0, 0, 0, time.UTC), time.UTC, time.Date(2026, 6, 16, 2, 0, 0, 0, time.UTC)},
		// Summer time: 02:00 in London is 01:00 UTC.
		{time.Date(2026, 6, 15, 0, 30, 0, 0, time.UTC), london, time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC)},
		{time.Date(2026, 6, 15, 1, 0, 0, 0, time.UTC), london, time.Date(2026, 6, 16, 1, 0, 0, 0, time.UTC)},
		{time.Date(2026, 1, 15, 1, 30, 0, 0, time.UTC), london, time.Date(2026, 1, 15, 2, 0, 0, 0, time.UTC)},
		{time.Date(2026, 1, 15, 1, 0, 0, 0, time.UTC), nil, time.Date(2026, 1, 15, 2, 0, 0, 0, time.UTC)}, // nil: UTC
	} {
		if got := NextArchiveRun(tt.now, tt.zone); !got.Equal(tt.want) {
			t.Errorf("NextArchiveRun(%s, %v) = %s, want %s", tt.now, tt.zone, got.UTC(), tt.want)
		}
	}
}
