package episode

import (
	"fmt"
	"time"
)

// The daily auto-archive (EPI-05, S22 answers 3 and 4). A job per hospital runs at 02:00 there
// (internal/jobs), reads its episodes and archives each one ArchiveDue says is due. "After the
// last PROMs form closes" waits for C9.

// ArchiveFacts is what ArchiveDue needs to know about one episode.
type ArchiveFacts struct {
	Status       string
	DischargedAt time.Time // the discharge date, as a date (midnight UTC); zero: none
	LastActivity time.Time // the newest write to the episode (episodes.last_activity_at)
	UnarchivedAt time.Time // the latest un-archive; zero: never un-archived
}

// ArchivePolicy is the hospital's effective settings (its own, else its trust's, else the
// platform's): autoArchive.afterDischargeDays (N) and autoArchive.inactiveMonths (M), counted in
// its time zone.
type ArchivePolicy struct {
	DaysAfterDischarge int
	InactiveMonths     int
	Zone               *time.Location // nil: UTC
}

// preOp are the statuses before admission, which the "no activity" rule archives; and a
// cancelled episode, which nothing else would ever archive (the lifecycle's "cancelled →
// archived", saas-requirements §5.1).
var preOp = map[string]bool{HQNotComplete: true, ReadyForReview: true, ReadyForPOA: true, POAComplete: true, OnHold: true,
	NotReady: true, ReadyForAdmission: true, Cancelled: true}

// ArchiveDue says whether the episode is due to be archived at now, and by which rule, as the
// status history names it ("Auto-archive: 30 days after discharge"). It is pure: the job reads
// the episode, asks, then archives.
//
//   - Discharged: N days after the discharge date, at the hospital (day N is due). An episode
//     un-archived since counts its N days from the un-archive, so it isn't archived again the
//     next night.
//   - Before admission, or cancelled: M months after the last write to it (answers, status,
//     notes, tasks, messages sent; an un-archive is one too).
//   - Admitted, archived, or any other status: never.
func ArchiveDue(f ArchiveFacts, p ArchivePolicy, now time.Time) (string, bool) {
	zone := p.Zone
	if zone == nil {
		zone = time.UTC
	}
	switch {
	case f.Status == Discharged && !f.DischargedAt.IsZero() && p.DaysAfterDischarge > 0:
		y, m, d := now.In(zone).Date()
		today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		from := f.DischargedAt
		if !f.UnarchivedAt.IsZero() {
			uy, um, ud := f.UnarchivedAt.In(zone).Date()
			if u := time.Date(uy, um, ud, 0, 0, 0, 0, time.UTC); u.After(from) {
				from = u
			}
		}
		if !today.Before(from.AddDate(0, 0, p.DaysAfterDischarge)) {
			return "Auto-archive: " + plural(p.DaysAfterDischarge, "day") + " after discharge", true
		}
	case preOp[f.Status] && p.InactiveMonths > 0:
		if !now.Before(f.LastActivity.In(zone).AddDate(0, p.InactiveMonths, 0)) {
			return "Auto-archive: " + plural(p.InactiveMonths, "month") + " without activity", true
		}
	}
	return "", false
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// ArchiveHour is when the auto-archive runs at each hospital: 02:00 there (S22 answers 4).
const ArchiveHour = 2

// NextArchiveRun is the hospital's next auto-archive after now: 02:00 in its zone (nil: UTC),
// today if that's still to come, else tomorrow.
func NextArchiveRun(now time.Time, zone *time.Location) time.Time {
	if zone == nil {
		zone = time.UTC
	}
	local := now.In(zone)
	y, m, d := local.Date()
	next := time.Date(y, m, d, ArchiveHour, 0, 0, 0, zone)
	if !next.After(now) {
		next = time.Date(y, m, d+1, ArchiveHour, 0, 0, 0, zone)
	}
	return next
}
