//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/apitest"
)

// The patient's Information for an episode (plan C5, session S36; INF-04): videos and leaflets
// from the content library (S33) and the hospital's phone and email.
//   - A procedure's list: PATCH (or POST) /api/o/{oid}/procedures/{prid} {"infoContent": [ids]},
//     by whole-trust super clinicians and admins, of Preopsis's and the trust's own items.
//   - It's copied onto an episode when the episode is made (S36 answers 2); changing the episode's
//     procedure swaps the copied items for the new procedure's.
//   - One episode's list: GET, POST and DELETE …/episodes/{eid}/information[/{cid}], for anyone at
//     the hospital, audited.
//   - The patient's read: GET /api/p/episodes/{eid}/information, their own episodes only, active
//     items only, and never another trust's.

// infoWorld adds content items to the clinic: trust A's own and Preopsis's, and trust B's.
type infoWorld struct {
	*clinic
	video, leaflet, retired, theirs, platformLeaflet uuid.UUID
}

func newInfoWorld(t *testing.T) *infoWorld {
	t.Helper()
	w := &infoWorld{clinic: newClinic(t)}
	w.video = w.contentItem(t, nil, "video", "Your anaesthetic", "active")
	w.platformLeaflet = w.contentItem(t, nil, "leaflet", "Stopping smoking", "active")
	w.leaflet = w.contentItem(t, &w.a.Org, "leaflet", "Fasting before your operation", "active")
	w.retired = w.contentItem(t, &w.a.Org, "leaflet", "Old leaflet", "retired")
	w.theirs = w.contentItem(t, &w.b.Org, "leaflet", "Trust B's leaflet", "active")
	return w
}

