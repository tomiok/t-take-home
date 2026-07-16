package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "courses.json")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return path
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		contents   string
		wantErr    bool
		wantLen    int
		lookupCode string
		wantFound  bool
	}{
		{
			name: "loads and indexes courses by code",
			contents: `{"courses": [
				{"code": "MTH101", "name": "Algebra I", "department": "Math", "grades": [9], "prerequisites": [], "corequisites": []},
				{"code": "MTH201", "name": "Algebra II", "department": "Math", "grades": [10], "prerequisites": ["MTH101"], "corequisites": []}
			]}`,
			wantLen:    2,
			lookupCode: "MTH201",
			wantFound:  true,
		},
		{
			name:       "unknown code is not found",
			contents:   `{"courses": [{"code": "MTH101", "name": "Algebra I", "department": "Math", "grades": [9], "prerequisites": [], "corequisites": []}]}`,
			wantLen:    1,
			lookupCode: "DOES_NOT_EXIST",
			wantFound:  false,
		},
		{
			name:     "malformed json errors instead of panicking",
			contents: `{"courses": [`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFixture(t, tt.contents)

			c, err := Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}

			if got := c.Len(); got != tt.wantLen {
				t.Errorf("Len() = %d, want %d", got, tt.wantLen)
			}

			if tt.lookupCode != "" {
				_, found := c.Lookup(tt.lookupCode)
				if found != tt.wantFound {
					t.Errorf("Lookup(%q) found = %v, want %v", tt.lookupCode, found, tt.wantFound)
				}
			}
		})
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err == nil {
		t.Fatal("Load() error = nil, want error for missing file")
	}
}

func TestLoad_RealCatalogFixture(t *testing.T) {
	c, err := Load(filepath.Join("..", "..", "data", "catalog", "courses.json"))
	if err != nil {
		t.Fatalf("Load() real catalog: %v", err)
	}

	if c.Len() == 0 {
		t.Fatal("Len() = 0, want a non-empty catalog")
	}

	// SCI301 declares MTH202 as a corequisite one-directionally in the real
	// data; assert that asymmetry is preserved as-is rather than the loader
	// silently mirroring it.
	sci301, ok := c.Lookup("SCI301")
	if !ok {
		t.Fatal(`Lookup("SCI301") not found in real catalog fixture`)
	}
	if len(sci301.Corequisites) != 1 || sci301.Corequisites[0] != "MTH202" {
		t.Errorf("SCI301.Corequisites = %v, want [MTH202]", sci301.Corequisites)
	}

	mth202, ok := c.Lookup("MTH202")
	if !ok {
		t.Fatal(`Lookup("MTH202") not found in real catalog fixture`)
	}
	if len(mth202.Corequisites) != 0 {
		t.Errorf("MTH202.Corequisites = %v, want empty (asymmetric in source data)", mth202.Corequisites)
	}

	if got := len(c.All()); got != c.Len() {
		t.Errorf("len(All()) = %d, want %d (Len())", got, c.Len())
	}
}
