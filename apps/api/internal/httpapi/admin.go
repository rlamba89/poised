package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/rlamba89/poised/apps/api/internal/db"
	"github.com/rlamba89/poised/apps/api/internal/role"
	"github.com/rlamba89/poised/apps/api/internal/settings"
)

// The superadmin console (plan C1, session S10; ORG-01…05, A-21, OPS-06): /api/admin/…, for
// platform admins only. They create trusts and hospitals, invite a trust's first admin (S09's
// invite), suspend and re-activate, and set the opt-ins.
//
// Nothing here returns patient data (OPS-06): only trusts, hospitals and their admins. A platform
// admin has no memberships, so the trust and hospital routes are 404 for them.

// alreadyTrustMember is the console's refusal to invite someone who already has whole-trust
// access: the console can't change it (S10 test report 1).
const alreadyTrustMember = "%s already has access to the whole trust. The trust's admins can change their role on the Staff screen."

const (
	notPlatformAdmin = "Only Poised's platform admins can do this."
	orgNotFound      = "Trust not found."
)

// PlatformAdmins says whether a user is a platform admin. *db.Queries satisfies it.
type PlatformAdmins interface {
	IsPlatformAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
}

// requirePlatformAdmin answers 403 unless the caller is a platform admin, read on every request,
// so removing the row takes effect at once. It must run inside requireUser.
func requirePlatformAdmin(q PlatformAdmins, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, err := q.IsPlatformAdmin(r.Context(), currentUser(r).ID)
		if err != nil {
			serverError(w, "platform admin", err)
			return
		}
		if !ok {
			writeError(w, http.StatusForbidden, notPlatformAdmin)
			return
		}
		next(w, r)
	}
}

// optInDefs are the features and licensed instruments (ORG-05), in the order screens show them.
func optInDefs() []settings.Def {
	var out []settings.Def
	for _, d := range settings.Defs {
		if d.OptIn {
			out = append(out, d)
		}
	}
	return out
}

// adminOptInDefs lists the opt-ins for the New trust and Add hospital forms.
func (s *server) adminOptInDefs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, optInDefs())
}

// adminOrgRow is a trust in the platform admin's list.
type adminOrgRow struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Code          string    `json:"code"`
	Status        string    `json:"status"`
	HospitalCount int       `json:"hospitalCount"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (s *server) adminListOrgs(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListOrgsForAdmin(r.Context())
	if err != nil {
		serverError(w, "list trusts", err)
		return
	}
	out := make([]adminOrgRow, len(rows))
	for i, o := range rows {
		out[i] = adminOrgRow{ID: o.ID, Name: o.Name, Code: o.Code, Status: o.Status, HospitalCount: int(o.HospitalCount), CreatedAt: o.CreatedAt}
	}
	writeJSON(w, http.StatusOK, out)
}

// adminHospital is a hospital as the platform admin sees it. Phone, Email and LogoURL are the
// hospital's own settings ("" when it follows the trust).
type adminHospital struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Address         string          `json:"address"`
	TimeZone        string          `json:"timeZone"`    // its own, or the deployment's (A-20)
	OwnTimeZone     *string         `json:"ownTimeZone"` // null: the deployment's
	Status          string          `json:"status"`
	Phone           string          `json:"phone"`
	Email           string          `json:"email"`
	LogoURL         string          `json:"logoUrl"`
	OptIns          map[string]bool `json:"optIns"`          // set for this hospital
	EffectiveOptIns map[string]bool `json:"effectiveOptIns"` // what applies there
}

// adminAdmin is one of a trust's whole-trust admins.
type adminAdmin struct {
	UserID          uuid.UUID  `json:"userId"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	Status          string     `json:"status"` // active, invited or expired, as on the Staff screen
	InviteExpiresAt *time.Time `json:"inviteExpiresAt"`
}

