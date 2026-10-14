package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// fakePool is a user pool in memory: email → enrolled.
type fakePool struct {
	users     map[string]bool
	passwords map[string]string
	ops       []string
}

func newFakePool() *fakePool {
	return &fakePool{users: map[string]bool{}, passwords: map[string]string{}}
}

func (f *fakePool) Find(_ context.Context, email string) (bool, bool, error) {
	enrolled, ok := f.users[email]
	return ok, enrolled, nil
}
func (f *fakePool) Create(_ context.Context, email string) error {
	f.ops = append(f.ops, "create "+email)
	f.users[email] = false
	return nil
}
func (f *fakePool) Delete(_ context.Context, email string) error {
	f.ops = append(f.ops, "delete "+email)
	delete(f.users, email)
	return nil
}
func (f *fakePool) SetPassword(_ context.Context, email, pw string) error {
	f.passwords[email] = pw
	return nil
}
func (f *fakePool) Enroll(_ context.Context, email, _ string) (string, error) {
	f.ops = append(f.ops, "enroll "+email)
	f.users[email] = true
	return "SECRET" + strings.ToUpper(email[:3]), nil
}

type memStore struct{ users []testUser }

func (m *memStore) Load(context.Context) ([]testUser, error)   { return m.users, nil }
func (m *memStore) Save(_ context.Context, u []testUser) error { m.users = u; return nil }

var people = []person{{"Ada Admin", "ada@trust-a.example"}, {"Cara Clinician", "cara@hospital-a.example"}}

func counter() func() string {
	n := 0
	return func() string { n++; return "pw" + string(rune('0'+n)) }
}

func TestSetupMakesEachDemoAccountASignIn(t *testing.T) {
	p, st := newFakePool(), &memStore{}
	users, err := setup(context.Background(), people, p, st, counter())
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].Password != "pw1" || users[0].TOTPSecret != "SECRETADA" || users[1].Name != "Cara Clinician" {
		t.Fatalf("users = %+v", users)
	}
	if p.passwords["ada@trust-a.example"] != "pw1" || !p.users["cara@hospital-a.example"] {
		t.Errorf("pool = %+v %+v", p.users, p.passwords)
	}
	if len(st.users) != 2 {
		t.Error("not saved")
	}
}

func TestSetupAgainKeepsTheSecretsAndPutsThePasswordsBack(t *testing.T) {
	p, st := newFakePool(), &memStore{}
	first, _ := setup(context.Background(), people, p, st, counter())
	p.passwords["ada@trust-a.example"] = "changed by a tester"
	p.ops = nil
	again, err := setup(context.Background(), people, p, st, counter())
	if err != nil {
		t.Fatal(err)
	}
	if again[0] != first[0] || again[1] != first[1] {
		t.Errorf("changed: %+v → %+v", first, again)
	}
	if len(p.ops) != 0 {
		t.Errorf("pool calls %v; want none but the passwords", p.ops)
	}
	if p.passwords["ada@trust-a.example"] != first[0].Password {
		t.Error("the stored password wasn't put back")
	}
}

func TestSetupRemakesAnAccountWhoseSecretIsUnknown(t *testing.T) {
	p, st := newFakePool(), &memStore{}
	p.users["ada@trust-a.example"] = true // e.g. enrolled by hand; the secret isn't stored
	if _, err := setup(context.Background(), people, p, st, counter()); err != nil {
		t.Fatal(err)
	}
	if strings.Join(p.ops[:3], ",") != "delete ada@trust-a.example,create ada@trust-a.example,enroll ada@trust-a.example" {
		t.Errorf("ops = %v", p.ops)
	}
}

func TestFindByNameOrEmail(t *testing.T) {
	users := []testUser{{Name: "Cara Clinician", Email: "cara@hospital-a.example"}}
	for _, who := range []string{"cara clinician", "CARA@hospital-a.example"} {
		if u, err := find(users, who); err != nil || u.Name != "Cara Clinician" {
			t.Errorf("%q: %+v, %v", who, u, err)
		}
	}
	if _, err := find(users, "Nobody"); err == nil || !strings.Contains(err.Error(), "Cara Clinician") {
		t.Errorf("err = %v; want one listing who there is", err)
	}
}

func TestPasswordsMeetThePoolsPolicy(t *testing.T) {
	pw := newPassword()
	if len(pw) < 12 || !strings.ContainsAny(pw, "ABCDEFGHJKLMNPQRSTUVWXYZ") || !strings.ContainsAny(pw, "abcdefghijkmnopqrstuvwxyz") ||
		!strings.ContainsAny(pw, "0123456789") || !strings.ContainsAny(pw, "!") {
		t.Errorf("password %q misses a kind of character", pw)
	}
	if newPassword() == pw {
		t.Error("two passwords are the same")
	}
}

// The command refuses any stage but QA before touching anything.
func TestRefusesOtherStages(t *testing.T) {
	if os.Getenv("QA_USERS_RUN_MAIN") == "1" {
		main()
		return
	}
	for _, stage := range []string{"", "training", "production"} {
		cmd := exec.Command(os.Args[0], "-test.run=TestRefusesOtherStages")
		cmd.Args = append(cmd.Args, "--", "setup")
		cmd.Env = append(os.Environ(), "QA_USERS_RUN_MAIN=1", "STAGE="+stage, "QA_USERS_PARAM=/sj/x", "DATABASE_URL=postgres://nowhere")
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "QA only") {
			t.Errorf("stage %q: err %v, output %s", stage, err, out)
		}
	}
}