// contentItem inserts a library item (org nil: Preopsis's) and returns its id.
func (w *infoWorld) contentItem(t *testing.T, org *uuid.UUID, kind, title, status string) uuid.UUID {
	t.Helper()
	url, length := "https://example.org/"+strings.ReplaceAll(strings.ToLower(title), " ", "-")+".pdf", (*int)(nil)
	if kind == "video" {
		url, length = "https://www.youtube.com/watch?v=aqz-KE-bpKQ", new(int)
		*length = 125
	}
	var id uuid.UUID
	if err := w.db.QueryRow(t.Context(), `INSERT INTO content_items (org_id, kind, title, url, length_seconds, status)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, org, kind, title, url, length, status).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// procedureInfo is a procedure's stored list.
func (w *infoWorld) procedureInfo(t *testing.T, id uuid.UUID) []uuid.UUID {
	t.Helper()
	var raw []byte
	if err := w.db.QueryRow(t.Context(), `SELECT info_content FROM procedures WHERE id = $1`, id).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var out []uuid.UUID
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("info_content %s: %v", raw, err)
	}
	return out
}

func (w *infoWorld) setProcedureInfo(t *testing.T, id uuid.UUID, items ...uuid.UUID) {
	t.Helper()
	raw, _ := json.Marshal(append([]uuid.UUID{}, items...))
	if _, err := w.db.Exec(t.Context(), `UPDATE procedures SET info_content = $2 WHERE id = $1`, id, raw); err != nil {
		t.Fatal(err)
	}
}

// infoRow is an episode_info_items row.
type infoRow struct {
	Item          uuid.UUID
	FromProcedure bool
	AddedBy       *uuid.UUID
}

func (w *infoWorld) episodeInfo(t *testing.T, episode uuid.UUID) []infoRow {
	t.Helper()
	rows, err := w.db.Query(t.Context(), `SELECT content_item_id, from_procedure, added_by FROM episode_info_items
		WHERE episode_id = $1 ORDER BY from_procedure DESC, position, added_at`, episode)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []infoRow
	for rows.Next() {
		var r infoRow
		if err := rows.Scan(&r.Item, &r.FromProcedure, &r.AddedBy); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

func infoItems(rows []infoRow) []uuid.UUID {
	out := []uuid.UUID{}
	for _, r := range rows {
		out = append(out, r.Item)
	}
	return out
}

// addInfo puts an item on an episode straight into the database (no checks: another trust's too).
func (w *infoWorld) addInfo(t *testing.T, episode, item uuid.UUID, fromProcedure bool, position int) {
	t.Helper()
	if _, err := w.db.Exec(t.Context(), `INSERT INTO episode_info_items (episode_id, content_item_id, from_procedure, position)
		VALUES ($1, $2, $3, $4)`, episode, item, fromProcedure, position); err != nil {
		t.Fatal(err)
	}
}

func TestProcedureInformation(t *testing.T) {
	t.Parallel()
	w := newInfoWorld(t)
	patch := func(c *apitest.Client, org, id uuid.UUID, body any) *apitest.Resp {
		return c.Do("PATCH", proceduresPath(org, "/"+id.String()), body)
	}
	set := func(items ...any) map[string]any { return map[string]any{"infoContent": append([]any{}, items...)} }

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, patch(w.anon, w.a.Org, w.hip, set(w.video)), http.StatusUnauthorized)
	})
	t.Run("wrong role: a clinician", func(t *testing.T) {
		refused(t, patch(w.clinician, w.a.Org, w.hip, set(w.video)), http.StatusForbidden)
		if got := w.procedureInfo(t, w.hip); len(got) != 0 {
			t.Errorf("info = %v", got)
		}
	})
	t.Run("another trust: not a member, its procedure by this trust's path, or its items", func(t *testing.T) {
		refused(t, patch(w.other, w.a.Org, w.hip, set(w.video)), http.StatusNotFound)
		refused(t, patch(w.other, w.b.Org, w.hip, set(w.video)), http.StatusNotFound)
		refused(t, patch(w.author, w.a.Org, w.hip, set(w.theirs)), http.StatusBadRequest)
		if got := w.procedureInfo(t, w.hip); len(got) != 0 {
			t.Errorf("info = %v", got)
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, bad := range []any{
			set("not-an-id"),
			set(uuid.New()),       // no such item
			set(w.retired),        // retired: not offered any more
			set(w.video, w.video), // twice
			map[string]any{"infoContent": nil},
			map[string]any{"infoContent": "x"},
		} {
			refused(t, patch(w.author, w.a.Org, w.hip, bad), http.StatusBadRequest)
		}
		many := []any{}
		for range 51 {
			many = append(many, w.video)
		}
		refused(t, patch(w.author, w.a.Org, w.hip, set(many...)), http.StatusBadRequest)
		if got := w.procedureInfo(t, w.hip); len(got) != 0 {
			t.Errorf("info = %v", got)
		}
	})
	t.Run("happy path: Preopsis's and the trust's own items, in order; the list names each; audited", func(t *testing.T) {
		before := lastAuditID(t, w.clinic)
		patch(w.author, w.a.Org, w.hip, set(w.leaflet, w.video)).Want(t, http.StatusNoContent)
		if got := w.procedureInfo(t, w.hip); !slices.Equal(got, []uuid.UUID{w.leaflet, w.video}) {
			t.Errorf("info = %v", got)
		}
		if rows := auditRows(t, w.clinic, before); len(rows) != 1 || rows[0].Action != "procedure.update" {
			t.Errorf("audit = %+v", rows)
		}
		var got []struct {
			ID        uuid.UUID
			InfoItems []struct {
				ID                  uuid.UUID
				Kind, Title, Status string
			}
		}
		r := w.author.Do("GET", proceduresPath(w.a.Org, ""), nil)
		r.Want(t, http.StatusOK)
		r.Decode(t, &got)
		i := slices.IndexFunc(got, func(p struct {
			ID        uuid.UUID
			InfoItems []struct {
				ID                  uuid.UUID
				Kind, Title, Status string
			}
		}) bool {
			return p.ID == w.hip
		})
		if i < 0 || len(got[i].InfoItems) != 2 || got[i].InfoItems[0].Title != "Fasting before your operation" || got[i].InfoItems[1].Kind != "video" ||
			got[i].InfoItems[1].Status != "active" {
			t.Errorf("procedures = %+v", got)
		}
	})
	t.Run("happy path: an admin may too; an empty list clears it", func(t *testing.T) {
		admin := w.anon.LoginAs(t, w.fx.TrustUser(t, w.a.Org, "admin"))
		patch(admin, w.a.Org, w.knee, set(w.platformLeaflet)).Want(t, http.StatusNoContent)
		patch(admin, w.a.Org, w.knee, set()).Want(t, http.StatusNoContent)
		if got := w.procedureInfo(t, w.knee); len(got) != 0 {
			t.Errorf("info = %v", got)
		}
	})
	t.Run("an item retired since stays while the list is changed around it, and is named retired", func(t *testing.T) {
		item := w.contentItem(t, &w.a.Org, "leaflet", "Soon retired", "active")
		patch(w.author, w.a.Org, w.cataract, set(item)).Want(t, http.StatusNoContent)
		if _, err := w.db.Exec(t.Context(), `UPDATE content_items SET status = 'retired' WHERE id = $1`, item); err != nil {
			t.Fatal(err)
		}
		patch(w.author, w.a.Org, w.cataract, set(item, w.video)).Want(t, http.StatusNoContent)
		patch(w.author, w.a.Org, w.cataract, map[string]any{"name": "Cataract surgery"}).Want(t, http.StatusNoContent)
		if got := w.procedureInfo(t, w.cataract); !slices.Equal(got, []uuid.UUID{item, w.video}) {
			t.Errorf("info = %v", got)
		}
	})
	t.Run("POST takes the list too", func(t *testing.T) {
		id := w.c().addProcedure(t, w.a.Org, map[string]any{"name": "Bunion", "code": "", "specialty": "", "infoContent": []any{w.video}})
		if got := w.procedureInfo(t, id); !slices.Equal(got, []uuid.UUID{w.video}) {
			t.Errorf("info = %v", got)
		}
	})
}

// c is a helper to create procedures as Alex.
type procedureMaker struct {
	w *infoWorld
}

func (w *infoWorld) c() procedureMaker { return procedureMaker{w} }

func (m procedureMaker) addProcedure(t *testing.T, org uuid.UUID, body map[string]any) uuid.UUID {
	t.Helper()
	r := m.w.author.Do("POST", proceduresPath(org, ""), body)
	r.Want(t, http.StatusCreated)
	var got struct{ ID uuid.UUID }
	r.Decode(t, &got)
	return got.ID
}

func TestEpisodeGetsItsProcedureInformation(t *testing.T) {
	t.Parallel()
	w := newInfoWorld(t)
	w.setProcedureInfo(t, w.hip, w.leaflet, w.video, w.retired)
	w.setProcedureInfo(t, w.knee, w.platformLeaflet, w.video)
	create := func(t *testing.T, procedure uuid.UUID) uuid.UUID {
		t.Helper()
		body := map[string]any{"patientId": w.ep.PatientID, "versionId": w.hq.VersionID, "procedureId": ""}
		if procedure != uuid.Nil {
			body["procedureId"] = procedure
		}
		r := w.clinician.Do("POST", at(w.a, "/episodes"), body)
		r.Want(t, http.StatusCreated)
		var got struct{ ID uuid.UUID }
		r.Decode(t, &got)
		return got.ID
	}

	t.Run("a new episode gets a copy of its procedure's list, in order, retired items too", func(t *testing.T) {
		id := create(t, w.hip)
		rows := w.episodeInfo(t, id)
		if !slices.Equal(infoItems(rows), []uuid.UUID{w.leaflet, w.video, w.retired}) {
			t.Fatalf("items = %v", rows)
		}
		for _, r := range rows {
			if !r.FromProcedure || r.AddedBy != nil {
				t.Errorf("row = %+v", r)
			}
		}
	})
	t.Run("no procedure, no list", func(t *testing.T) {
		if rows := w.episodeInfo(t, create(t, uuid.Nil)); len(rows) != 0 {
			t.Errorf("items = %v", rows)
		}
	})
	t.Run("changing the procedure's list later leaves the episodes already made alone", func(t *testing.T) {
		id := create(t, w.hip)
		w.author.Do("PATCH", proceduresPath(w.a.Org, "/"+w.hip.String()), map[string]any{"infoContent": []any{w.video}}).Want(t, http.StatusNoContent)
		if got := infoItems(w.episodeInfo(t, id)); !slices.Equal(got, []uuid.UUID{w.leaflet, w.video, w.retired}) {
			t.Errorf("items = %v", got)
		}
		if got := infoItems(w.episodeInfo(t, create(t, w.hip))); !slices.Equal(got, []uuid.UUID{w.video}) {
			t.Errorf("a new episode's items = %v", got)
		}
		w.setProcedureInfo(t, w.hip, w.leaflet, w.video, w.retired)
	})
	t.Run("changing the episode's procedure swaps the copied items; a clinician's stay", func(t *testing.T) {
		id := create(t, w.hip)
		w.addInfo(t, id, w.platformLeaflet, false, 1) // a clinician's, also on the knee's list
		mine := w.contentItem(t, &w.a.Org, "leaflet", "Only for this patient", "active")
		w.addInfo(t, id, mine, false, 2)
		r := w.clinician.Do("PATCH", at(w.a, "/episodes/"+id.String()), map[string]any{"procedureId": w.knee, "revision": revisionOf(t, w.clinic, id),
			"anaesthetic": "", "consultant": ""})
		r.Want(t, http.StatusNoContent)
		rows := w.episodeInfo(t, id)
		if !slices.Equal(infoItems(rows), []uuid.UUID{w.video, w.platformLeaflet, mine}) || !rows[0].FromProcedure || rows[1].FromProcedure {
			t.Errorf("items = %+v", rows)
		}
		// Cleared: the copied items go; a clinician's stay.
		r = w.clinician.Do("PATCH", at(w.a, "/episodes/"+id.String()), map[string]any{"procedureId": "", "revision": revisionOf(t, w.clinic, id),
			"anaesthetic": "", "consultant": ""})
		r.Want(t, http.StatusNoContent)
		if got := infoItems(w.episodeInfo(t, id)); !slices.Equal(got, []uuid.UUID{w.platformLeaflet, mine}) {
			t.Errorf("items = %v", got)
		}
	})
	t.Run("other changes to the episode leave its list alone", func(t *testing.T) {
		id := create(t, w.hip)
		w.setProcedureInfo(t, w.hip, w.video)
		r := w.clinician.Do("PATCH", at(w.a, "/episodes/"+id.String()), map[string]any{"plannedDate": "2026-12-01", "revision": revisionOf(t, w.clinic, id),
			"anaesthetic": "General", "consultant": ""})
		r.Want(t, http.StatusNoContent)
		if got := infoItems(w.episodeInfo(t, id)); !slices.Equal(got, []uuid.UUID{w.leaflet, w.video, w.retired}) {
			t.Errorf("items = %v", got)
		}
	})
}

// staffInfo is GET …/episodes/{eid}/information as the tests read it.
type staffInfo struct {
	Items []struct {
		ID                  uuid.UUID
		Kind, Title, Status string
		URL                 string
		LengthSeconds       *int
		FromProcedure       bool
		AddedByName         *string
	}
	Library []struct {
		ID          uuid.UUID
		Kind, Title string
	}
}

func TestListEpisodeInformation(t *testing.T) {
	t.Parallel()
	w := newInfoWorld(t)
	w.addInfo(t, w.ep.ID, w.leaflet, true, 1)
	w.addInfo(t, w.ep.ID, w.retired, true, 2)
	w.addInfo(t, w.ep.ID, w.video, false, 1)
	w.addInfo(t, w.ep.ID, w.theirs, false, 2) // never another trust's, even if it got there
	get := func(c *apitest.Client, h apitest.Hospital) *apitest.Resp {
		return c.Do("GET", epPath(h, w.ep, "/information"), nil)
	}

	t.Run("happy path: the episode's items in order, retired ones named; the library's active items to add; audited", func(t *testing.T) {
		before := lastAuditID(t, w.clinic)
		r := get(w.clinician, w.a)
		r.Want(t, http.StatusOK)
		var got staffInfo
		r.Decode(t, &got)
		var ids []uuid.UUID
		for _, i := range got.Items {
			ids = append(ids, i.ID)
		}
		if !slices.Equal(ids, []uuid.UUID{w.leaflet, w.retired, w.video}) || got.Items[1].Status != "retired" || !got.Items[0].FromProcedure ||
			got.Items[2].FromProcedure || got.Items[2].LengthSeconds == nil || *got.Items[2].LengthSeconds != 125 {
			t.Errorf("items = %+v", got.Items)
		}
		var lib []uuid.UUID
		for _, i := range got.Library {
			lib = append(lib, i.ID)
		}
		if !slices.Contains(lib, w.video) || !slices.Contains(lib, w.leaflet) || !slices.Contains(lib, w.platformLeaflet) ||
			slices.Contains(lib, w.retired) || slices.Contains(lib, w.theirs) {
			t.Errorf("library = %+v", got.Library)
		}
		rows := auditRows(t, w.clinic, before)
		if len(rows) != 1 || rows[0].Action != "information.list" || !slices.Equal(rows[0].Episodes, []uuid.UUID{w.ep.ID}) {
			t.Errorf("audit = %+v", rows)
		}
	})
	t.Run("happy path: none is an empty list", func(t *testing.T) {
		r := w.clinician.Do("GET", epPath(w.a, w.fx.Episode(t, w.hq, w.cara), "/information"), nil)
		r.Want(t, http.StatusOK)
		if !strings.Contains(string(r.Body), `"items":[]`) {
			t.Errorf("body = %s", r.Body)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, get(w.anon, w.a), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, get(w.site2, w.a), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, get(w.otherClin, w.a), http.StatusNotFound)
		refused(t, get(w.otherClin, w.b), http.StatusNotFound)
	})
	t.Run("invalid input: not an episode", func(t *testing.T) {
		refused(t, w.clinician.Do("GET", at(w.a, "/episodes/abc/information"), nil), http.StatusNotFound)
		refused(t, w.clinician.Do("GET", at(w.a, "/episodes/"+uuid.NewString()+"/information"), nil), http.StatusNotFound)
	})
}

func TestAddEpisodeInformation(t *testing.T) {
	t.Parallel()
	w := newInfoWorld(t)
	post := func(c *apitest.Client, h apitest.Hospital, body any) *apitest.Resp {
		return c.Do("POST", epPath(h, w.ep, "/information"), body)
	}
	item := func(id any) map[string]any { return map[string]any{"contentId": id} }

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, post(w.anon, w.a, item(w.leaflet)), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, post(w.site2, w.a, item(w.leaflet)), http.StatusNotFound)
	})
	t.Run("another trust: not a member, by its own path, or its item", func(t *testing.T) {
		refused(t, post(w.otherClin, w.a, item(w.leaflet)), http.StatusNotFound)
		refused(t, post(w.otherClin, w.b, item(w.leaflet)), http.StatusNotFound)
		refused(t, post(w.clinician, w.a, item(w.theirs)), http.StatusBadRequest)
	})
	t.Run("invalid input", func(t *testing.T) {
		for _, bad := range []any{item("abc"), item(uuid.New()), item(w.retired), item(nil), map[string]any{}} {
			refused(t, post(w.clinician, w.a, bad), http.StatusBadRequest)
		}
		refused(t, w.clinician.Do("POST", at(w.a, "/episodes/abc/information"), item(w.leaflet)), http.StatusNotFound)
	})
	if rows := w.episodeInfo(t, w.ep.ID); len(rows) != 0 {
		t.Fatalf("refusals added %v", rows)
	}
	t.Run("happy path: added last, by the clinician; audited", func(t *testing.T) {
		w.addInfo(t, w.ep.ID, w.video, true, 1)
		before := lastAuditID(t, w.clinic)
		post(w.clinician, w.a, item(w.leaflet)).Want(t, http.StatusNoContent)
		post(w.clinician, w.a, item(w.platformLeaflet)).Want(t, http.StatusNoContent)
		rows := w.episodeInfo(t, w.ep.ID)
		if !slices.Equal(infoItems(rows), []uuid.UUID{w.video, w.leaflet, w.platformLeaflet}) || rows[1].FromProcedure || rows[1].AddedBy == nil ||
			*rows[1].AddedBy != w.cara.ID {
			t.Errorf("items = %+v", rows)
		}
		audit := auditRows(t, w.clinic, before)
		if len(audit) != 2 || audit[0].Action != "information.add" || !slices.Equal(audit[0].Episodes, []uuid.UUID{w.ep.ID}) {
			t.Errorf("audit = %+v", audit)
		}
		// In the General notes, by name.
		events := readEvents(t, w.clinic, w.ep.ID)
		if !slices.ContainsFunc(events, func(e eventRow) bool { return e.Text == "Information added: Fasting before your operation" }) {
			t.Errorf("events = %+v", events)
		}
	})
	t.Run("one it already has is refused", func(t *testing.T) {
		refused(t, post(w.clinician, w.a, item(w.leaflet)), http.StatusConflict)
		refused(t, post(w.clinician, w.a, item(w.video)), http.StatusConflict)
	})
	t.Run("a cancelled or archived episode can't be changed", func(t *testing.T) {
		for _, status := range []string{"cancelled", "archived"} {
			e := w.fx.Episode(t, w.hq, w.cara)
			if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET status = $2, archived_from = CASE WHEN $2 = 'archived' THEN status END WHERE id = $1`, e.ID, status); err != nil {
				t.Fatal(err)
			}
			refused(t, w.clinician.Do("POST", epPath(w.a, e, "/information"), item(w.leaflet)), http.StatusConflict)
		}
	})
}

