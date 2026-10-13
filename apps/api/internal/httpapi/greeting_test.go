package httpapi

import "testing"

// The home greets the person by the newest first name written with a capital, else the newest,
// capitalised (S15 test report 2: a hospital typed " sam ").
func TestGreeting(t *testing.T) {
	for _, tc := range []struct {
		names []string // newest first
		want  string
	}{
		{nil, ""},
		{[]string{"Sam"}, "Sam"},
		{[]string{" sam ", "Sam"}, "Sam"},
		{[]string{"sam", "sam"}, "Sam"},
		{[]string{"Sam", "SAM"}, "Sam"},
		{[]string{"émile"}, "Émile"},
		{[]string{"  "}, ""},
	} {
		got := ""
		for _, n := range tc.names {
			got = greeting(got, n)
		}
		if got != tc.want {
			t.Errorf("%q → %q, want %q", tc.names, got, tc.want)
		}
	}
}
