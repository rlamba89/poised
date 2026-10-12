package config_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A-20 rule 2: no region, time zone or locale literal in code outside this package. This scans
// the Go API, the web app and packages/clinical (tests and comments aside), so a per-country
// value can't creep back in. A new value belongs in config.Country, and the web app gets it
// from GET /api/config.
var countryLiteral = regexp.MustCompile(
	`["'` + "`" + `](` +
		`[a-z]{2}-[A-Z]{2}` + // a locale such as en-GB
		`|(Africa|America|Antarctica|Asia|Atlantic|Australia|Europe|Indian|Pacific)/[A-Za-z_]+` + // an IANA zone
		`|[a-z]{2}-(north|south|east|west|central|northeast|southeast)-\d` + // an AWS region
		`)["'` + "`" + `]`)

func TestNoCountryLiteralsOutsideConfig(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..")) // the repo, from apps/api/internal/config
	if err != nil {
		t.Fatal(err)
	}
	dirs := []string{"apps/api", "apps/web/src", "packages/clinical/src"}
	self := filepath.Join(root, "apps", "api", "internal", "config")
	scanned := 0
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path == self || d.Name() == "node_modules" || d.Name() == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			ext := filepath.Ext(path)
			if (ext != ".go" && ext != ".ts" && ext != ".tsx") || isTest(path) {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			scanned++
			for i, line := range strings.Split(string(src), "\n") {
				if code := stripComment(line); countryLiteral.MatchString(code) {
					rel, _ := filepath.Rel(root, path)
					t.Errorf("%s:%d has a per-country literal (A-20); read it from config instead:\n\t%s",
						rel, i+1, strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if scanned < 50 {
		t.Fatalf("scanned only %d files under %s; is the repo layout different?", scanned, root)
	}
}

func isTest(path string) bool {
	return strings.HasSuffix(path, "_test.go") || strings.Contains(filepath.Base(path), ".test.")
}

// stripComment drops a // comment (and lines inside /* */ or JSDoc blocks), roughly: enough
// for a guard, since a URL's // inside a string doesn't hold a country literal.
func stripComment(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/*") {
		return ""
	}
	if i := strings.Index(line, "//"); i >= 0 && !strings.Contains(line[:i], "://") && !strings.HasSuffix(line[:i], ":") {
		return line[:i]
	}
	return line
}
