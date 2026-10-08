package httpapi

import "testing"

// FRM-07: a PATCH sends only the details it changes. A detail sent as null is refused, not a
// crash (found in S01: the handler dereferenced the nil pointer and dropped the connection).
func TestChapterDetailsValidate(t *testing.T) {
	s := func(v string) *string { return &v }
	valid := func() chapterDetails {
		return chapterDetails{Name: s(" Heart "), Description: s(" About your heart "), Icon: s("heart"), Audience: s("patient")}
	}

	in := valid()
	if msg := in.validate(); msg != "" || *in.Name != "Heart" || *in.Description != "About your heart" {
		t.Errorf("valid details: %q, name %q, description %q; want no problem, trimmed", msg, *in.Name, *in.Description)
	}
	for name, change := range map[string]func(*chapterDetails){
		"null name":        func(d *chapterDetails) { d.Name = nil },
		"null description": func(d *chapterDetails) { d.Description = nil },
		"null icon":        func(d *chapterDetails) { d.Icon = nil },
		"null audience":    func(d *chapterDetails) { d.Audience = nil },
		"blank name":       func(d *chapterDetails) { d.Name = s("  ") },
		"unknown icon":     func(d *chapterDetails) { d.Icon = s("Heart!") },
		"unknown audience": func(d *chapterDetails) { d.Audience = s("nurse") },
	} {
		in := valid()
		change(&in)
		if in.validate() == "" {
			t.Errorf("%s: no problem reported", name)
		}
	}
}
