package repocheck_test

import (
	"bytes"
	"encoding/base64"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The incumbent product's name appears nowhere in the repo: not in code, comments, docs, tools,
// tests or file names, in any letter case (Rahul, 22 Oct). It's base64-encoded here so this
// file doesn't contain it either.
const encodedName = "bGlmZWJveA=="

// Folders that are built, downloaded or tool caches, not ours.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, ".venv": true, "dist": true, "cdk.out": true,
	"test-results": true, "playwright-report": true, "__pycache__": true, ".pytest_cache": true,
}

func TestIncumbentNameAppearsNowhere(t *testing.T) {
	name, err := base64.StdEncoding.DecodeString(encodedName)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..")) // the repo, from apps/api/internal/repocheck
	if err != nil {
		t.Fatal(err)
	}
	scanned := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() && skipDirs[d.Name()] {
			return filepath.SkipDir
		}
		if strings.Contains(strings.ToLower(rel), string(name)) {
			t.Errorf("%s: the incumbent's name is in the path", rel)
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		for i, line := range bytes.Split(bytes.ToLower(src), []byte("\n")) {
			if bytes.Contains(line, name) {
				t.Errorf("%s:%d mentions the incumbent by name", rel, i+1)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 200 {
		t.Fatalf("scanned only %d files under %s; is the repo layout different?", scanned, root)
	}
}