func TestRemoveEpisodeInformation(t *testing.T) {
	t.Parallel()
	w := newInfoWorld(t)
	w.addInfo(t, w.ep.ID, w.video, true, 1)
	w.addInfo(t, w.ep.ID, w.leaflet, false, 1)
	del := func(c *apitest.Client, h apitest.Hospital, item uuid.UUID) *apitest.Resp {
		return c.Do("DELETE", epPath(h, w.ep, "/information/"+item.String()), nil)
	}

	t.Run("no sign-in", func(t *testing.T) {
		refused(t, del(w.anon, w.a, w.leaflet), http.StatusUnauthorized)
	})
	t.Run("role: every member is a clinician at least, so the case is a clinician at the trust's other hospital", func(t *testing.T) {
		refused(t, del(w.site2, w.a, w.leaflet), http.StatusNotFound)
	})
	t.Run("another trust", func(t *testing.T) {
		refused(t, del(w.otherClin, w.a, w.leaflet), http.StatusNotFound)
		refused(t, del(w.otherClin, w.b, w.leaflet), http.StatusNotFound)
	})
	t.Run("invalid input: not an item of the episode, or not an id", func(t *testing.T) {
		refused(t, del(w.clinician, w.a, w.platformLeaflet), http.StatusNotFound)
		refused(t, w.clinician.Do("DELETE", epPath(w.a, w.ep, "/information/abc"), nil), http.StatusNotFound)
	})
	if rows := w.episodeInfo(t, w.ep.ID); len(rows) != 2 {
		t.Fatalf("refusals removed some: %v", rows)
	}
	t.Run("happy path: a clinician's or the procedure's; audited and in the General notes", func(t *testing.T) {
		before := lastAuditID(t, w.clinic)
		del(w.clinician, w.a, w.leaflet).Want(t, http.StatusNoContent)
		del(w.clinician, w.a, w.video).Want(t, http.StatusNoContent)
		if rows := w.episodeInfo(t, w.ep.ID); len(rows) != 0 {
			t.Errorf("items = %v", rows)
		}
		audit := auditRows(t, w.clinic, before)
		if len(audit) != 2 || audit[0].Action != "information.remove" {
			t.Errorf("audit = %+v", audit)
		}
		events := readEvents(t, w.clinic, w.ep.ID)
		if !slices.ContainsFunc(events, func(e eventRow) bool { return e.Text == "Information removed: Your anaesthetic" }) {
			t.Errorf("events = %+v", events)
		}
	})
	t.Run("an archived episode can't be changed", func(t *testing.T) {
		e := w.fx.Episode(t, w.hq, w.cara)
		w.addInfo(t, e.ID, w.video, true, 1)
		if _, err := w.db.Exec(t.Context(), `UPDATE episodes SET status = 'archived', archived_from = status WHERE id = $1`, e.ID); err != nil {
			t.Fatal(err)
		}
		refused(t, w.clinician.Do("DELETE", epPath(w.a, e, "/information/"+w.video.String()), nil), http.StatusConflict)
	})
}

