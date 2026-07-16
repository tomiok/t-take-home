package validation

import (
	"os"
	"path/filepath"
	"testing"

	"timely-take-home/internal/catalog"
	"timely-take-home/internal/domain"
)

func loadCatalog(t *testing.T, coursesJSON string) *catalog.Catalog {
	t.Helper()
	path := filepath.Join(t.TempDir(), "courses.json")
	if err := os.WriteFile(path, []byte(coursesJSON), 0o644); err != nil {
		t.Fatalf("writing catalog fixture: %v", err)
	}
	c, err := catalog.Load(path)
	if err != nil {
		t.Fatalf("Load() catalog fixture: %v", err)
	}
	return c
}

const twoCoreqCourses = `{"courses": [
	{"code": "MTH202", "name": "Pre-Calculus", "department": "Math", "grades": [11], "prerequisites": [], "corequisites": []},
	{"code": "SCI301", "name": "Physics", "department": "Science", "grades": [11], "prerequisites": [], "corequisites": ["MTH202"]}
]}`

func TestValidate_UnknownCourseCode(t *testing.T) {
	cat := loadCatalog(t, twoCoreqCourses)
	requests := []domain.CourseRequest{
		{StudentID: "meridian:1", CourseCode: "DOES_NOT_EXIST"},
	}

	report := Validate(cat, requests)

	if len(report.UnknownCourses) != 1 {
		t.Fatalf("len(UnknownCourses) = %d, want 1", len(report.UnknownCourses))
	}
	want := UnknownCourseFinding{StudentID: "meridian:1", CourseCode: "DOES_NOT_EXIST"}
	if report.UnknownCourses[0] != want {
		t.Errorf("UnknownCourses[0] = %+v, want %+v", report.UnknownCourses[0], want)
	}
	if len(report.MissingCorequisites) != 0 {
		t.Errorf("MissingCorequisites = %+v, want none", report.MissingCorequisites)
	}
}

func TestValidate_MissingCorequisite_DeclaredDirection(t *testing.T) {
	cat := loadCatalog(t, twoCoreqCourses)
	// SCI301 declares MTH202 as a corequisite; student has SCI301 but not MTH202.
	requests := []domain.CourseRequest{
		{StudentID: "meridian:1", CourseCode: "SCI301"},
	}

	report := Validate(cat, requests)

	if len(report.MissingCorequisites) != 1 {
		t.Fatalf("len(MissingCorequisites) = %d, want 1", len(report.MissingCorequisites))
	}
	want := MissingCorequisiteFinding{StudentID: "meridian:1", HasCourse: "SCI301", MissingCourse: "MTH202"}
	if report.MissingCorequisites[0] != want {
		t.Errorf("MissingCorequisites[0] = %+v, want %+v", report.MissingCorequisites[0], want)
	}
}

func TestValidate_MissingCorequisite_ReverseDirection(t *testing.T) {
	cat := loadCatalog(t, twoCoreqCourses)
	// MTH202 does NOT declare SCI301 as a corequisite in the raw data, but
	// the pair must still be treated as symmetric: student has MTH202 but
	// not SCI301 should still be flagged.
	requests := []domain.CourseRequest{
		{StudentID: "meridian:1", CourseCode: "MTH202"},
	}

	report := Validate(cat, requests)

	if len(report.MissingCorequisites) != 1 {
		t.Fatalf("len(MissingCorequisites) = %d, want 1", len(report.MissingCorequisites))
	}
	want := MissingCorequisiteFinding{StudentID: "meridian:1", HasCourse: "MTH202", MissingCourse: "SCI301"}
	if report.MissingCorequisites[0] != want {
		t.Errorf("MissingCorequisites[0] = %+v, want %+v", report.MissingCorequisites[0], want)
	}
}

func TestValidate_SatisfiedCorequisitePairIsNotFlagged(t *testing.T) {
	cat := loadCatalog(t, twoCoreqCourses)
	requests := []domain.CourseRequest{
		{StudentID: "meridian:1", CourseCode: "SCI301"},
		{StudentID: "meridian:1", CourseCode: "MTH202"},
	}

	report := Validate(cat, requests)

	if len(report.MissingCorequisites) != 0 {
		t.Errorf("MissingCorequisites = %+v, want none (student has both halves)", report.MissingCorequisites)
	}
}

