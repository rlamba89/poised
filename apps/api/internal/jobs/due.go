package jobs

import "time"

// Followup is what Due needs to know about one episode's follow-up.
type Followup struct {
	InvitedAt     time.Time // the first invite that reached the patient; zero: none has
	Submitted     bool      // the HQ has been sent
	Started       bool      // the patient (or staff for them, PAT-09) has answered something
	Open          bool      // its trust and hospital are active (not suspended, ORG-03)
	RemindersSent int       // how many of the reminder days have been dealt with
	NotResponding bool      // already marked "not responding"
	Locked        bool      // five wrong dates of birth locked the episode's links until staff resend (S14)
}

// Schedule is the hospital's follow-up settings (ORG-04): reminders.days (in order) and
// reminders.notRespondingDay, counted in the hospital's time zone.
type Schedule struct {
	ReminderDays     []int
	NotRespondingDay int
	Zone             *time.Location // nil: UTC
}

// Actions is what to do now, and when to look again.
type Actions struct {
	Reminder      int       // the reminder for this day after the invite (e.g. 2) is due now; 0: none
	Withheld      bool      // …but isn't sent: the episode is locked. It's dealt with (counted), not sent later
	ReminderCount int       // the episode's reminders_sent once this is done
	NotResponding bool      // mark the episode "not responding" now
	Next          time.Time // when to look again (always after now); zero: the follow-up is finished
}

// Due decides an episode's follow-up at now (S16). It is pure: the runner reads the episode,
// asks Due, then does what it says.
//
//   - Day n is the invite's local time n calendar days later in the hospital's zone, so a
//     reminder keeps its time of day across a clock change.
//   - Reminders go while the patient hasn't started (S16 answers 2: starting stops them). If runs
//     were missed and several reminder days have passed, only the latest is sent, once.
//   - While five wrong dates of birth have locked the episode, a due reminder is withheld: counted
//     but not sent (a reminder's new link would undo the lock, which only staff's Resend may), and
//     not sent later either. After a Resend the later reminder days carry on as usual.
//   - "Not responding" is marked once, on its day, if the HQ hasn't been sent, whether or not the
//     patient started (EPI-02: "no submit").
//   - Once the HQ is sent, or if no invite ever reached the patient, there's nothing to do.
//   - While the trust or hospital is suspended nothing is done; it looks again in an hour.
func Due(f Followup, s Schedule, now time.Time) Actions {
	a := Actions{ReminderCount: f.RemindersSent}
	if f.InvitedAt.IsZero() || f.Submitted {
		return a
	}
	if !f.Open {
		a.Next = now.Add(time.Hour)
		return a
	}
	zone := s.Zone
	if zone == nil {
		zone = time.UTC
	}
	day := func(n int) time.Time { return f.InvitedAt.In(zone).AddDate(0, 0, n) }

	var next []time.Time
	if !f.Started {
		passed := 0
		for _, d := range s.ReminderDays {
			if !day(d).After(now) {
				passed++
			}
		}
		if passed > f.RemindersSent {
			a.Reminder, a.ReminderCount, a.Withheld = s.ReminderDays[passed-1], passed, f.Locked
		}
		if a.ReminderCount < len(s.ReminderDays) { // the days are in order, so this one is after now
			next = append(next, day(s.ReminderDays[a.ReminderCount]))
		}
	}
	if !f.NotResponding {
		if d := day(s.NotRespondingDay); d.After(now) {
			next = append(next, d)
		} else {
			a.NotResponding = true
		}
	}
	for _, t := range next {
		if a.Next.IsZero() || t.Before(a.Next) {
			a.Next = t
		}
	}
	return a
}