// adminTrust is one trust, its hospitals and admins, as the platform admin sees it.
type adminTrust struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Code            string          `json:"code"`
	Status          string          `json:"status"`
	ContactName     string          `json:"contactName"`
	ContactEmail    string          `json:"contactEmail"`
	ContactPhone    string          `json:"contactPhone"`
	LogoURL         string          `json:"logoUrl"`
	CreatedAt       time.Time       `json:"createdAt"`
	OptIns          map[string]bool `json:"optIns"`          // set for the trust
	EffectiveOptIns map[string]bool `json:"effectiveOptIns"` // the trust's default for its hospitals
	Hospitals       []adminHospital `json:"hospitals"`
	Admins          []adminAdmin    `json:"admins"`
	OptInDefs       []settings.Def  `json:"optInDefs"`
}

// optIns picks the opt-ins out of settings values.
func optIns(v settings.Values) map[string]bool {
	out := map[string]bool{}
	for k, raw := range v {
		if settings.IsOptIn(k) {
			var on bool
			_ = json.Unmarshal(raw, &on)
			out[k] = on
		}
	}
	return out
}

// text is a string setting's value, or "".
func text(v settings.Values, key string) string {
	var s string
	_ = json.Unmarshal(v[key], &s)
	return s
}

// adminTrust reads one trust for the platform admin; pgx.ErrNoRows when there's no such trust.
func (s *server) adminTrust(ctx context.Context, q db.Querier, org uuid.UUID) (adminTrust, error) {
	o, err := q.GetOrgForAdmin(ctx, org)
	if err != nil {
		return adminTrust{}, err
	}
	orgValues, err := settings.Decode(o.Settings)
	if err != nil {
		return adminTrust{}, fmt.Errorf("trust settings: %w", err)
	}
	platform := settings.Platform()
	out := adminTrust{
		ID: o.ID, Name: o.Name, Code: o.Code, Status: o.Status, CreatedAt: o.CreatedAt,
		ContactName: o.ContactName, ContactEmail: o.ContactEmail, ContactPhone: o.ContactPhone,
		LogoURL: text(orgValues, "branding.logoUrl"), OptIns: optIns(orgValues),
		EffectiveOptIns: optIns(settings.Effective(platform, orgValues, nil)),
		Hospitals:       []adminHospital{}, Admins: []adminAdmin{},
	}
	out.OptInDefs = optInDefs()
	hospitals, err := q.ListOrgHospitals(ctx, org)
	if err != nil {
		return adminTrust{}, err
	}
	for _, h := range hospitals {
		own, err := settings.Decode(h.Settings)
		if err != nil {
			return adminTrust{}, fmt.Errorf("hospital settings: %w", err)
		}
		tz := s.cfg.Country.TimeZone
		if h.Timezone != nil {
			tz = *h.Timezone
		}
		out.Hospitals = append(out.Hospitals, adminHospital{
			ID: h.ID, Name: h.Name, Address: h.Address, TimeZone: tz, OwnTimeZone: h.Timezone, Status: h.Status,
			Phone: text(own, "contact.phone"), Email: text(own, "contact.email"), LogoURL: text(own, "branding.logoUrl"),
			OptIns: optIns(own), EffectiveOptIns: optIns(settings.Effective(platform, orgValues, own)),
		})
	}
	admins, err := q.ListTrustAdmins(ctx, org)
	if err != nil {
		return adminTrust{}, err
	}
	for _, a := range admins {
		row := adminAdmin{UserID: a.UserID, Name: a.Name, Email: a.Email}
		row.Status, row.InviteExpiresAt = s.inviteStatus(a.InviteExpiresAt)
		out.Admins = append(out.Admins, row)
	}
	return out, nil
}