func TestValidate_NeitherHalfRequestedIsNotFlagged(t *testing.T) {
	// A course entirely unrelated to any corequisite pair in the catalog
	// must never appear in a MissingCorequisiteFinding.
	cat := loadCatalog(t, `{"courses": [
		{"code": "MTH202", "name": "Pre-Calculus", "department": "Math", "grades": [11], "prerequisites": [], "corequisites": []},
		{"code": "SCI301", "name": "Physics", "department": "Science", "grades": [11], "prerequisites": [], "corequisites": ["MTH202"]},
		{"code": "ART101", "name": "Art I", "department": "Arts & Tech", "grades": [9], "prerequisites": [], "corequisites": []}
	]}`)

	report := Validate(cat, []domain.CourseRequest{
		{StudentID: "meridian:1", CourseCode: "ART101"},
	})

	if len(report.MissingCorequisites) != 0 {
		t.Errorf("MissingCorequisites = %+v, want none (ART101 has no corequisite pair)", report.MissingCorequisites)
	}
}

func TestValidate_FindingsAreIsolatedPerStudent(t *testing.T) {
	cat := loadCatalog(t, twoCoreqCourses)
	requests := []domain.CourseRequest{
		{StudentID: "meridian:1", CourseCode: "SCI301"},
		{StudentID: "meridian:1", CourseCode: "MTH202"},
		{StudentID: "meridian:2", CourseCode: "SCI301"},
	}

	report := Validate(cat, requests)

	if len(report.MissingCorequisites) != 1 {
		t.Fatalf("len(MissingCorequisites) = %d, want 1 (only student 2 is missing a half)", len(report.MissingCorequisites))
	}
	if report.MissingCorequisites[0].StudentID != "meridian:2" {
		t.Errorf("MissingCorequisites[0].StudentID = %q, want meridian:2", report.MissingCorequisites[0].StudentID)
	}
}

func TestValidate_UnknownCourseDoesNotParticipateInCorequisiteCheck(t *testing.T) {
	cat := loadCatalog(t, twoCoreqCourses)
	requests := []domain.CourseRequest{
		{StudentID: "meridian:1", CourseCode: "GHOST_COURSE"},
	}

	report := Validate(cat, requests)

	if len(report.UnknownCourses) != 1 {
		t.Fatalf("len(UnknownCourses) = %d, want 1", len(report.UnknownCourses))
	}
	if len(report.MissingCorequisites) != 0 {
		t.Errorf("MissingCorequisites = %+v, want none (unresolved course shouldn't trigger a corequisite check)", report.MissingCorequisites)
	}
}

func TestValidate_RealFixture(t *testing.T) {
	cat, err := catalog.Load(filepath.Join("..", "..", "data", "catalog", "courses.json"))
	if err != nil {
		t.Fatalf("Load() real catalog: %v", err)
	}

	// Known from the real data: apex CA-2291205 has MTH202 but not SCI301;
	// apex CA-2291309 has SCI301 but not MTH202; helix HX-560 has MTH202
	// but not SCI301; meridian 10042 and 10057 have ENG101 but not ENG102.
	// Plus 3 unknown course codes: apex's 0999, cornerstone's ART999,
	// helix's zz9zz9 (all confirmed absent from the real catalog).
	requests := []domain.CourseRequest{
		{StudentID: "apex:CA-2291205", CourseCode: "MTH202"},
		{StudentID: "apex:CA-2291205", CourseCode: "0999"},
		{StudentID: "apex:CA-2291309", CourseCode: "SCI301"},
		{StudentID: "helix:HX-560", CourseCode: "MTH202"},
		{StudentID: "helix:HX-560", CourseCode: "zz9zz9"},
		{StudentID: "meridian:10042", CourseCode: "ENG101"},
		{StudentID: "meridian:10057", CourseCode: "ENG101"},
		{StudentID: "cornerstone:nina.torres@cornerstone.edu", CourseCode: "ART999"},
	}

	report := Validate(cat, requests)

	if len(report.UnknownCourses) != 3 {
		t.Errorf("len(UnknownCourses) = %d, want 3", len(report.UnknownCourses))
	}
	if len(report.MissingCorequisites) != 5 {
		t.Errorf("len(MissingCorequisites) = %d, want 5", len(report.MissingCorequisites))
	}
}
