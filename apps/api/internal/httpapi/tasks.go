package httpapi

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/episode"
)

// An episode's tasks (plan C3, session S21; TSK-01):
//
//   - GET …/episodes/{eid}/tasks: the tasks, oldest first, with who added, completed and
//     re-opened each;
//   - POST …/episodes/{eid}/tasks {title, team, revision}: a new task for a team (S21 answers 1:
//     nurse, anaesthetist, admin);
//   - PATCH …/episodes/{eid}/tasks/{tid} {done, revision}: complete it (true) or re-open it (false).
//
// Anyone at the hospital may add, complete or re-open any task. A change to the tasks is a change
// to the episode: it sends the revision the page loaded and bumps it (S20), so a stale tab is told
// rather than acting on tasks that have changed, and it writes a General notes line. A cancelled
// episode is read-only (S20): its tasks can't be changed.

const taskNotFound = "Task not found."

func (s *server) listTasks(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	tasks, err := s.q.ListTasks(r.Context(), e.ID)
	if err != nil {
		serverError(w, "list tasks", err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *server) createTask(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	var in struct {
		Title    string `json:"title"`
		Team     string `json:"team"`
		Revision *int32 `json:"revision"`
	}
	if !readJSON(w, r, 4<<10, &in) {
		return
	}
	title, msg := episode.TaskInput(in.Title, in.Team)
	if msg == "" && (in.Revision == nil || *in.Revision < 0) {
		msg = "The request could not be read."
	}
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	var id uuid.UUID
	note := "Task added for the " + in.Team + " team: " + title
	if !s.changeTasks(w, r, e, *in.Revision, note, func(q *db.Queries) (bool, error) {
		var err error
		id, err = q.CreateTask(r.Context(), db.CreateTaskParams{EpisodeID: e.ID, Title: title, Team: in.Team,
			CreatedBy: uuid.NullUUID{UUID: currentUser(r).ID, Valid: true}})
		return true, err
	}) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]uuid.UUID{"id": id})
}

func (s *server) updateTask(w http.ResponseWriter, r *http.Request) {
	e, ok := s.loadEpisode(w, r)
	if !ok {
		return
	}
	tid, ok := pathID(w, r, "tid", taskNotFound)
	if !ok {
		return
	}
	var in struct {
		Done     *bool  `json:"done"`
		Revision *int32 `json:"revision"`
	}
	if !readJSON(w, r, 1<<10, &in) {
		return
	}
	if in.Done == nil || in.Revision == nil || *in.Revision < 0 {
		writeError(w, http.StatusBadRequest, "The request could not be read.")
		return
	}
	task, err := s.q.GetTask(r.Context(), db.GetTaskParams{ID: tid, EpisodeID: e.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, taskNotFound)
		return
	}
	if err != nil {
		serverError(w, "get task", err)
		return
	}
	who := uuid.NullUUID{UUID: currentUser(r).ID, Valid: true}
	note, already := "Task completed: "+task.Title, "This task is already done."
	change := func(q *db.Queries) (int64, error) {
		return q.CompleteTask(r.Context(), db.CompleteTaskParams{ID: tid, DoneBy: who})
	}
	if !*in.Done {
		note, already = "Task re-opened: "+task.Title, "This task is already open."
		change = func(q *db.Queries) (int64, error) {
			return q.ReopenTask(r.Context(), db.ReopenTaskParams{ID: tid, ReopenedBy: who})
		}
	}
	if !s.changeTasks(w, r, e, *in.Revision, note, func(q *db.Queries) (bool, error) {
		n, err := change(q)
		if err == nil && n == 0 {
			writeError(w, http.StatusConflict, already)
		}
		return n > 0, err
	}) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// changeTasks makes one change to the episode's tasks in a transaction: the episode is locked, its
// revision checked (409 stale) and that it isn't cancelled (409), then change runs, then the
// revision is bumped and the General notes line written. change returns false (having answered
// itself) to stop. changeTasks answers every refusal and error; true means the change was made.
func (s *server) changeTasks(w http.ResponseWriter, r *http.Request, e db.GetEpisodeRow, revision int32, note string,
	change func(q *db.Queries) (bool, error)) bool {
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		serverError(w, "begin", err)
		return false
	}
	defer tx.Rollback(ctx)
	q := db.New(tx)
	now, err := q.LockEpisodeStatus(ctx, e.ID)
	if err != nil {
		serverError(w, "lock episode", err)
		return false
	}
	if now.Revision != revision {
		writeStale(w)
		return false
	}
	if now.Status == episode.Cancelled {
		writeError(w, http.StatusConflict, cancelledMsg)
		return false
	}
	ok, err := change(q)
	if err != nil {
		serverError(w, "change task", err)
		return false
	}
	if !ok {
		return false
	}
	if err := q.BumpEpisodeRevision(ctx, e.ID); err != nil {
		serverError(w, "revision", err)
		return false
	}
	user := uuid.NullUUID{UUID: currentUser(r).ID, Valid: true}
	if err := q.AddEpisodeEvent(ctx, db.AddEpisodeEventParams{EpisodeID: e.ID, Kind: "event", Text: note, UserID: user}); err != nil {
		serverError(w, "note", err)
		return false
	}
	if err := tx.Commit(ctx); err != nil {
		serverError(w, "commit", err)
		return false
	}
	return true
}
