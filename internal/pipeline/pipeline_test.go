package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func realDataDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "data")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("real data dir %s not found: %v", dir, err)
	}
	return dir
}

func TestRun_CombinesAllFourSources(t *testing.T) {
	out, err := Run(realDataDir(t))
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Known per-adapter counts (pinned in each adapter's own tests):
	// meridian 5/24/2, apex 4/17/1, cornerstone 3/12/0, helix 3/13/1.
	if len(out.Students) != 15 {
		t.Errorf("len(Students) = %d, want 15", len(out.Students))
	}
	if len(out.Requests) != 66 {
		t.Errorf("len(Requests) = %d, want 66", len(out.Requests))
	}
	if len(out.Issues) != 4 {
		t.Errorf("len(Issues) = %d, want 4", len(out.Issues))
	}
}

func TestRun_NoDuplicateStudentIDsAcrossSources(t *testing.T) {
	out, err := Run(realDataDir(t))
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	seen := make(map[string]bool, len(out.Students))
	for _, s := range out.Students {
		if seen[s.ID] {
			t.Errorf("duplicate Student.ID %q across sources", s.ID)
		}
		seen[s.ID] = true
	}
}

func TestRun_MissingDataDir(t *testing.T) {
	if _, err := Run(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("Run() error = nil, want error for a data dir that doesn't exist")
	}
}

func TestWrite_ProducesValidJSON(t *testing.T) {
	out, err := Run(realDataDir(t))
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	path := filepath.Join(t.TempDir(), "unified.json")
	if err := Write(out, path); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}

	var roundTripped Output
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("Write() produced invalid JSON: %v", err)
	}

	if len(roundTripped.Students) != len(out.Students) || len(roundTripped.Requests) != len(out.Requests) {
		t.Errorf("round-tripped Output = %d students/%d requests, want %d/%d",
			len(roundTripped.Students), len(roundTripped.Requests), len(out.Students), len(out.Requests))
	}
}
