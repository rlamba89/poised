package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/observation"
	"github.com/rlamba89/poised/apps/api/internal/settings"
)

// Observations (plan C7, session S46; OBS-01–03): a clinician's readings on an episode.
//
//   - GET  …/episodes/{eid}/observations: every reading, oldest first, with the kinds, the
//     measures and the ranges that apply (the trust's own, else the platform's; S46 answers 1),
//     so the panel can warn outside the usual range before it saves;
//   - POST …/episodes/{eid}/observations {kind, value, value2, note, takenAt}: a new reading. A
//     value outside the hard limits, or a malformed reading, is a 400 (internal/observation);
//   - POST …/episodes/{eid}/observations/{obsid}/correct {value, value2, note, takenAt, reason}:
//     a new reading of the same kind replaces it (the old one is kept, superseded). takenAt left
//     out keeps the original's time;
//   - POST …/episodes/{eid}/observations/{obsid}/entered-in-error {reason}: it no longer counts,
//     and stays in the history.
//
// A reading is never changed or deleted (S46 answers 2; migration 00037 refuses it too), and
// only a reading that still counts can be corrected or marked (409). Any clinician at the hospital
// may record; a cancelled or archived episode takes nothing (409); with the hospital's
// feature.observations off, every route is a 403. Readings don't change the episode's revision or
// write General notes lines: they have their own history (who, when, why), and the panel saves
// each on its own, so a reading never makes another tab's episode edit stale.

// maxReadingBody fits the longest note in any script, with a reason.
const maxReadingBody = 16 << 10

const (
	observationNotFound = "Reading not found."
	obsNotCountingMsg   = "This reading has already been corrected or marked entered in error."
	observationsOff     = "Observations aren't turned on at this hospital."
)

type observationsView struct {
	Readings []db.ListObservationsRow `json:"readings"`
	Kinds    []observation.Kind       `json:"kinds"`
	Measures []observation.Measure    `json:"measures"`
	Ranges   observation.Ranges       `json:"ranges"`
	MaxNote  int                      `json:"maxNote"`
}

// observationsOn says whether the hospital has observations; the episode's GET sends it, so a
// hospital without them doesn't call the routes (and write a refused audit row) on each page open.
func observationsOn(v settings.Values) (bool, error) {
	var on bool
	err := settings.Get(v, "feature.observations", &on)
	return on, err
}

// observationRanges are the ranges at the hospital: the trust's own, else the platform's. With
// observations off there it answers 403 itself and returns false.
func (s *server) observationRanges(w http.ResponseWriter, r *http.Request) (observation.Ranges, bool) {
	v, err := s.hospitalSettings(r.Context(), orgID(r), hospitalID(r))
	if err != nil {
		serverError(w, "settings", err)
		return nil, false
	}
	on, err := observationsOn(v)
	if err != nil {
		serverError(w, "settings", err)
		return nil, false
	}
	if !on {
		writeError(w, http.StatusForbidden, observationsOff)
		return nil, false
	}
	var trust observation.Ranges
	if err := settings.Get(v, "observations.ranges", &trust); err != nil {
		serverError(w, "settings", err)
		return nil, false
	}
	return observation.Effective(trust), true
}

func (s *server) listObservations(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	ranges, ok := s.observationRanges(w, r)
	if !ok {
		return
	}
	rows, err := s.q.ListObservations(r.Context(), e.ID)
	if err != nil {
		serverError(w, "list observations", err)
		return
	}
	if rows == nil {
		rows = []db.ListObservationsRow{}
	}
	writeJSON(w, http.StatusOK, observationsView{Readings: rows, Kinds: observation.Kinds, Measures: observation.Measures, Ranges: ranges, MaxNote: observation.MaxNote})
}

// readingIn is a reading as the panel sends it.
type readingIn struct {
	Kind    string   `json:"kind"`
	Value   *float64 `json:"value"`
	Value2  *float64 `json:"value2"`
	Note    string   `json:"note"`
	TakenAt *string  `json:"takenAt"`
	Reason  string   `json:"reason"`
}

// reading checks in against the hard limits; "" means it's fine. A takenAt that isn't a time is
// "could not be read"; none at all is the observation package's "When was it measured?".
func (in readingIn) reading(ranges observation.Ranges, now, born time.Time) (observation.Reading, string) {
	rd := observation.Reading{Kind: in.Kind, Value: in.Value, Value2: in.Value2, Note: in.Note}
	if in.TakenAt != nil {
		t, err := time.Parse(time.RFC3339, *in.TakenAt)
		if err != nil {
			return rd, "The request could not be read."
		}
		rd.TakenAt = t
	}
	return observation.Check(rd, ranges, now, born)
}

func (s *server) addObservation(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	ranges, ok := s.observationRanges(w, r)
	if !ok {
		return
	}
	var in readingIn
	if !readJSON(w, r, maxReadingBody, &in) {
		return
	}
	rd, msg := in.reading(ranges, s.clock.Now(), e.DateOfBirth.Time)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	var id uuid.UUID
	if !s.lockedObservations(w, r, e.ID, func(q *db.Queries) (bool, error) {
		var err error
		id, err = q.AddObservation(r.Context(), addParams(e.ID, currentUser(r).ID, rd, nil, nil))
		return true, err
	}) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
}

