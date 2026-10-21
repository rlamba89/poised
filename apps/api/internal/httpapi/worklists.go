package httpapi

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
	"github.com/rlamba89/poised/apps/api/internal/role"
	"github.com/rlamba89/poised/apps/api/internal/settings"
)

// The worklists on the clinician home (plan C3, session S21; EPI-02): built-in saved filters over
// the episode list. GET …/episodes?worklist=<key> lists one (writeEpisodes); these routes give the
// tabs their labels and counts, scoped as the lists are:
//
//   - GET /api/o/{oid}/h/{hid}/worklists: one hospital's, with its "procedure within N days";
//   - GET /api/o/{oid}/worklists: "All hospitals" (STF-03), every hospital's for a trust-wide
//     member, else the caller's hospitals', each episode counted with its own hospital's days.
//
// Each worklist's rule is the database's in_worklist (migration 00022), which both the list and
// the counts use, so a tab's count is its list's length. Counts are not patient data (no names, no
// IDs), so these routes aren't audited; opening a list is (episode.list).

// worklistCount is one tab.
type worklistCount struct {
	episode.Worklist
	Count int32 `json:"count"`
}

func (s *server) listWorklists(w http.ResponseWriter, r *http.Request) {
	v, err := s.hospitalSettings(r.Context(), orgID(r), hospitalID(r))
	if err != nil {
		serverError(w, "settings", err)
		return
	}
	var days int
	if err := settings.Get(v, "worklists.soonDays", &days); err != nil {
		serverError(w, "settings", err)
		return
	}
	s.writeWorklists(w, r, episode.Worklists(days), db.CountWorklistsParams{OrgID: orgID(r), HospitalIds: []uuid.UUID{hospitalID(r)}})
}

func (s *server) listTrustWorklists(w http.ResponseWriter, r *http.Request) {
	p := here(r)
	s.writeWorklists(w, r, episode.Worklists(0), db.CountWorklistsParams{OrgID: p.Org, AllHospitals: p.Role != role.None, HospitalIds: p.Hospitals})
}

func (s *server) writeWorklists(w http.ResponseWriter, r *http.Request, lists []episode.Worklist, arg db.CountWorklistsParams) {
	for _, l := range lists {
		arg.Worklists = append(arg.Worklists, l.Key)
	}
	if arg.HospitalIds == nil {
		arg.HospitalIds = []uuid.UUID{}
	}
	arg.Now, arg.DefaultZone, arg.SoonDays = s.clock.Now(), s.cfg.Country.TimeZone, defaultSoonDays()
	rows, err := s.q.CountWorklists(r.Context(), arg)
	if err != nil {
		serverError(w, "count worklists", err)
		return
	}
	n := map[string]int32{}
	for _, row := range rows {
		n[row.Worklist] = row.Episodes
	}
	out := []worklistCount{}
	for _, l := range lists {
		out = append(out, worklistCount{Worklist: l, Count: n[l.Key]})
	}
	writeJSON(w, http.StatusOK, out)
}

// defaultSoonDays is the platform's "procedure within N days" (14), for a hospital whose trust
// sets none either. The database reads each hospital's and trust's own (the list is of several).
func defaultSoonDays() int32 {
	var n int32
	if err := settings.Get(settings.Platform(), "worklists.soonDays", &n); err != nil {
		panic(err) // the default is the code's own, checked by settings' tests
	}
	return n
}
