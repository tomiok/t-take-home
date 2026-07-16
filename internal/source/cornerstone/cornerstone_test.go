package cornerstone

import (
	"os"
	"path/filepath"
	"testing"

	"timely-take-home/internal/domain"
)

func writeFixtures(t *testing.T, primaryCSV, alternateCSV string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "primary_requests.csv")
	alternatePath := filepath.Join(dir, "alternate_requests.csv")
	if err := os.WriteFile(primaryPath, []byte(primaryCSV), 0o644); err != nil {
		t.Fatalf("writing primary fixture: %v", err)
	}
	if err := os.WriteFile(alternatePath, []byte(alternateCSV), 0o644); err != nil {
		t.Fatalf("writing alternate fixture: %v", err)
	}
	return primaryPath, alternatePath
}

func TestLoad_HappyPath(t *testing.T) {
	primaryPath, alternatePath := writeFixtures(t,
		"student_email,course_code,terms\nmarcus.thompson@cornerstone.edu,MTH202,FULL\n",
		"student_email,course_code,terms\nmarcus.thompson@cornerstone.edu,AT302,FULL\n",
	)

	result, err := Load(primaryPath, alternatePath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Students) != 1 {
		t.Fatalf("len(Students) = %d, want 1", len(result.Students))
	}
	if len(result.Requests) != 2 {
		t.Fatalf("len(Requests) = %d, want 2", len(result.Requests))
	}
	if result.Requests[0].Type != domain.Required || result.Requests[0].CourseCode != "MTH202" {
		t.Errorf("Requests[0] = %+v, want Required MTH202 (from primary file)", result.Requests[0])
	}
	if result.Requests[1].Type != domain.Elective || result.Requests[1].CourseCode != "AT302" {
		t.Errorf("Requests[1] = %+v, want Elective AT302 (from alternate file)", result.Requests[1])
	}
	if result.Requests[0].Terms != "FULL" {
		t.Errorf("Terms = %q, want FULL", result.Requests[0].Terms)
	}
}

func TestLoad_EmailNormalizationMergesSameStudent(t *testing.T) {
	primaryPath, alternatePath := writeFixtures(t,
		"student_email,course_code,terms\n"+
			"marcus.thompson@cornerstone.edu,MTH202,FULL\n"+
			" Marcus.Thompson@cornerstone.edu ,SS301,FULL\n",
		"student_email,course_code,terms\n",
	)

	result, err := Load(primaryPath, alternatePath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Students) != 1 {
		t.Fatalf("len(Students) = %d, want 1 (case/whitespace variants of the same email must merge)", len(result.Students))
	}
	if len(result.Requests) != 2 {
		t.Fatalf("len(Requests) = %d, want 2", len(result.Requests))
	}
	if result.Requests[0].StudentID != result.Requests[1].StudentID {
		t.Errorf("StudentID mismatch: %q vs %q, want same student", result.Requests[0].StudentID, result.Requests[1].StudentID)
	}
	if result.Requests[1].Rank != 1 {
		t.Errorf("second request Rank = %d, want 1 (same student's second Required request)", result.Requests[1].Rank)
	}
}

func TestLoad_MissingFieldsSkipWithIssue(t *testing.T) {
	primaryPath, alternatePath := writeFixtures(t,
		"student_email,course_code,terms\n,MTH202,FULL\nnina.torres@cornerstone.edu,,FULL\n",
		"student_email,course_code,terms\n",
	)

	result, err := Load(primaryPath, alternatePath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Requests) != 0 {
		t.Errorf("len(Requests) = %d, want 0", len(result.Requests))
	}
	if len(result.Issues) != 2 {
		t.Fatalf("len(Issues) = %d, want 2", len(result.Issues))
	}
}

func TestLoad_MissingFiles(t *testing.T) {
	dir := t.TempDir()
	missingPrimary := filepath.Join(dir, "primary_requests.csv")
	missingAlternate := filepath.Join(dir, "alternate_requests.csv")

	if _, err := Load(missingPrimary, missingAlternate); err == nil {
		t.Fatal("Load() error = nil, want error when both files are missing")
	}

	_, alternatePath := writeFixtures(t, "student_email,course_code,terms\n", "student_email,course_code,terms\n")
	if _, err := Load(missingPrimary, alternatePath); err == nil {
		t.Fatal("Load() error = nil, want error when primary file is missing")
	}
}

func TestLoad_RealFixture(t *testing.T) {
	result, err := Load(
		filepath.Join("..", "..", "..", "data", "cornerstone", "primary_requests.csv"),
		filepath.Join("..", "..", "..", "data", "cornerstone", "alternate_requests.csv"),
	)
	if err != nil {
		t.Fatalf("Load() real fixture: %v", err)
	}

	if len(result.Students) != 3 {
		t.Errorf("len(Students) = %d, want 3 (marcus.thompson, nina.torres, aisha.j)", len(result.Students))
	}
	if len(result.Requests) != 12 {
		t.Errorf("len(Requests) = %d, want 12 (9 primary + 3 alternate rows)", len(result.Requests))
	}

	var marcusRequired int
	for _, r := range result.Requests {
		if r.StudentID == "cornerstone:marcus.thompson@cornerstone.edu" && r.Type == domain.Required {
			marcusRequired++
		}
	}
	if marcusRequired != 4 {
		t.Errorf("marcus.thompson Required requests = %d, want 4 (the whitespace/case-variant row must merge into the same student)", marcusRequired)
	}
}
