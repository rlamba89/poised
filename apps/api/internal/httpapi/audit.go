package httpapi

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
)

// The audit log (plan C1, session S11; STF-06, AUD-01). Every route that reads or changes patient
// data is wrapped in audited, which writes one audit_log row per request (S11 answers 1, Rahul,
// 12 Oct): who (the staff member, or the patient through their link), when, the trust, the
// hospital, the action, the route pattern, the status answered, and the IDs of the patients and
// episodes the request touched. Never patient values: names are looked up when the log is read.
//
// No row, no data: the handler's response is held until its row is written, and if the row
// can't be written the caller gets a 500 instead. A change the handler already saved stays saved.
//
// Rows are written only for requests that passed the sign-in and membership checks (they're
// the trust's to read); a refusal after that, such as an episode id from another hospital, is
// written with its status. The patient's link writes a row once its token names an episode.

// auditActions are the actions written to audit_log.action, and how people read them.
var auditActions = map[string]string{
	"patient.search":     "Searched patients",
	"patient.view":       "Opened a patient",
	"patient.list":       "Listed patients", // before S13
	"patient.create":     "Added a patient", // before S13: now with the episode (episode.create)
	"episode.list":       "Listed episodes",
	"episode.list_trust": "Listed episodes (all hospitals)",
	"episode.create":     "Created an episode",
	"episode.view":       "Opened an episode",
	"episode.update":     "Changed an episode",
	"episode.note":       "Added a note",
	"episode.status":     "Changed an episode's status", // S20
	"episode.cancel":     "Cancelled an episode",
	"episode.reopen":     "Re-opened the HQ review",
	"episode.undo":       "Undid a status change",
	"episode.archive":    "Archived an episode", // S22
	"episode.unarchive":  "Un-archived an episode",
	"task.list":          "Opened an episode's tasks", // S21
	"task.create":        "Added a task",
	"task.update":        "Completed or re-opened a task",
	"answers.view":       "Opened the answers",
	"answers.save":       "Saved the clinician's answers",
	"review.complete":    "Completed the HQ review",
	"invite.resend":      "Resent the patient's invite",
	"behalf.view":        "Opened the patient's HQ to fill in for them",
	"behalf.save":        "Saved answers for the patient",
	"behalf.submit":      "Sent the HQ for the patient",
	"behalf.prefill":     "Checked pre-filled answers with the patient", // S25: the opening card's button
	"link.open":          "Patient opened their link",                   // S14: Continue
	"link.dob":           "Patient entered their date of birth",
	"link.view":          "Patient opened their questionnaire",
	"link.save":          "Patient saved answers",
	"link.submit":        "Patient sent their answers",
	"link.prefill":       "Patient checked pre-filled answers", // S25
	"home.view":          "Patient opened their home",          // S15: a row per hospital it listed
	"account.join":       "Patient's record joined their sign-in account",
	"contact.update":     "Patient changed their contact details",
	"account.close":      "Patient closed their account", // S22: a row per trust whose records it unlinked
	"procedure.create":   "Added a procedure",            // S19: the trust's procedure list
	"procedure.update":   "Changed a procedure",
	"audit.view":         "Viewed the audit log",
	"audit.export":       "Exported the audit log",
}

const auditKey ctxKey = 100

// auditEntry is what a request's row will say. audited starts it; the handler adds the records
// it touches (and, on the patient's link, the trust and hospital).
type auditEntry struct {
	action   string
	org      uuid.UUID
	hospital uuid.UUID // uuid.Nil: a trust route
	patients idSet
	episodes idSet
	// more are the request's further rows: a patient's request can read or change records in
	// several trusts and hospitals (their home, a contact change, S15), and each trust's log
	// gets its own row.
	more []*auditEntry
}

// also is the request's row for another trust (and hospital, uuid.Nil for none), and action:
// the one already started, or a new one.
func (e *auditEntry) also(action string, org, hospital uuid.UUID) *auditEntry {
	for _, m := range e.more {
		if m.action == action && m.org == org && m.hospital == hospital {
			return m
		}
	}
	m := &auditEntry{action: action, org: org, hospital: hospital}
	e.more = append(e.more, m)
	return m
}

// rows are the entries to write: this one if it has a trust, and the further ones.
func (e *auditEntry) rows() []*auditEntry {
	var out []*auditEntry
	if e.org != uuid.Nil {
		out = append(out, e)
	}
	return append(out, e.more...)
}

// idSet is a list of IDs without repeats, in the order first added. A list screen can touch every
// patient in the trust, so the repeat check is a map, not a scan.
type idSet struct {
	ids  []uuid.UUID
	seen map[uuid.UUID]struct{}
}

