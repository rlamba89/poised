package episode

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Tasks, worklists and status messages (plan C3, session S21; TSK-01, EPI-02, EPI-08).

// A task is for a team (S21 answers 1). The permission roles (clinician, super clinician, admin)
// aren't job functions, so the teams are a list of their own. Anyone at the hospital may
// complete any task.
const (
	Nurse        = "nurse"
	Anaesthetist = "anaesthetist"
	Admin        = "admin"
)

// Teams in the order the Tasks panel lists them.
var Teams = []string{Nurse, Anaesthetist, Admin}

// TeamLabels are the teams' names as staff see them.
var TeamLabels = map[string]string{Nurse: "Nurse", Anaesthetist: "Anaesthetist", Admin: "Admin"}

// MaxTaskTitle is the most characters a task's title may have.
const MaxTaskTitle = 200

// TaskInput checks a new task: its title (trimmed, one line, 1 to 200 characters) and its team's
// key. It returns the title to store, or a plain-language problem.
func TaskInput(title, team string) (string, string) {
	title = strings.TrimSpace(title)
	switch {
	case title == "":
		return "", "Enter the task."
	case strings.ContainsAny(title, "\r\n"):
		return "", "Keep the task on one line."
	case utf8.RuneCountInString(title) > MaxTaskTitle:
		return "", fmt.Sprintf("The task is too long (%d characters at most).", MaxTaskTitle)
	case !slices.Contains(Teams, team):
		return "", "Choose the team: Nurse, Anaesthetist or Admin."
	}
	return title, ""
}

// Worklist is one of the clinician home's built-in worklists (EPI-02, S21 answers 2): a saved
// filter over the episode list. Each one's rule is the database's in_worklist (migration 00022).
type Worklist struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Help  string `json:"help"` // one line saying what's in it
}

// Worklists are the worklists, in the order the tabs show them, for a list whose "procedure within N days" is soonDays (the
// hospital's worklists.soonDays setting), or 0 for a list of several hospitals, each with its own.
func Worklists(soonDays int) []Worklist {
	soon := Worklist{Key: "procedure_soon", Label: "Procedure soon",
		Help: "A planned procedure date within each hospital's number of days (its settings), and not yet Ready for admission."}
	if soonDays > 0 {
		days := fmt.Sprintf("%d days", soonDays)
		if soonDays == 1 {
			days = "1 day"
		}
		soon.Label = "Procedure within " + days
		soon.Help = "A planned procedure date today or in the next " + days + ", and not yet Ready for admission."
	}
	// Auto-triaged comes with auto-triage (C6).
	return []Worklist{
		{Key: "ready_for_review", Label: "Ready for review", Help: "The patient has sent their HQ, and the HQ review isn't complete."},
		{Key: "not_responding", Label: "Not responding", Help: "Marked not responding: the patient hasn't sent their HQ by the hospital's day."},
		{Key: "anaesthetic_review", Label: "Anaesthetic review needed", Help: "An open task for the anaesthetist team."},
		soon,
		{Key: "on_hold", Label: "On hold", Help: "The episode is On hold."},
		{Key: "open_tasks", Label: "Open tasks", Help: "At least one task that isn't done, for any team."},
	}
}

// IsWorklist reports whether key is one of the worklists.
func IsWorklist(key string) bool {
	return slices.ContainsFunc(Worklists(0), func(w Worklist) bool { return w.Key == key })
}

// StatusMessage is the message the patient may be sent after a move (EPI-08, S21 answers 4): the
// stem of its settings ("readyForAdmission": statusMessages.readyForAdmissionSms switches on
// templates.readyForAdmissionSms, and the same for Email), or "" for none. Only a move forward by
// hand sends one: a pick of Ready for admission, or cancelling. Undoing an admission, which lands
// on Ready for admission again, tells the patient nothing.
func StatusMessage(move Move, to string) string {
	switch {
	case move == Set && to == ReadyForAdmission:
		return "readyForAdmission"
	case move == Cancel:
		return "cancelled"
	}
	return ""
}
