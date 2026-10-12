// Package role is the staff role ladder (A-5): clinician < super clinician < admin. Each role
// can do everything the one below it can, so a permission check is one comparison. Everyone
// in a hospital sees all its data; roles only limit what people can do.
package role

// Role is a rung on the ladder. The zero value, None, can do nothing.
type Role int

const (
	None Role = iota
	// Clinician: patients and episodes, validating HQs.
	Clinician
	// SuperClinician: also authors, reviews and publishes HQs.
	SuperClinician
	// Admin: also settings, staff and roles (S09, S10).
	Admin
)

var names = map[Role]string{None: "none", Clinician: "clinician", SuperClinician: "super_clinician", Admin: "admin"}

// String is the role as stored in memberships.role and sent to the browser.
func (r Role) String() string { return names[r] }

// AtLeast reports whether r can do what need can.
func (r Role) AtLeast(need Role) bool { return r >= need && r != None }

// Parse reads a stored role. Anything else, the old roles included, is None.
func Parse(s string) (Role, bool) {
	for r, name := range names {
		if r != None && name == s {
			return r, true
		}
	}
	return None, false
}

// Highest is the highest of the stored roles: someone with a trust-wide role and a role at one
// hospital has the higher of the two there. Unknown roles count for nothing.
func Highest(stored ...string) Role {
	best := None
	for _, s := range stored {
		if r, _ := Parse(s); r > best {
			best = r
		}
	}
	return best
}
