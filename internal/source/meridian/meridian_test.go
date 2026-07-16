package meridian

import (
	"os"
	"path/filepath"
	"testing"

	"timely-take-home/internal/domain"
)

func writeCSV(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "course_requests.csv")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return path
}

func TestLoad_HappyPath(t *testing.T) {
	path := writeCSV(t, `student_id,student_name,grade,course_code,request_type
10042,Alex Rivera,9,MTH101,REQUIRED
10042,Alex Rivera,9,AT101,ELECTIVE
`)

	result, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Students) != 1 {
		t.Fatalf("len(Students) = %d, want 1", len(result.Students))
	}
	want := domain.Student{ID: "meridian:10042", Name: "Alex Rivera", Grade: 9, Source: "meridian"}
	if result.Students[0] != want {
		t.Errorf("Students[0] = %+v, want %+v", result.Students[0], want)
	}

	if len(result.Requests) != 2 {
		t.Fatalf("len(Requests) = %d, want 2", len(result.Requests))
	}
	if result.Requests[0].Type != domain.Required || result.Requests[0].CourseCode != "MTH101" {
		t.Errorf("Requests[0] = %+v, want Required MTH101", result.Requests[0])
	}
	if result.Requests[1].Type != domain.Elective || result.Requests[1].CourseCode != "AT101" {
		t.Errorf("Requests[1] = %+v, want Elective AT101", result.Requests[1])
	}

	if len(result.Issues) != 0 {
		t.Errorf("Issues = %+v, want none", result.Issues)
	}
}

func TestLoad_BlankRequestTypeDefaultsToElectiveWithIssue(t *testing.T) {
	path := writeCSV(t, `student_id,student_name,grade,course_code,request_type
10057,Dana Cole,9,SCI101,
`)

	result, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Requests) != 1 {
		t.Fatalf("len(Requests) = %d, want 1", len(result.Requests))
	}
	if result.Requests[0].Type != domain.Elective {
		t.Errorf("Type = %v, want Elective (safe default for ambiguous request_type)", result.Requests[0].Type)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}
}

func TestLoad_DuplicateRowIsDedupedWithIssue(t *testing.T) {
	path := writeCSV(t, `student_id,student_name,grade,course_code,request_type
10042,Alex Rivera,9,AT101,ELECTIVE
10042,Alex Rivera,9,AT101,ELECTIVE
`)

	result, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Requests) != 1 {
		t.Fatalf("len(Requests) = %d, want 1 (duplicate deduped)", len(result.Requests))
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}
}

func TestLoad_RankIsPerStudentPerType(t *testing.T) {
	path := writeCSV(t, `student_id,student_name,grade,course_code,request_type
10042,Alex Rivera,9,MTH101,REQUIRED
10042,Alex Rivera,9,ENG101,REQUIRED
10042,Alex Rivera,9,AT101,ELECTIVE
`)

	result, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Requests) != 3 {
		t.Fatalf("len(Requests) = %d, want 3", len(result.Requests))
	}
	if result.Requests[0].Rank != 0 || result.Requests[1].Rank != 1 {
		t.Errorf("required ranks = %d, %d, want 0, 1", result.Requests[0].Rank, result.Requests[1].Rank)
	}
	if result.Requests[2].Rank != 0 {
		t.Errorf("elective rank = %d, want 0 (separate group from required)", result.Requests[2].Rank)
	}
}

func TestLoad_UnparseableGradeDefaultsToZeroWithIssue(t *testing.T) {
	path := writeCSV(t, `student_id,student_name,grade,course_code,request_type
10099,Jordan Lee,N/A,MTH101,REQUIRED
`)

	result, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Students) != 1 || result.Students[0].Grade != 0 {
		t.Fatalf("Students = %+v, want one student with Grade 0", result.Students)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.csv"))
	if err == nil {
		t.Fatal("Load() error = nil, want error for missing file")
	}
}

func TestLoad_RealFixture(t *testing.T) {
	result, err := Load(filepath.Join("..", "..", "..", "data", "meridian", "course_requests.csv"))
	if err != nil {
		t.Fatalf("Load() real fixture: %v", err)
	}

	if len(result.Students) != 5 {
		t.Errorf("len(Students) = %d, want 5", len(result.Students))
	}
	if len(result.Requests) != 24 {
		t.Errorf("len(Requests) = %d, want 24 (25 data rows, minus one deduped AT101 duplicate)", len(result.Requests))
	}
	if len(result.Issues) != 2 {
		t.Errorf("len(Issues) = %d, want 2 (the AT101 duplicate + the blank request_type row)", len(result.Issues))
	}
}
