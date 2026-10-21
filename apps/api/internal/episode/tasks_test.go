package episode

import (
	"strings"
	"testing"
)

func TestTaskInput(t *testing.T) {
	for _, c := range []struct {
		title, team string
		want, msg   string
	}{
		{"  Book anaesthetic review ", Anaesthetist, "Book anaesthetic review", ""},
		{"Check bloods", Nurse, "Check bloods", ""},
		{"Send the letter", Admin, "Send the letter", ""},
		{strings.Repeat("é", 200), Nurse, strings.Repeat("é", 200), ""}, // characters, not bytes
		{"", Nurse, "", "Enter the task."},
		{"   ", Nurse, "", "Enter the task."},
		{strings.Repeat("x", 201), Nurse, "", "The task is too long (200 characters at most)."},
		{"Check bloods", "", "", "Choose the team: Nurse, Anaesthetist or Admin."},
		{"Check bloods", "surgeon", "", "Choose the team: Nurse, Anaesthetist or Admin."},
		{"Check bloods", "Nurse", "", "Choose the team: Nurse, Anaesthetist or Admin."}, // keys, not labels
		{"Line one\nline two", Nurse, "", "Keep the task on one line."},
	} {
		got, msg := TaskInput(c.title, c.team)
		if got != c.want || msg != c.msg {
			t.Errorf("TaskInput(%q, %q) = %q, %q; want %q, %q", c.title, c.team, got, msg, c.want, c.msg)
		}
	}
}

func TestEveryTeamHasALabel(t *testing.T) {
	if len(Teams) != 3 {
		t.Errorf("Teams = %v, want nurse, anaesthetist and admin (S21 answers 1)", Teams)
	}
	for _, k := range Teams {
		if TeamLabels[k] == "" {
			t.Errorf("team %q has no label", k)
		}
	}
}

func TestWorklists(t *testing.T) {
	keys := func(ws []Worklist) []string {
		var out []string
		for _, w := range ws {
			out = append(out, w.Key)
		}
		return out
	}
	// The six fixed worklists, in the agreed order (S21 answers 2).
	want := []string{"ready_for_review", "not_responding", "anaesthetic_review", "procedure_soon", "on_hold", "open_tasks"}
	if got := keys(Worklists(14)); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Worklists = %v, want %v", got, want)
	}
	for _, w := range Worklists(14) {
		if w.Label == "" || w.Help == "" || !IsWorklist(w.Key) {
			t.Errorf("worklist %+v: want a label, a help line and IsWorklist", w)
		}
	}
	soon := Worklists(10)[3]
	if soon.Label != "Procedure within 10 days" || !strings.Contains(soon.Help, "next 10 days") {
		t.Errorf("one hospital's soon worklist = %+v, want its days", soon)
	}
	if one := Worklists(1)[3]; one.Label != "Procedure within 1 day" || !strings.Contains(one.Help, "next 1 day,") {
		t.Errorf("one day = %+v, want the singular", one)
	}
	if soon := Worklists(0)[3]; soon.Label != "Procedure soon" || !strings.Contains(soon.Help, "each hospital's") {
		t.Errorf("several hospitals' soon worklist = %+v, want no one number of days", soon)
	}
	for _, k := range []string{"", "all", "Ready_for_review", "auto_triaged"} {
		if IsWorklist(k) {
			t.Errorf("IsWorklist(%q) = true", k)
		}
	}
}

func TestStatusMessage(t *testing.T) {
	for _, c := range []struct {
		move Move
		to   string
		want string
	}{
		{Set, ReadyForAdmission, "readyForAdmission"}, // a pick by hand (S21 answers 4)
		{Cancel, Cancelled, "cancelled"},
		{Undo, ReadyForAdmission, ""}, // stepping back an admission tells the patient nothing
		{Set, OnHold, ""},
		{Admit, Admitted, ""},
		{Discharge, Discharged, ""},
		{CompleteReview, ReadyForPOA, ""},
		{Reopen, ReadyForReview, ""},
		{Submit, ReadyForReview, ""},
	} {
		if got := StatusMessage(c.move, c.to); got != c.want {
			t.Errorf("StatusMessage(%s, %s) = %q, want %q", c.move, c.to, got, c.want)
		}
	}
}
