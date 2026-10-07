package httpapi

import (
	"github.com/google/uuid"

	"github.com/rlamba89/poised/apps/api/internal/db"
)

// forOrg returns the store that holds one trust's patient data (saas-requirements.md §13,
// rule 1). Every trust is in the one London database today, so it is always s.q; a trust in
// another data region would get its own pool here, which is why staff handlers that touch
// patient data go through it. The patient link routes (patient.go) and event still use s.q:
// they find the trust from the link, which comes with patient sign-in (plan C2).
func (s *server) forOrg(orgID uuid.UUID) db.Querier {
	return s.q
}
