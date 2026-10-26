package httpapi

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
)

// The patient's Information for an episode (plan C5, session S36; INF-04): videos and leaflets from
// the content library (S33), and the hospital's phone and email.
//   - The procedure's list (procedures.info_content, set on the procedure: procedures.go) is copied
//     onto the episode when it's made (S36 answers 2), and swapped when its procedure is changed.
//   - GET, POST and DELETE /api/o/{oid}/h/{hid}/episodes/{eid}/information[/{cid}]: the episode's
//     list for anyone at the hospital, who may add the library's active items to it or remove any.
//     Each is audited (patient data), and a change is in the General notes.
//   - GET /api/p/episodes/{eid}/information: the patient's, for their own episodes only, with the
//     active items only (a retired item disappears from every patient's Information).
// Every read joins only Preopsis's items and the episode's own trust's, so another trust's item
// never shows even if one were in a list.

const (
	infoNotOnEpisode = "That video or leaflet isn't in this episode's Information."
	infoChoose       = "Choose a video or leaflet from the content library."
)

// episodeInfoItem is one item of an episode's list, as staff see it.
type episodeInfoItem struct {
	ID            uuid.UUID `json:"id"`
	Kind          string    `json:"kind"`
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	LengthSeconds *int32    `json:"lengthSeconds"`
	Status        string    `json:"status"`        // a retired one isn't shown to the patient
	FromProcedure bool      `json:"fromProcedure"` // copied from the procedure, or added for this episode
	AddedByName   *string   `json:"addedByName"`
}

// listEpisodeInfo is the episode's list, and the library's active items a clinician may add.
func (s *server) listEpisodeInfo(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	ctx, org := r.Context(), uuid.NullUUID{UUID: orgID(r), Valid: true}
	rows, err := s.q.ListEpisodeInfo(ctx, db.ListEpisodeInfoParams{EpisodeID: e.ID, OrgID: org})
	if err != nil {
		serverError(w, "list information", err)
		return
	}
	library, err := s.q.SearchContent(ctx, db.SearchContentParams{OrgID: org})
	if err != nil {
		serverError(w, "content library", err)
		return
	}
	items := []episodeInfoItem{}
	for _, i := range rows {
		items = append(items, episodeInfoItem{ID: i.ID, Kind: i.Kind, Title: i.Title, URL: i.Url, LengthSeconds: i.LengthSeconds,
			Status: i.Status, FromProcedure: i.FromProcedure, AddedByName: i.AddedByName})
	}
	type offered struct {
		ID            uuid.UUID `json:"id"`
		Kind          string    `json:"kind"`
		Title         string    `json:"title"`
		LengthSeconds *int32    `json:"lengthSeconds"`
		Tags          []string  `json:"tags"`
	}
	lib := []offered{}
	for _, c := range library {
		lib = append(lib, offered{ID: c.ID, Kind: c.Kind, Title: c.Title, LengthSeconds: c.LengthSeconds, Tags: c.Tags})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "library": lib})
}

// addEpisodeInfo adds one of the library's active items (Preopsis's or the trust's) to the episode's
// list, last; {"contentId": id}.
func (s *server) addEpisodeInfo(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	var in struct {
		ContentID string `json:"contentId"`
	}
	if !readJSON(w, r, 1<<10, &in) {
		return
	}
	id, err := uuid.Parse(in.ContentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, infoChoose)
		return
	}
	if msg := readOnly(e.Status); msg != "" {
		writeError(w, http.StatusConflict, msg)
		return
	}
	ctx := r.Context()
	found, err := s.q.TrustContentItems(ctx, db.TrustContentItemsParams{Ids: []uuid.UUID{id}, OrgID: uuid.NullUUID{UUID: orgID(r), Valid: true}})
	if err != nil {
		serverError(w, "content item", err)
		return
	}
	if len(found) == 0 || found[0].Status != "active" {
		writeError(w, http.StatusBadRequest, infoChoose)
		return
	}
	n, err := s.q.AddEpisodeInfo(ctx, db.AddEpisodeInfoParams{EpisodeID: e.ID, ContentItemID: id, AddedBy: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true}})
	if err != nil {
		serverError(w, "add information", err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusConflict, "That video or leaflet is already in this episode's Information.")
		return
	}
	s.event(r, e.ID, "Information added: "+found[0].Title, currentUser(r).ID)
	w.WriteHeader(http.StatusNoContent)
}

// removeEpisodeInfo removes an item from the episode's list, whether it came from the procedure or
// a clinician added it.
func (s *server) removeEpisodeInfo(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "cid", infoNotOnEpisode)
	if !ok {
		return
	}
	if msg := readOnly(e.Status); msg != "" {
		writeError(w, http.StatusConflict, msg)
		return
	}
	title, err := s.q.RemoveEpisodeInfo(r.Context(), db.RemoveEpisodeInfoParams{EpisodeID: e.ID, ContentItemID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, infoNotOnEpisode)
		return
	}
	if err != nil {
		serverError(w, "remove information", err)
		return
	}
	s.event(r, e.ID, "Information removed: "+title, currentUser(r).ID)
	w.WriteHeader(http.StatusNoContent)
}

// patientInfoItem is one item as the patient gets it: what to show, nothing about who added it.
type patientInfoItem struct {
	ID            uuid.UUID `json:"id"`
	Kind          string    `json:"kind"`
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	LengthSeconds *int32    `json:"lengthSeconds"`
}

// patientInfo is GET /api/p/episodes/{eid}/information: the episode's active items, and the
// hospital's phone and email for patients (its settings, S10; "" when not set).
func (s *server) patientInfo(w http.ResponseWriter, r *http.Request) {
	f, ok := s.loadPatientForm(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	rows, err := s.q.ListEpisodeInfo(ctx, db.ListEpisodeInfoParams{EpisodeID: f.ID, OrgID: uuid.NullUUID{UUID: f.OrgID, Valid: true}})
	if err != nil {
		serverError(w, "list information", err)
		return
	}
	settings, err := s.hospitalSettings(ctx, f.OrgID, f.HospitalID)
	if err != nil {
		serverError(w, "hospital settings", err)
		return
	}
	items := []patientInfoItem{}
	for _, i := range rows {
		if i.Status == "active" {
			items = append(items, patientInfoItem{ID: i.ID, Kind: i.Kind, Title: i.Title, URL: i.Url, LengthSeconds: i.LengthSeconds})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"hospital": f.HospitalName,
		"phone":    text(settings, "contact.phone"),
		"email":    text(settings, "contact.email"),
		"items":    items,
	})
}