func (s *idSet) add(id uuid.UUID) {
	if _, ok := s.seen[id]; ok {
		return
	}
	if s.seen == nil {
		s.seen = map[uuid.UUID]struct{}{}
	}
	s.seen[id] = struct{}{}
	s.ids = append(s.ids, id)
}

// list is the IDs, never nil (a NULL array would break the NOT NULL column).
func (s *idSet) list() []uuid.UUID {
	if s.ids == nil {
		return []uuid.UUID{}
	}
	return s.ids
}

// auditOf is the request's audit entry. Outside an audited route it's a throwaway one, so
// helpers shared with other routes can note records without checking.
func auditOf(r *http.Request) *auditEntry {
	if e, ok := r.Context().Value(auditKey).(*auditEntry); ok {
		return e
	}
	return &auditEntry{}
}

// touched notes a patient and (if not Nil) an episode the request read or changed.
func (e *auditEntry) touched(patient, episode uuid.UUID) {
	if patient != uuid.Nil {
		e.patients.add(patient)
	}
	if episode != uuid.Nil {
		e.episodes.add(episode)
	}
}

// audited wraps a handler that reads or changes patient data. On a staff route it runs inside
// requireHospital or requireOrg, which give it the trust and hospital.
func (s *server) audited(action string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := here(r)
		e := &auditEntry{action: action, org: p.Org, hospital: p.Hospital}
		held := &heldResponse{header: http.Header{}}
		next(held, r.WithContext(context.WithValue(r.Context(), auditKey, e)))
		// None: the patient's link with a token that names no episode, or a patient request that
		// touched no trust's records.
		for _, row := range e.rows() {
			if err := s.writeAudit(r, row, held.code()); err != nil {
				serverError(w, "audit "+row.action, err)
				return
			}
		}
		held.sendTo(w)
	}
}

func (s *server) writeAudit(r *http.Request, e *auditEntry, status int) error {
	arg := db.InsertAuditParams{
		At: s.clock.Now(), OrgID: e.org, HospitalID: uuid.NullUUID{UUID: e.hospital, Valid: e.hospital != uuid.Nil},
		Actor: "patient", Action: e.action, Route: r.Pattern, Status: int16(status),
		PatientIds: e.patients.list(), EpisodeIds: e.episodes.list(),
	}
	if u := currentUser(r); u.ID != uuid.Nil {
		arg.Actor, arg.UserID = "staff", uuid.NullUUID{UUID: u.ID, Valid: true}
	}
	// Written even if the caller has gone: a change they asked for may already be saved.
	return s.q.InsertAudit(context.WithoutCancel(r.Context()), arg)
}

// heldResponse holds a handler's response until its audit row is written.
type heldResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (h *heldResponse) Header() http.Header { return h.header }

func (h *heldResponse) WriteHeader(code int) {
	if h.status == 0 {
		h.status = code
	}
}

func (h *heldResponse) Write(b []byte) (int, error) {
	h.WriteHeader(http.StatusOK)
	return h.body.Write(b)
}

func (h *heldResponse) code() int {
	if h.status == 0 {
		return http.StatusOK
	}
	return h.status
}

func (h *heldResponse) sendTo(w http.ResponseWriter) {
	for k, v := range h.header {
		w.Header()[k] = v
	}
	w.WriteHeader(h.code())
	w.Write(h.body.Bytes())
}

// --- GET /api/o/{oid}/audit: the trust's log, for its whole-trust admins ---

// auditPageSize is how many rows the screen shows; the CSV has them all.
const auditPageSize = 500

// auditRowPatients is how many patients (and episode ids) a row gives the screen, with their
// counts. A list screen's row can touch every patient in the trust, so naming them all would make
// the page grow with the trust (S11 verification 1). The CSV names them all.
const auditRowPatients = 10

// auditAdmin answers 403 unless the caller is an admin of the whole trust (as for Staff and
// Settings). It runs before audited, so a refused read of the log writes no row.
func auditAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !here(r).TrustRole.AtLeast(role.Admin) {
			writeError(w, http.StatusForbidden, "Only the trust's admins can see the audit log.")
			return
		}
		next(w, r)
	}
}

// auditFilter is the list's and the CSV's query string.
type auditFilter struct {
	params  db.ListAuditParams
	patient string         // the Patient filter as typed
	csv     bool           // format=csv
	zone    *time.Location // the CSV's times: tz, or the deployment's
}