func (s *server) correctObservation(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	ranges, ok := s.observationRanges(w, r)
	if !ok {
		return
	}
	old, ok := s.loadObservation(w, r, e.ID)
	if !ok {
		return
	}
	var in readingIn
	if !readJSON(w, r, maxReadingBody, &in) {
		return
	}
	reason, msg := optionalReason(in.Reason)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	in.Kind = old.Kind
	if in.TakenAt == nil {
		t := old.TakenAt.Format(time.RFC3339Nano)
		in.TakenAt = &t
	}
	rd, msg := in.reading(ranges, s.clock.Now(), e.DateOfBirth.Time)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if reason == "" {
		writeError(w, http.StatusUnprocessableEntity, "Say why the reading is corrected.")
		return
	}
	var id uuid.UUID
	if !s.lockedObservations(w, r, e.ID, func(q *db.Queries) (bool, error) {
		// Read again under the lock: another correction may have won since loadObservation.
		cur, err := q.GetObservation(r.Context(), db.GetObservationParams{ID: old.ID, EpisodeID: e.ID})
		if err != nil {
			return false, err
		}
		if cur.SupersededBy.Valid || cur.ErrorAt.Valid {
			writeError(w, http.StatusConflict, obsNotCountingMsg)
			return false, nil
		}
		id, err = q.AddObservation(r.Context(), addParams(e.ID, currentUser(r).ID, rd, &old.ID, &reason))
		if err != nil {
			return false, err
		}
		_, err = q.SupersedeObservation(r.Context(), db.SupersedeObservationParams{By: uuid.NullUUID{UUID: id, Valid: true}, ID: old.ID, EpisodeID: e.ID})
		return err == nil, err
	}) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
}

func (s *server) markObservationInError(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	if _, ok := s.observationRanges(w, r); !ok {
		return
	}
	old, ok := s.loadObservation(w, r, e.ID)
	if !ok {
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if !readJSON(w, r, 4<<10, &in) {
		return
	}
	reason, msg := optionalReason(in.Reason)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if reason == "" {
		writeError(w, http.StatusUnprocessableEntity, "Say why the reading was entered in error.")
		return
	}
	if !s.lockedObservations(w, r, e.ID, func(q *db.Queries) (bool, error) {
		n, err := q.MarkObservationInError(r.Context(), db.MarkObservationInErrorParams{
			UserID: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true}, Reason: reason, ID: old.ID, EpisodeID: e.ID})
		if err == nil && n == 0 {
			writeError(w, http.StatusConflict, obsNotCountingMsg)
		}
		return n > 0, err
	}) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// loadObservation fetches {obsid} on the episode, answering 404 itself, and 409 when it no longer
// counts (corrected or entered in error).
func (s *server) loadObservation(w http.ResponseWriter, r *http.Request, episodeID uuid.UUID) (db.GetObservationRow, bool) {
	id, ok := pathID(w, r, "obsid", observationNotFound)
	if !ok {
		return db.GetObservationRow{}, false
	}
	o, err := s.q.GetObservation(r.Context(), db.GetObservationParams{ID: id, EpisodeID: episodeID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, observationNotFound)
		return o, false
	case err != nil:
		serverError(w, "get observation", err)
		return o, false
	case o.SupersededBy.Valid || o.ErrorAt.Valid:
		writeError(w, http.StatusConflict, obsNotCountingMsg)
		return o, false
	}
	return o, true
}

// lockedObservations runs change in a transaction holding the episode's lock, after checking it
// isn't cancelled or archived (409). change returns false (having answered itself) to stop.
func (s *server) lockedObservations(w http.ResponseWriter, r *http.Request, episodeID uuid.UUID, change func(q *db.Queries) (bool, error)) bool {
	return s.inTx(w, r, func(q *db.Queries) bool {
		now, err := q.LockEpisodeStatus(r.Context(), episodeID)
		if err != nil {
			serverError(w, "lock episode", err)
			return false
		}
		if msg := readOnly(now.Status); msg != "" {
			writeError(w, http.StatusConflict, msg)
			return false
		}
		ok, err := change(q)
		if err != nil {
			serverError(w, "observation", err)
			return false
		}
		return ok
	})
}

// addParams is a checked reading as a row: its kind's stored unit for a measured one.
func addParams(episodeID, userID uuid.UUID, rd observation.Reading, corrects *uuid.UUID, reason *string) db.AddObservationParams {
	p := db.AddObservationParams{EpisodeID: episodeID, Kind: rd.Kind, Value: rd.Value, Value2: rd.Value2,
		TakenAt: rd.TakenAt, UserID: userID, Reason: reason}
	if corrects != nil {
		p.Corrects = uuid.NullUUID{UUID: *corrects, Valid: true}
	}
	if k, _ := observation.KindOf(rd.Kind); k.Note {
		p.Note = &rd.Note
	} else {
		m, _ := observation.MeasureOf(k.Measures[0])
		p.Unit = &m.Unit
	}
	return p
}