// writeAdminTrust answers with the trust, read after a change.
func (s *server) writeAdminTrust(w http.ResponseWriter, r *http.Request, org uuid.UUID, status int, existing *bool) {
	t, err := s.adminTrust(r.Context(), s.q, org)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, orgNotFound)
		return
	}
	if err != nil {
		serverError(w, "read trust", err)
		return
	}
	if existing == nil {
		writeJSON(w, status, t)
		return
	}
	writeJSON(w, status, struct {
		adminTrust
		ExistingAccount bool `json:"existingAccount"`
	}{t, *existing})
}

// adminOrg is the {oid} of an admin route; the trust must exist (404 otherwise).
func (s *server) adminOrg(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	org, ok := pathID(w, r, "oid", orgNotFound)
	if !ok {
		return org, false
	}
	if _, err := s.q.GetOrgForAdmin(r.Context(), org); errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, orgNotFound)
		return org, false
	} else if err != nil {
		serverError(w, "get trust", err)
		return org, false
	}
	return org, true
}

func (s *server) adminGetOrg(w http.ResponseWriter, r *http.Request) {
	org, ok := pathID(w, r, "oid", orgNotFound)
	if ok {
		s.writeAdminTrust(w, r, org, http.StatusOK, nil)
	}
}

var codeFormat = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{1,11}$`)

// trustDetails are a trust's own details (ORG-02), set when it's created and edited at any time
// (S10 answers 3).
type trustDetails struct {
	Name         string `json:"name"`
	Code         string `json:"code"`
	ContactName  string `json:"contactName"`
	ContactEmail string `json:"contactEmail"`
	ContactPhone string `json:"contactPhone"`
}

// validate trims and checks the details, putting the code in capitals and the email in lower case.
func (d *trustDetails) validate() string {
	if msg := required(&d.Name, 200, "Enter the trust's name.", "The trust's name must be 200 characters or fewer."); msg != "" {
		return msg
	}
	if d.Code = strings.ToUpper(strings.TrimSpace(d.Code)); !codeFormat.MatchString(d.Code) {
		return "Enter a short code of 2 to 12 letters, digits or hyphens, like RXH or ST-MARY."
	}
	if msg := required(&d.ContactName, 200, "Enter the name of the trust's contact.", "The contact's name must be 200 characters or fewer."); msg != "" {
		return msg
	}
	var msg string
	if d.ContactEmail, msg = normalizeEmail(d.ContactEmail); msg != "" {
		return "Contact: " + msg
	}
	if d.ContactPhone = strings.TrimSpace(d.ContactPhone); d.ContactPhone != "" && !settings.PhoneOK(d.ContactPhone) {
		return "Contact: enter a phone number (digits, spaces and + ( ) - only), or leave it empty."
	}
	return ""
}

// codeTaken is the refusal for a code another trust has.
func codeTaken(code string) string {
	return fmt.Sprintf("Another trust already has the code %s. Choose another.", code)
}

// newTrustInput is the New trust form (ORG-02).
type newTrustInput struct {
	trustDetails
	LogoURL    string                     `json:"logoUrl"`
	AdminEmail string                     `json:"adminEmail"`
	AdminName  string                     `json:"adminName"`
	OptIns     map[string]json.RawMessage `json:"optIns"`
}

// required trims s and refuses it when it's empty or longer than max characters.
func required(s *string, max int, empty, long string) string {
	*s = strings.TrimSpace(*s)
	switch {
	case *s == "":
		return empty
	case utf8.RuneCountInString(*s) > max:
		return long
	}
	return ""
}

// setting checks one value for a settings key, returning its stored form.
func setting(key string, v any) (json.RawMessage, string) {
	raw, _ := json.Marshal(v)
	return settings.Validate(key, raw)
}

// textSettings checks text settings given as strings; "" becomes nil, which removes the key (the
// hospital follows the trust, or the trust has none).
func textSettings(values map[string]*string) (map[string]json.RawMessage, string) {
	out := map[string]json.RawMessage{}
	for _, key := range slices.Sorted(maps.Keys(values)) {
		v := values[key]
		if v == nil {
			continue
		}
		stored, msg := setting(key, *v)
		if msg != "" {
			return nil, msg
		}
		if string(stored) == `""` {
			stored = nil
		}
		out[key] = stored
	}
	return out, ""
}

// optInChanges checks a body's opt-ins: each must be a feature or instrument, and on or off, or
// (when allowNull) null, which removes it.
func optInChanges(in map[string]json.RawMessage, allowNull bool) (map[string]json.RawMessage, string) {
	out := map[string]json.RawMessage{}
	for _, k := range slices.Sorted(maps.Keys(in)) {
		if !settings.IsOptIn(k) {
			return nil, fmt.Sprintf("%q isn't a feature or opt-in.", k)
		}
		if allowNull && strings.TrimSpace(string(in[k])) == "null" {
			out[k] = nil
			continue
		}
		v, msg := settings.Validate(k, in[k])
		if msg != "" {
			return nil, msg
		}
		out[k] = v
	}
	return out, ""
}

// validate checks the form and returns the trust's first settings: its logo and opt-ins.
func (in *newTrustInput) validate() (settings.Values, string) {
	if msg := in.trustDetails.validate(); msg != "" {
		return nil, msg
	}
	var msg string
	if in.AdminEmail, msg = normalizeEmail(in.AdminEmail); msg != "" {
		return nil, "First admin: " + msg
	}
	if in.AdminName = strings.TrimSpace(in.AdminName); utf8.RuneCountInString(in.AdminName) > 200 {
		return nil, "The admin's name must be 200 characters or fewer."
	}
	if in.AdminName == "" {
		in.AdminName = in.AdminEmail
	}
	values, msg := optInChanges(in.OptIns, false)
	if msg != "" {
		return nil, msg
	}
	logo, msg := textSettings(map[string]*string{"branding.logoUrl": &in.LogoURL})
	if msg != "" {
		return nil, msg
	}
	maps.Copy(values, logo)
	return settings.Apply(nil, values), ""
}

// adminCreateOrg creates a trust and invites its first admin (ORG-01, ORG-02). The trust starts
// with no hospitals; the platform admin adds them next.
func (s *server) adminCreateOrg(w http.ResponseWriter, r *http.Request) {
	var in newTrustInput
	if !readJSON(w, r, 16<<10, &in) {
		return
	}
	values, msg := in.validate()
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	stored, err := json.Marshal(values)
	if err != nil {
		serverError(w, "encode settings", err)
		return
	}
	ctx := r.Context()
	var org uuid.UUID
	var existing bool
	ok := s.inTx(w, r, func(q *db.Queries) bool {
		ids, err := q.CreateOrg(ctx, db.CreateOrgParams{Name: in.Name, Code: in.Code, ContactName: in.ContactName,
			ContactEmail: in.ContactEmail, ContactPhone: in.ContactPhone, Settings: stored})
		if err != nil {
			serverError(w, "create trust", err)
			return false
		}
		if len(ids) == 0 {
			writeError(w, http.StatusConflict, codeTaken(in.Code))
			return false
		}
		org = ids[0]
		var invited bool
		_, existing, invited = s.inviteTo(w, r, q, org, uuid.NullUUID{}, in.AdminEmail, in.AdminName, role.Admin, alreadyTrustMember)
		return invited
	})
	if ok {
		s.writeAdminTrust(w, r, org, http.StatusCreated, &existing)
	}
}

// hospitalDetails are a hospital's own details (ORG-02), set when it's added and edited at any
// time (S10 answers 3). TimeZone "" is the deployment's (A-20).
type hospitalDetails struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	TimeZone string `json:"timeZone"`
}

func (d *hospitalDetails) validate() string {
	if msg := required(&d.Name, 200, "Enter the hospital's name.", "The hospital's name must be 200 characters or fewer."); msg != "" {
		return msg
	}
	if msg := required(&d.Address, 500, "Enter the hospital's address.", "The address must be 500 characters or fewer."); msg != "" {
		return msg
	}
	if d.TimeZone = strings.TrimSpace(d.TimeZone); d.TimeZone != "" && !validTimeZone(d.TimeZone) {
		return "Choose a time zone from the list, or leave it as the default."
	}
	return ""
}

// zone is the time zone column: nil for the deployment's.
func (d *hospitalDetails) zone() *string {
	if d.TimeZone == "" {
		return nil
	}
	return &d.TimeZone
}

// nameTaken is the refusal for a hospital name the trust already has.
func nameTaken(name string) string {
	return fmt.Sprintf("This trust already has a hospital called %s.", name)
}

// newHospitalInput is the Add hospital form (ORG-02). Phone, email and logo become the hospital's
// own settings; left empty, it follows the trust's.
type newHospitalInput struct {
	hospitalDetails
	Phone   string                     `json:"phone"`
	Email   string                     `json:"email"`
	LogoURL string                     `json:"logoUrl"`
	OptIns  map[string]json.RawMessage `json:"optIns"`
}

// validTimeZone reports whether name is an IANA time zone. Go's own zone database is built in
// (time/tzdata), so it doesn't depend on the server. "Local" is the server's, so it's refused.
func validTimeZone(name string) bool {
	if name == "" || name == "Local" {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}

func (in *newHospitalInput) validate() (settings.Values, string) {
	if msg := in.hospitalDetails.validate(); msg != "" {
		return nil, msg
	}
	values, msg := optInChanges(in.OptIns, false)
	if msg != "" {
		return nil, msg
	}
	own, msg := textSettings(map[string]*string{"contact.phone": &in.Phone, "contact.email": &in.Email, "branding.logoUrl": &in.LogoURL})
	if msg != "" {
		return nil, msg
	}
	maps.Copy(values, own)
	return settings.Apply(nil, values), ""
}

// adminAddHospital adds a hospital to a trust (ORG-01, ORG-02).
func (s *server) adminAddHospital(w http.ResponseWriter, r *http.Request) {
	org, ok := s.adminOrg(w, r)
	if !ok {
		return
	}
	var in newHospitalInput
	if !readJSON(w, r, 16<<10, &in) {
		return
	}
	values, msg := in.validate()
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	stored, err := json.Marshal(values)
	if err != nil {
		serverError(w, "encode settings", err)
		return
	}
	ids, err := s.q.CreateHospital(r.Context(), db.CreateHospitalParams{OrgID: org, Name: in.Name, Address: in.Address, Timezone: in.zone(), Settings: stored})
	if err != nil {
		serverError(w, "create hospital", err)
		return
	}
	if len(ids) == 0 {
		writeError(w, http.StatusConflict, nameTaken(in.Name))
		return
	}
	s.writeAdminTrust(w, r, org, http.StatusCreated, nil)
}

// statusPatch is the part of a PATCH common to trusts and hospitals: the status and/or opt-ins
// (null: back to following the trust, or for the trust, the platform's default).
type statusPatch struct {
	Status *string                    `json:"status"`
	OptIns map[string]json.RawMessage `json:"optIns"`
}

// validate checks the status and opt-ins; given says whether the PATCH changes anything else.
func (p *statusPatch) validate(given bool) (map[string]json.RawMessage, string) {
	if p.Status == nil && p.OptIns == nil && !given {
		return nil, "Nothing to change."
	}
	if p.Status != nil && *p.Status != "active" && *p.Status != "suspended" {
		return nil, "The status must be active or suspended."
	}
	if p.OptIns != nil && len(p.OptIns) == 0 {
		return nil, "Nothing to change."
	}
	return optInChanges(p.OptIns, true)
}

// orgPatch is a PATCH of a trust: any of its details, its logo, status and opt-ins. Only what is
// sent changes.
type orgPatch struct {
	statusPatch
	Name         *string `json:"name"`
	Code         *string `json:"code"`
	ContactName  *string `json:"contactName"`
	ContactEmail *string `json:"contactEmail"`
	ContactPhone *string `json:"contactPhone"`
	LogoURL      *string `json:"logoUrl"`
}

func (p *orgPatch) details() bool {
	return p.Name != nil || p.Code != nil || p.ContactName != nil || p.ContactEmail != nil || p.ContactPhone != nil
}

// hospitalPatch is a PATCH of a hospital: any of its details, its own phone, email and logo ("":
// the trust's), status and opt-ins. Only what is sent changes.
type hospitalPatch struct {
	statusPatch
	Name     *string `json:"name"`
	Address  *string `json:"address"`
	TimeZone *string `json:"timeZone"`
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	LogoURL  *string `json:"logoUrl"`
}

func (p *hospitalPatch) details() bool { return p.Name != nil || p.Address != nil || p.TimeZone != nil }

// overlay sets *dst to *v when v is given.
func overlay(dst *string, v *string) {
	if v != nil {
		*dst = *v
	}
}

// changeSettings applies checked changes to stored settings and encodes them.
func changeSettings(stored json.RawMessage, changes map[string]json.RawMessage) (json.RawMessage, error) {
	current, err := settings.Decode(stored)
	if err != nil {
		return nil, err
	}
	return json.Marshal(settings.Apply(current, changes))
}

// adminUpdateOrg edits a trust's details and logo (S10 answers 3), suspends or re-activates it
// (ORG-03: no sign-in, data kept), and sets its opt-ins (ORG-05).
func (s *server) adminUpdateOrg(w http.ResponseWriter, r *http.Request) {
	org, ok := pathID(w, r, "oid", orgNotFound)
	if !ok {
		return
	}
	var in orgPatch
	if !readJSON(w, r, 8<<10, &in) {
		return
	}
	changes, msg := in.validate(in.details() || in.LogoURL != nil)
	if msg == "" {
		var logo map[string]json.RawMessage
		logo, msg = textSettings(map[string]*string{"branding.logoUrl": in.LogoURL})
		maps.Copy(changes, logo)
	}
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	ok = s.inTx(w, r, func(q *db.Queries) bool {
		stored, err := q.LockOrgSettings(ctx, org)
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, orgNotFound)
			return false
		}
		if err != nil {
			serverError(w, "lock trust", err)
			return false
		}
		if in.details() {
			o, err := q.GetOrgForAdmin(ctx, org)
			if err != nil {
				serverError(w, "get trust", err)
				return false
			}
			d := trustDetails{Name: o.Name, Code: o.Code, ContactName: o.ContactName, ContactEmail: o.ContactEmail, ContactPhone: o.ContactPhone}
			overlay(&d.Name, in.Name)
			overlay(&d.Code, in.Code)
			overlay(&d.ContactName, in.ContactName)
			overlay(&d.ContactEmail, in.ContactEmail)
			overlay(&d.ContactPhone, in.ContactPhone)
			if msg := d.validate(); msg != "" {
				writeError(w, http.StatusBadRequest, msg)
				return false
			}
			err = q.UpdateOrgDetails(ctx, db.UpdateOrgDetailsParams{ID: org, Name: d.Name, Code: d.Code,
				ContactName: d.ContactName, ContactEmail: d.ContactEmail, ContactPhone: d.ContactPhone})
			if isUniqueViolation(err) {
				writeError(w, http.StatusConflict, codeTaken(d.Code))
				return false
			}
			if err != nil {
				serverError(w, "trust details", err)
				return false
			}
		}
		if len(changes) > 0 {
			next, err := changeSettings(stored, changes)
			if err == nil {
				err = q.SetOrgSettings(ctx, db.SetOrgSettingsParams{ID: org, Settings: next})
			}
			if err != nil {
				serverError(w, "trust settings", err)
				return false
			}
		}
		if in.Status != nil {
			if err := q.SetOrgStatus(ctx, db.SetOrgStatusParams{ID: org, Status: *in.Status}); err != nil {
				serverError(w, "trust status", err)
				return false
			}
		}
		return true
	})
	if ok {
		s.writeAdminTrust(w, r, org, http.StatusOK, nil)
	}
}

// adminUpdateHospital edits one hospital's details, phone, email and logo (S10 answers 3),
// suspends or re-activates it, and sets its own opt-ins.
func (s *server) adminUpdateHospital(w http.ResponseWriter, r *http.Request) {
	org, ok := pathID(w, r, "oid", orgNotFound)
	if !ok {
		return
	}
	hid, ok := pathID(w, r, "hid", "Hospital not found.")
	if !ok {
		return
	}
	var in hospitalPatch
	if !readJSON(w, r, 8<<10, &in) {
		return
	}
	own := map[string]*string{"contact.phone": in.Phone, "contact.email": in.Email, "branding.logoUrl": in.LogoURL}
	changes, msg := in.validate(in.details() || in.Phone != nil || in.Email != nil || in.LogoURL != nil)
	if msg == "" {
		var contact map[string]json.RawMessage
		contact, msg = textSettings(own)
		maps.Copy(changes, contact)
	}
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	ok = s.inTx(w, r, func(q *db.Queries) bool {
		h, err := q.LockHospitalForAdmin(ctx, db.LockHospitalForAdminParams{OrgID: org, ID: hid})
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "Hospital not found.")
			return false
		}
		if err != nil {
			serverError(w, "lock hospital", err)
			return false
		}
		d := hospitalDetails{Name: h.Name, Address: h.Address}
		if h.Timezone != nil {
			d.TimeZone = *h.Timezone
		}
		overlay(&d.Name, in.Name)
		overlay(&d.Address, in.Address)
		overlay(&d.TimeZone, in.TimeZone)
		if msg := d.validate(); msg != "" && in.details() { // details from before S10 may lack an address
			writeError(w, http.StatusBadRequest, msg)
			return false
		}
		next, err := changeSettings(h.Settings, changes)
		if err == nil {
			err = q.UpdateHospitalDetails(ctx, db.UpdateHospitalDetailsParams{OrgID: org, ID: hid, Name: d.Name,
				Address: d.Address, Timezone: d.zone(), Settings: next})
		}
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, nameTaken(d.Name))
			return false
		}
		if err != nil {
			serverError(w, "hospital details", err)
			return false
		}
		if in.Status != nil {
			if err := q.SetHospitalStatus(ctx, db.SetHospitalStatusParams{OrgID: org, ID: hid, Status: *in.Status}); err != nil {
				serverError(w, "hospital status", err)
				return false
			}
		}
		return true
	})
	if ok {
		s.writeAdminTrust(w, r, org, http.StatusOK, nil)
	}
}

// adminInviteAdmin invites another admin of the whole trust: for a first admin whose invitation
// expired, or a trust left without a working admin. It's S09's invite.
func (s *server) adminInviteAdmin(w http.ResponseWriter, r *http.Request) {
	org, ok := s.adminOrg(w, r)
	if !ok {
		return
	}
	var in struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if !readJSON(w, r, 4<<10, &in) {
		return
	}
	email, msg := normalizeEmail(in.Email)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	name := strings.TrimSpace(in.Name)
	if utf8.RuneCountInString(name) > 200 {
		writeError(w, http.StatusBadRequest, "The name must be 200 characters or fewer.")
		return
	}
	if name == "" {
		name = email
	}
	var existing bool
	ok = s.orgTx(w, r, org, func(q *db.Queries) bool {
		_, existing, ok = s.inviteTo(w, r, q, org, uuid.NullUUID{}, email, name, role.Admin, alreadyTrustMember)
		return ok
	})
	if ok {
		s.writeAdminTrust(w, r, org, http.StatusCreated, &existing)
	}
}