// parseAuditFilter reads ?user=&from=&to=&patient=&format=&tz=, or returns a plain-language
// problem. user is a staff member's id or "patient" (the patient's link); from and to are
// RFC 3339 times (the browser turns the days chosen into its own midnight), from inclusive, to not.
func (s *server) parseAuditFilter(r *http.Request) (auditFilter, string) {
	q := r.URL.Query()
	f := auditFilter{params: db.ListAuditParams{OrgID: orgID(r), MaxRows: auditPageSize + 1}}
	switch user := q.Get("user"); user {
	case "":
	case "patient":
		f.params.PatientsOnly = true
	default:
		id, err := uuid.Parse(user)
		if err != nil {
			return f, "Choose someone from the list."
		}
		f.params.UserID = uuid.NullUUID{UUID: id, Valid: true}
	}
	for _, t := range []struct {
		name string
		dest *pgtype.Timestamptz
	}{{"from", &f.params.FromAt}, {"to", &f.params.ToAt}} {
		if v := q.Get(t.name); v != "" {
			at, err := time.Parse(time.RFC3339Nano, v)
			if err != nil {
				return f, "Enter valid dates."
			}
			*t.dest = pgtype.Timestamptz{Time: at, Valid: true}
		}
	}
	if f.params.FromAt.Valid && f.params.ToAt.Valid && !f.params.FromAt.Time.Before(f.params.ToAt.Time) {
		return f, "The start date must be before the end date."
	}
	f.patient = strings.TrimSpace(q.Get("patient"))
	if len(f.patient) > 100 {
		return f, "The patient search is too long."
	}
	switch q.Get("format") {
	case "", "json":
	case "csv":
		f.csv = true
		f.params.MaxRows = math.MaxInt32
	default:
		return f, "Choose JSON or CSV."
	}
	// The CSV's times are in the browser's zone (tz), so they match the screen; without one, the
	// deployment's (A-20: checked at start-up). "Local" would be the server's own, which means
	// nothing to the reader.
	tz := q.Get("tz")
	if tz == "" {
		tz = s.cfg.Country.TimeZone
	}
	zone, err := time.LoadLocation(tz)
	if err != nil || tz == "Local" {
		return f, "Choose a valid time zone."
	}
	f.zone = zone
	return f, ""
}

// auditListRow is one row of the list, with the names read now.
type auditListRow struct {
	ID           int64          `json:"id"`
	At           time.Time      `json:"at"`
	Actor        string         `json:"actor"` // staff or patient
	UserID       uuid.NullUUID  `json:"userId"`
	UserName     string         `json:"userName"`
	UserEmail    string         `json:"userEmail"`
	HospitalID   uuid.NullUUID  `json:"hospitalId"`
	HospitalName string         `json:"hospitalName"` // "" on a trust route
	Action       string         `json:"action"`
	ActionLabel  string         `json:"actionLabel"`
	Route        string         `json:"route"`
	Status       int16          `json:"status"`
	Patients     []auditPatient `json:"patients"`     // on the screen, the first auditRowPatients
	PatientCount int            `json:"patientCount"` // every patient the request touched
	EpisodeIDs   []uuid.UUID    `json:"episodeIds"`   // on the screen, the first auditRowPatients
	EpisodeCount int            `json:"episodeCount"` // every episode the request touched
}

type auditPatient struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	NationalID string    `json:"nationalId"` // "" if none
}

