// Package role is the staff role ladder (decision A-5, saas-requirements.md §2):
// clinician < super_clinician < admin. Each role can do everything the role below it can,
// so a permission check is one comparison. Roles limit what people can do; every member of
// a hospital can see all of its data.
package role

// Role is one rung of the ladder, stored as text in memberships.role.
type Role string

const (
	Clinician      Role = "clinician"       // patients, episodes, validating HQs
	SuperClinician Role = "super_clinician" // also authors, reviews and publishes HQs
	Admin          Role = "admin"           // also settings, staff and roles
)

var rank = map[Role]int{Clinician: 1, SuperClinician: 2, Admin: 3}

// AtLeast reports whether r is want or above it. A role off the ladder is never enough.
func (r Role) AtLeast(want Role) bool {
	return rank[r] > 0 && rank[r] >= rank[want]
}

// Highest returns the highest role on the ladder among rs, or "" when there is none.
func Highest(rs ...Role) Role {
	var best Role
	for _, r := range rs {
		if rank[r] > rank[best] {
			best = r
		}
	}
	return best
}
