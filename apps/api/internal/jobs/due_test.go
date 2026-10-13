package jobs

import (
	"testing"
	"time"
)

// The follow-up's decisions (S16): reminders on the hospital's days after the invite (2 and 5 by
// default), "not responding" on its day (7), nothing twice, nothing once the HQ is sent, and no
// more reminders once the patient has started (S16 answers 2).

func zone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestDue(t *testing.T) {
	london := zone(t, "Europe/London")
	sched := Schedule{ReminderDays: []int{2, 5}, NotRespondingDay: 7, Zone: london}
	invited := time.Date(2026, 10, 1, 14, 30, 0, 0, time.UTC) // 15:30 in London
	at := func(days int, d time.Duration) time.Time { return invited.AddDate(0, 0, days).Add(d) }
	open := Followup{InvitedAt: invited, Open: true}

	cases := []struct {
		name string
		f    Followup
		now  time.Time
		want Actions
	}{
		{"just invited: nothing yet; look again on day 2",
			open, invited, Actions{Next: at(2, 0)}},
		{"a minute before day 2: nothing yet",
			open, at(2, -time.Minute), Actions{Next: at(2, 0)}},
		{"day 2: reminder 1; look again on day 5",
			open, at(2, 0), Actions{Reminder: 2, ReminderCount: 1, Next: at(5, 0)}},
		{"day 2 again after reminder 1: nothing (no double sending)",
			Followup{InvitedAt: invited, Open: true, RemindersSent: 1}, at(2, time.Hour), Actions{ReminderCount: 1, Next: at(5, 0)}},
		{"day 5: reminder 2; look again on day 7",
			Followup{InvitedAt: invited, Open: true, RemindersSent: 1}, at(5, 0), Actions{Reminder: 5, ReminderCount: 2, Next: at(7, 0)}},
		{"day 7: not responding; nothing more to do",
			Followup{InvitedAt: invited, Open: true, RemindersSent: 2}, at(7, 0), Actions{ReminderCount: 2, NotResponding: true}},
		{"day 7 again: nothing (marked once)",
			Followup{InvitedAt: invited, Open: true, RemindersSent: 2, NotResponding: true}, at(7, time.Hour), Actions{ReminderCount: 2}},
		{"a run missed days 2 and 5: only the latest reminder, once",
			open, at(5, time.Hour), Actions{Reminder: 5, ReminderCount: 2, Next: at(7, 0)}},
		{"the first run is after day 7: the last reminder and not responding together",
			open, at(9, 0), Actions{Reminder: 5, ReminderCount: 2, NotResponding: true}},
		{"submitted: finished, nothing sent",
			Followup{InvitedAt: invited, Open: true, Submitted: true}, at(5, 0), Actions{}},
		{"submitted after reminder 1: finished",
			Followup{InvitedAt: invited, Open: true, Submitted: true, RemindersSent: 1}, at(7, 0), Actions{ReminderCount: 1}},
		{"started: no reminders, but not responding is still due on day 7",
			Followup{InvitedAt: invited, Open: true, Started: true}, at(2, 0), Actions{Next: at(7, 0)}},
		{"started, day 7: not responding",
			Followup{InvitedAt: invited, Open: true, Started: true, RemindersSent: 1}, at(7, 0), Actions{ReminderCount: 1, NotResponding: true}},
		{"never invited: nothing, ever",
			Followup{Open: true}, at(9, 0), Actions{}},
		{"locked (5 wrong dates of birth): the day 2 reminder is withheld, not sent later; day 5 is still due",
			Followup{InvitedAt: invited, Open: true, Locked: true}, at(2, 0), Actions{Reminder: 2, ReminderCount: 1, Withheld: true, Next: at(5, 0)}},
		{"locked on day 7: not responding still comes",
			Followup{InvitedAt: invited, Open: true, Locked: true, RemindersSent: 2}, at(7, 0), Actions{ReminderCount: 2, NotResponding: true}},
		{"unlocked again (Resend) before day 5: day 5's reminder goes",
			Followup{InvitedAt: invited, Open: true, RemindersSent: 1}, at(5, 0), Actions{Reminder: 5, ReminderCount: 2, Next: at(7, 0)}},
		{"trust or hospital suspended: nothing now, look again in an hour",
			Followup{InvitedAt: invited, RemindersSent: 1}, at(5, 0), Actions{ReminderCount: 1, Next: at(5, time.Hour)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Due(c.f, sched, c.now)
			if got.Reminder != c.want.Reminder || got.ReminderCount != c.want.ReminderCount || got.Withheld != c.want.Withheld ||
				got.NotResponding != c.want.NotResponding || !got.Next.Equal(c.want.Next) {
				t.Errorf("Due = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestDueSchedules(t *testing.T) {
	london := zone(t, "Europe/London")
	invited := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	f := Followup{InvitedAt: invited, Open: true}

	t.Run("no reminder days: only not responding", func(t *testing.T) {
		got := Due(f, Schedule{ReminderDays: []int{}, NotRespondingDay: 7, Zone: london}, invited.AddDate(0, 0, 3))
		if got.Reminder != 0 || !got.Next.Equal(invited.AddDate(0, 0, 7)) {
			t.Errorf("Due = %+v", got)
		}
	})
	t.Run("a reminder after the not-responding day still goes", func(t *testing.T) {
		s := Schedule{ReminderDays: []int{2, 10}, NotRespondingDay: 7, Zone: london}
		got := Due(Followup{InvitedAt: invited, Open: true, RemindersSent: 1, NotResponding: true}, s, invited.AddDate(0, 0, 10))
		if got.Reminder != 10 || got.ReminderCount != 2 || !got.Next.IsZero() {
			t.Errorf("Due = %+v", got)
		}
	})
	t.Run("fewer days than reminders already sent (the setting changed): no more reminders", func(t *testing.T) {
		s := Schedule{ReminderDays: []int{3}, NotRespondingDay: 7, Zone: london}
		got := Due(Followup{InvitedAt: invited, Open: true, RemindersSent: 2}, s, invited.AddDate(0, 0, 4))
		if got.Reminder != 0 || got.ReminderCount != 2 || !got.Next.Equal(invited.AddDate(0, 0, 7)) {
			t.Errorf("Due = %+v", got)
		}
	})
	t.Run("days are the hospital's calendar days: same local time across a clock change", func(t *testing.T) {
		// Invited at 10:00 GMT on Friday 27 March 2026; the clocks go forward on Sunday 29 March,
		// so day 2 is 10:00 BST, 09:00 UTC: 47 hours later, not 48.
		inv := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
		s := Schedule{ReminderDays: []int{2}, NotRespondingDay: 7, Zone: london}
		day2 := time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC)
		if got := Due(Followup{InvitedAt: inv, Open: true}, s, inv); !got.Next.Equal(day2) {
			t.Errorf("next = %v, want %v", got.Next, day2)
		}
		if got := Due(Followup{InvitedAt: inv, Open: true}, s, day2); got.Reminder != 2 {
			t.Errorf("at 10:00 local on day 2: %+v", got)
		}
		// Another zone's hospital: no clock change that weekend, so 48 hours.
		toronto := Schedule{ReminderDays: []int{2}, NotRespondingDay: 7, Zone: zone(t, "America/Toronto")}
		if got := Due(Followup{InvitedAt: inv, Open: true}, toronto, inv); !got.Next.Equal(inv.Add(48 * time.Hour)) {
			t.Errorf("Toronto next = %v", got.Next)
		}
	})
	t.Run("the next look is always after now", func(t *testing.T) {
		s := Schedule{ReminderDays: []int{1, 2, 3, 4, 5}, NotRespondingDay: 6, Zone: london}
		for h := 0; h < 24*8; h++ {
			now := invited.Add(time.Duration(h) * time.Hour)
			for sent := 0; sent <= 5; sent++ {
				for _, flagged := range []bool{false, true} {
					got := Due(Followup{InvitedAt: invited, Open: true, RemindersSent: sent, NotResponding: flagged}, s, now)
					if !got.Next.IsZero() && !got.Next.After(now) {
						t.Fatalf("at +%dh, %d sent: next %v isn't after now", h, sent, got.Next)
					}
				}
			}
		}
	})
	t.Run("no zone: UTC", func(t *testing.T) {
		got := Due(f, Schedule{ReminderDays: []int{2}, NotRespondingDay: 7}, invited)
		if !got.Next.Equal(invited.AddDate(0, 0, 2)) {
			t.Errorf("Due = %+v", got)
		}
	})
}