// patientInfo is GET /api/p/episodes/{eid}/information as the tests read it.
type patientInfo struct {
	Hospital, Phone, Email string
	Items                  []struct {
		ID               uuid.UUID
		Kind, Title, URL string
		LengthSeconds    *int
	}
}

func TestPatientInformation(t *testing.T) {
	t.Parallel()
	w := newInfoWorld(t)
	w.addInfo(t, w.ep.ID, w.leaflet, true, 1)
	w.addInfo(t, w.ep.ID, w.retired, true, 2)
	w.addInfo(t, w.ep.ID, w.video, false, 1)
	w.addInfo(t, w.ep.ID, w.theirs, false, 2) // never another trust's, even if it got there
	if _, err := w.db.Exec(t.Context(), `UPDATE hospitals SET settings = settings || '{"contact.phone": "0100 555 0123", "contact.email": "poa@hospital-a.example"}'
		WHERE id = $1`, w.a.ID); err != nil {
		t.Fatal(err)
	}
	other := w.fx.Episode(t, w.hq, w.cara) // another person's, with the same kind of list
	w.addInfo(t, other.ID, w.platformLeaflet, false, 1)
	p := w.anon.AsPatient(t, w.ep.Link, w.ep.DateOfBirth)
	get := func(c *apitest.Client, e apitest.Episode) *apitest.Resp {
		return c.Do("GET", patientPath(e, "/information"), nil)
	}

	t.Run("happy path: the active items in order, the hospital's phone and email; nothing else; audited", func(t *testing.T) {
		before := lastAuditID(t, w.clinic)
		r := get(p, w.ep)
		r.Want(t, http.StatusOK)
		if r.Header("Cache-Control") != "no-store" {
			t.Errorf("Cache-Control %q", r.Header("Cache-Control"))
		}
		var got patientInfo
		r.Decode(t, &got)
		if len(got.Items) != 2 || got.Items[0].ID != w.leaflet || got.Items[1].ID != w.video || got.Items[1].Kind != "video" ||
			got.Items[1].LengthSeconds == nil || *got.Items[1].LengthSeconds != 125 || !strings.HasPrefix(got.Items[0].URL, "https://") {
			t.Errorf("items = %+v", got.Items)
		}
		if got.Phone != "0100 555 0123" || got.Email != "poa@hospital-a.example" || got.Hospital == "" {
			t.Errorf("contact = %+v", got)
		}
		for _, hidden := range []string{"Old leaflet", "Trust B", "retired", "fromProcedure", "addedBy", w.ep.PatientID.String()} {
			if strings.Contains(string(r.Body), hidden) {
				t.Errorf("the patient's information holds %q: %s", hidden, r.Body)
			}
		}
		rows := auditRows(t, w.clinic, before)
		if len(rows) != 1 || rows[0].Action != "link.information" || rows[0].Actor != "patient" || !slices.Equal(rows[0].Episodes, []uuid.UUID{w.ep.ID}) {
			t.Errorf("audit = %+v", rows)
		}
	})
	t.Run("happy path: none, and no contact details set, is empty", func(t *testing.T) {
		mine := w.fx.EpisodeAt(t, w.hq, w.a2, w.cara)
		r := get(w.anon.AsPatient(t, mine.Link, mine.DateOfBirth), mine)
		r.Want(t, http.StatusOK)
		var got patientInfo
		r.Decode(t, &got)
		if len(got.Items) != 0 || got.Phone != "" || got.Email != "" {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("a retired item brought back shows again", func(t *testing.T) {
		if _, err := w.db.Exec(t.Context(), `UPDATE content_items SET status = 'active' WHERE id = $1`, w.retired); err != nil {
			t.Fatal(err)
		}
		defer w.db.Exec(t.Context(), `UPDATE content_items SET status = 'retired' WHERE id = $1`, w.retired)
		var got patientInfo
		r := get(p, w.ep)
		r.Want(t, http.StatusOK)
		r.Decode(t, &got)
		if len(got.Items) != 3 || got.Items[1].ID != w.retired {
			t.Errorf("items = %+v", got.Items)
		}
	})
	t.Run("no sign-in", func(t *testing.T) {
		refused(t, get(w.anon, w.ep), http.StatusUnauthorized)
	})
	t.Run("role: staff signed in aren't the patient", func(t *testing.T) {
		refused(t, get(w.clinician, w.ep), http.StatusUnauthorized)
	})
	t.Run("another person's episode, at this trust or another", func(t *testing.T) {
		refused(t, get(p, other), http.StatusNotFound)
		theirs := w.fx.Episode(t, w.fx.PublishedHQ(t, w.b, w.bea, "basic.json"), w.bob)
		refused(t, get(p, theirs), http.StatusNotFound)
	})
	t.Run("invalid input: not an episode", func(t *testing.T) {
		refused(t, p.Do("GET", "/api/p/episodes/abc/information", nil), http.StatusNotFound)
		refused(t, p.Do("GET", "/api/p/episodes/"+uuid.NewString()+"/information", nil), http.StatusNotFound)
	})
}
