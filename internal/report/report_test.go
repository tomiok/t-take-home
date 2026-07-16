package report

import (
	"os"
	"path/filepath"
	"testing"

	"timely-take-home/internal/catalog"
	"timely-take-home/internal/domain"
	"timely-take-home/internal/pipeline"
)

func testCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	path := filepath.Join(t.TempDir(), "courses.json")
	contents := `{"courses": [
		{"code": "MTH101", "name": "Algebra I", "department": "Math", "grades": [9], "prerequisites": [], "corequisites": []},
		{"code": "ENG101", "name": "English 9", "department": "English", "grades": [9], "prerequisites": [], "corequisites": []}
	]}`
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing catalog fixture: %v", err)
	}
	cat, err := catalog.Load(path)
	if err != nil {
		t.Fatalf("Load() catalog fixture: %v", err)
	}
	return cat
}

func TestBuildStudentSummary_SplitsAndOrdersByRank(t *testing.T) {
	cat := testCatalog(t)
	out := pipeline.Output{
		Students: []domain.Student{
			{ID: "meridian:1", Name: "Alex Rivera", Grade: 9, Source: "meridian"},
		},
		Requests: []domain.CourseRequest{
			// Deliberately out of rank order in the slice to prove the
			// summary sorts rather than trusting append order.
			{StudentID: "meridian:1", CourseCode: "AT101", Type: domain.Elective, Rank: 0},
			{StudentID: "meridian:1", CourseCode: "ENG101", Type: domain.Required, Rank: 1},
			{StudentID: "meridian:1", CourseCode: "MTH101", Type: domain.Required, Rank: 0},
		},
	}

	summary, found := BuildStudentSummary(out, cat, "meridian:1")
	if !found {
		t.Fatal("BuildStudentSummary() found = false, want true")
	}

	if len(summary.Required) != 2 {
		t.Fatalf("len(Required) = %d, want 2", len(summary.Required))
	}
	if summary.Required[0].CourseCode != "MTH101" || summary.Required[1].CourseCode != "ENG101" {
		t.Errorf("Required order = %+v, want [MTH101, ENG101] (Rank order, not append order)", summary.Required)
	}
	if summary.Required[0].CourseName != "Algebra I" || !summary.Required[0].Resolved {
		t.Errorf("Required[0] = %+v, want resolved Algebra I", summary.Required[0])
	}

	if len(summary.Elective) != 1 || summary.Elective[0].CourseCode != "AT101" {
		t.Fatalf("Elective = %+v, want [AT101]", summary.Elective)
	}
}

func TestBuildStudentSummary_UnresolvedCourseIsMarkedNotResolved(t *testing.T) {
	cat := testCatalog(t)
	out := pipeline.Output{
		Students: []domain.Student{{ID: "helix:1", Source: "helix"}},
		Requests: []domain.CourseRequest{
			{StudentID: "helix:1", CourseCode: "zz9zz9", Type: domain.Required, Rank: 0},
		},
	}

	summary, found := BuildStudentSummary(out, cat, "helix:1")
	if !found {
		t.Fatal("BuildStudentSummary() found = false, want true")
	}
	if len(summary.Required) != 1 {
		t.Fatalf("len(Required) = %d, want 1", len(summary.Required))
	}
	if summary.Required[0].Resolved {
		t.Error("Resolved = true, want false for a course code absent from the catalog")
	}
	if summary.Required[0].CourseName != "" {
		t.Errorf("CourseName = %q, want empty for an unresolved course", summary.Required[0].CourseName)
	}
}

func TestBuildStudentSummary_UnknownStudentNotFound(t *testing.T) {
	cat := testCatalog(t)
	out := pipeline.Output{
		Students: []domain.Student{{ID: "meridian:1", Source: "meridian"}},
	}

	_, found := BuildStudentSummary(out, cat, "meridian:does-not-exist")
	if found {
		t.Error("BuildStudentSummary() found = true, want false for an unknown student ID")
	}
}

func TestBuildStudentSummary_OnlyMatchesRequestsForThatStudent(t *testing.T) {
	cat := testCatalog(t)
	out := pipeline.Output{
		Students: []domain.Student{
			{ID: "meridian:1", Source: "meridian"},
			{ID: "meridian:2", Source: "meridian"},
		},
		Requests: []domain.CourseRequest{
			{StudentID: "meridian:1", CourseCode: "MTH101", Type: domain.Required, Rank: 0},
			{StudentID: "meridian:2", CourseCode: "ENG101", Type: domain.Required, Rank: 0},
		},
	}

	summary, found := BuildStudentSummary(out, cat, "meridian:1")
	if !found {
		t.Fatal("BuildStudentSummary() found = false, want true")
	}
	if len(summary.Required) != 1 || summary.Required[0].CourseCode != "MTH101" {
		t.Errorf("Required = %+v, want only meridian:1's MTH101", summary.Required)
	}
}
