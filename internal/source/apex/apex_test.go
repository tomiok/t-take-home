package apex

import (
	"os"
	"path/filepath"
	"testing"

	"timely-take-home/internal/domain"
)

func writeFixtures(t *testing.T, requestsJSON, crosswalkCSV string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	reqPath := filepath.Join(dir, "requests.json")
	cwPath := filepath.Join(dir, "course_crosswalk.csv")
	if err := os.WriteFile(reqPath, []byte(requestsJSON), 0o644); err != nil {
		t.Fatalf("writing requests fixture: %v", err)
	}
	if err := os.WriteFile(cwPath, []byte(crosswalkCSV), 0o644); err != nil {
		t.Fatalf("writing crosswalk fixture: %v", err)
	}
	return reqPath, cwPath
}

const crosswalk = `courseNumber,canonical_code
0140,MTH202
0815,AT301
`

func TestLoad_HappyPath(t *testing.T) {
	reqPath, cwPath := writeFixtures(t, `{"students": [
		{"stateStudentId": "CA-1", "name": "Sofia Mendez", "gradeLevel": 11, "requests": [
			{"courseNumber": "0140", "rank": 1},
			{"courseNumber": "0815", "rank": 500}
		]}
	]}`, crosswalk)

	result, err := Load(reqPath, cwPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Students) != 1 {
		t.Fatalf("len(Students) = %d, want 1", len(result.Students))
	}
	wantStudent := domain.Student{ID: "apex:CA-1", Name: "Sofia Mendez", Grade: 11, Source: "apex"}
	if result.Students[0] != wantStudent {
		t.Errorf("Students[0] = %+v, want %+v", result.Students[0], wantStudent)
	}

	if len(result.Requests) != 2 {
		t.Fatalf("len(Requests) = %d, want 2", len(result.Requests))
	}
	if got := result.Requests[0]; got.CourseCode != "MTH202" || got.Type != domain.Required || got.Rank != 0 {
		t.Errorf("Requests[0] = %+v, want CourseCode MTH202, Required, Rank 0", got)
	}
	if got := result.Requests[1]; got.CourseCode != "AT301" || got.Type != domain.Elective {
		t.Errorf("Requests[1] = %+v, want CourseCode AT301, Elective", got)
	}

	if len(result.Issues) != 0 {
		t.Errorf("Issues = %+v, want none", result.Issues)
	}
}

func TestLoad_Rank500IsBackupSentinelNotPriority(t *testing.T) {
	reqPath, cwPath := writeFixtures(t, `{"students": [
		{"stateStudentId": "CA-1", "name": "Sofia Mendez", "gradeLevel": 11, "requests": [
			{"courseNumber": "0815", "rank": 500}
		]}
	]}`, crosswalk)

	result, err := Load(reqPath, cwPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Requests) != 1 {
		t.Fatalf("len(Requests) = %d, want 1", len(result.Requests))
	}
	if result.Requests[0].Type != domain.Elective {
		t.Errorf("Type = %v, want Elective for rank-500 sentinel", result.Requests[0].Type)
	}
	if result.Requests[0].Rank == 500 {
		t.Error("Rank = 500, want the sentinel value NOT carried through as a literal priority number")
	}
}

func TestLoad_UnmappedCourseNumberKeepsRawIdentifierWithIssue(t *testing.T) {
	reqPath, cwPath := writeFixtures(t, `{"students": [
		{"stateStudentId": "CA-1", "name": "Hana Kim", "gradeLevel": 11, "requests": [
			{"courseNumber": "0999", "rank": 3}
		]}
	]}`, crosswalk)

	result, err := Load(reqPath, cwPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Requests) != 1 {
		t.Fatalf("len(Requests) = %d, want 1", len(result.Requests))
	}
	if result.Requests[0].CourseCode != "0999" {
		t.Errorf("CourseCode = %q, want raw %q kept when crosswalk has no entry", result.Requests[0].CourseCode, "0999")
	}
	if result.Requests[0].Type != domain.Required {
		t.Errorf("Type = %v, want Required — an unresolved course number at a real priority rank must not be conflated with a backup", result.Requests[0].Type)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}
}

func TestLoad_MissingFiles(t *testing.T) {
	dir := t.TempDir()
	missingReq := filepath.Join(dir, "requests.json")
	missingCW := filepath.Join(dir, "course_crosswalk.csv")

	if _, err := Load(missingReq, missingCW); err == nil {
		t.Fatal("Load() error = nil, want error when both files are missing")
	}

	_, cwPath := writeFixtures(t, `{"students": []}`, crosswalk)
	if _, err := Load(missingReq, cwPath); err == nil {
		t.Fatal("Load() error = nil, want error when requests.json is missing")
	}
}

func TestLoad_RealFixture(t *testing.T) {
	result, err := Load(
		filepath.Join("..", "..", "..", "data", "apex", "requests.json"),
		filepath.Join("..", "..", "..", "data", "apex", "course_crosswalk.csv"),
	)
	if err != nil {
		t.Fatalf("Load() real fixture: %v", err)
	}

	if len(result.Students) != 4 {
		t.Errorf("len(Students) = %d, want 4", len(result.Students))
	}
	if len(result.Requests) != 17 {
		t.Errorf("len(Requests) = %d, want 17", len(result.Requests))
	}
	if len(result.Issues) != 1 {
		t.Errorf("len(Issues) = %d, want 1 (Hana Kim's unmapped course number 0999)", len(result.Issues))
	}
}