func (s *server) listAudit(w http.ResponseWriter, r *http.Request) {
	// An export is logged as one even when its filters are refused (S11 test report 2).
	if r.URL.Query().Get("format") == "csv" {
		auditOf(r).action = "audit.export"
	}
	f, msg := s.parseAuditFilter(r)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	if f.patient != "" {
		ids, err := s.q.MatchAuditPatients(ctx, db.MatchAuditPatientsParams{OrgID: orgID(r), Query: f.patient,
			NationalID: s.cfg.Country.NationalID().Normalise(f.patient)})
		if err != nil {
			serverError(w, "match patients", err)
			return
		}
		f.params.ByPatient, f.params.PatientIds = true, ids
		// Looking up a patient's history reads their data (STF-06), so this view or export is
		// recorded against the patients found. An unfiltered view names none: it would copy every
		// name shown into the log.
		for _, id := range ids {
			auditOf(r).touched(id, uuid.Nil)
		}
	}
	if f.params.PatientIds == nil {
		f.params.PatientIds = []uuid.UUID{}
	}
	rows, err := s.q.ListAudit(ctx, f.params)
	if err != nil {
		serverError(w, "list audit", err)
		return
	}
	more := !f.csv && len(rows) > auditPageSize
	if more {
		rows = rows[:auditPageSize]
	}
	perRow := auditRowPatients
	if f.csv {
		perRow = 0 // all of them
	}
	list, err := s.auditListRows(ctx, orgID(r), rows, perRow)
	if err != nil {
		serverError(w, "audit names", err)
		return
	}
	if f.csv {
		s.writeAuditCSV(w, r, list, f.zone)
		return
	}
	users, err := s.q.ListAuditUsers(ctx, orgID(r))
	if err != nil {
		serverError(w, "audit users", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": list, "more": more, "limit": auditPageSize, "users": users})
}

// auditListRows adds the patients' names, read now from the trust's records: the first perRow
// of each row's patients, or all of them when perRow is 0.
func (s *server) auditListRows(ctx context.Context, org uuid.UUID, rows []db.ListAuditRow, perRow int) ([]auditListRow, error) {
	shown := func(row db.ListAuditRow) []uuid.UUID {
		if perRow > 0 && len(row.PatientIds) > perRow {
			return row.PatientIds[:perRow]
		}
		return row.PatientIds
	}
	var all idSet
	for _, row := range rows {
		for _, id := range shown(row) {
			all.add(id)
		}
	}
	names := map[uuid.UUID]auditPatient{}
	if len(all.ids) > 0 {
		found, err := s.q.AuditPatientNames(ctx, db.AuditPatientNamesParams{OrgID: org, Ids: all.ids})
		if err != nil {
			return nil, err
		}
		for _, p := range found {
			names[p.ID] = auditPatient{ID: p.ID, Name: p.FirstName + " " + p.LastName, NationalID: p.NationalID}
		}
	}
	out := make([]auditListRow, 0, len(rows))
	for _, row := range rows {
		l := auditListRow{
			ID: row.ID, At: row.At, Actor: row.Actor, UserID: row.UserID, HospitalID: row.HospitalID,
			Action: row.Action, ActionLabel: auditActions[row.Action], Route: row.Route, Status: row.Status,
			Patients: []auditPatient{}, PatientCount: len(row.PatientIds),
			EpisodeIDs: row.EpisodeIds, EpisodeCount: len(row.EpisodeIds),
		}
		if perRow > 0 && len(l.EpisodeIDs) > perRow {
			l.EpisodeIDs = l.EpisodeIDs[:perRow]
		}
		if l.ActionLabel == "" {
			l.ActionLabel = row.Action
		}
		if row.UserName != nil {
			l.UserName, l.UserEmail = *row.UserName, *row.UserEmail
		}
		if row.HospitalName != nil {
			l.HospitalName = *row.HospitalName
		}
		for _, id := range shown(row) {
			p, ok := names[id]
			if !ok {
				p = auditPatient{ID: id, Name: "(record not found)"}
			}
			l.Patients = append(l.Patients, p)
		}
		out = append(out, l)
	}
	return out, nil
}

// writeAuditCSV writes the rows as a CSV download, times in zone.
func (s *server) writeAuditCSV(w http.ResponseWriter, r *http.Request, rows []auditListRow, zone *time.Location) {
	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)
	cw.Write([]string{"Time", "Who", "Email", "Action", "Hospital", "Patients", "Patient record IDs", "Episode IDs", "Route", "Result"})
	for _, row := range rows {
		who := row.UserName
		if row.Actor == "patient" {
			who = "Patient (their link)"
		}
		var names, pids, eids []string
		for _, p := range row.Patients {
			names = append(names, p.Name)
			pids = append(pids, p.ID.String())
		}
		for _, id := range row.EpisodeIDs {
			eids = append(eids, id.String())
		}
		cw.Write(csvSafe(
			row.At.In(zone).Format(time.RFC3339), who, row.UserEmail, row.ActionLabel, row.HospitalName,
			strings.Join(names, "; "), strings.Join(pids, "; "), strings.Join(eids, "; "), row.Route, strconv.Itoa(int(row.Status)),
		))
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		serverError(w, "audit csv", err)
		return
	}
	name := fmt.Sprintf("audit-%s.csv", s.clock.Now().In(zone).Format("2006-01-02-150405"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

// csvSafe defuses cells a spreadsheet would run as a formula (CSV injection): a cell starting
// with = + - @ (or a tab or carriage return) gets a leading apostrophe. Names and emails come
// from people, so they can't be trusted.
func csvSafe(cells ...string) []string {
	for i, c := range cells {
		if c != "" && strings.ContainsRune("=+-@\t\r", rune(c[0])) {
			cells[i] = "'" + c
		}
	}
	return cells
}
