package helix

import (
	"os"
	"path/filepath"
	"testing"

	"timely-take-home/internal/domain"
)

func writeFixtures(t *testing.T, rosterJSON, courseMapJSON string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	rosterPath := filepath.Join(dir, "roster_export.json")
	mapPath := filepath.Join(dir, "course_map.json")
	if err := os.WriteFile(rosterPath, []byte(rosterJSON), 0o644); err != nil {
		t.Fatalf("writing roster fixture: %v", err)
	}
	if err := os.WriteFile(mapPath, []byte(courseMapJSON), 0o644); err != nil {
		t.Fatalf("writing course map fixture: %v", err)
	}
	return rosterPath, mapPath
}

const courseMap = `{"a1b2c3": "MTH401", "c3d4e5": "ENG401", "e5f6a7": "ENG402", "f6a7b8": "AT301"}`

func TestLoad_HappyPath(t *testing.T) {
	rosterPath, mapPath := writeFixtures(t, `{"roster": [
		{"studentId": "HX-1", "name": "Liam O'Brien", "grade": 12,
		 "courseRequests": [{"helixCourseUuid": "a1b2c3", "isRequired": true}],
		 "alternates": [{"helixCourseUuid": "f6a7b8"}]}
	]}`, courseMap)

	result, err := Load(rosterPath, mapPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Students) != 1 {
		t.Fatalf("len(Students) = %d, want 1", len(result.Students))
	}
	wantStudent := domain.Student{ID: "helix:HX-1", Name: "Liam O'Brien", Grade: 12, Source: "helix"}
	if result.Students[0] != wantStudent {
		t.Errorf("Students[0] = %+v, want %+v", result.Students[0], wantStudent)
	}

	if len(result.Requests) != 2 {
		t.Fatalf("len(Requests) = %d, want 2", len(result.Requests))
	}
	if got := result.Requests[0]; got.CourseCode != "MTH401" || got.Type != domain.Required {
		t.Errorf("Requests[0] = %+v, want CourseCode MTH401, Required", got)
	}
	if got := result.Requests[1]; got.CourseCode != "AT301" || got.Type != domain.Elective {
		t.Errorf("Requests[1] = %+v, want CourseCode AT301, Elective (from alternates)", got)
	}
	if len(result.Issues) != 0 {
		t.Errorf("Issues = %+v, want none", result.Issues)
	}
}

func TestLoad_ChainSequenceStaysOrderedAndContiguous(t *testing.T) {
	// Chain items appear out of chainSeq order and non-contiguously in the
	// file; both must be corrected: pulled together and reordered by
	// chainSeq, anchored at the earliest position either chain member
	// appears at.
	rosterPath, mapPath := writeFixtures(t, `{"roster": [
		{"studentId": "HX-1", "name": "Liam O'Brien", "grade": 12,
		 "courseRequests": [
			{"helixCourseUuid": "e5f6a7", "isRequired": true, "chainId": "CHAIN-7", "chainSeq": 2},
			{"helixCourseUuid": "a1b2c3", "isRequired": true},
			{"helixCourseUuid": "c3d4e5", "isRequired": true, "chainId": "CHAIN-7", "chainSeq": 1}
		 ],
		 "alternates": []}
	]}`, courseMap)

	result, err := Load(rosterPath, mapPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Requests) != 3 {
		t.Fatalf("len(Requests) = %d, want 3", len(result.Requests))
	}
	// The chain (e5f6a7 seq2, c3d4e5 seq1) is anchored at index 0 (e5f6a7's
	// file position), so it comes first, reordered seq1 -> seq2, then the
	// unchained a1b2c3.
	wantOrder := []string{"ENG401", "ENG402", "MTH401"}
	for i, want := range wantOrder {
		if result.Requests[i].CourseCode != want {
			t.Errorf("Requests[%d].CourseCode = %q, want %q (order: %v)", i, result.Requests[i].CourseCode, want, requestCodes(result.Requests))
		}
	}
}

func TestLoad_InterleavedChainsPullMembersForwardToTheirAnchor(t *testing.T) {
	// File order: chainA-seq1(0), unrelated(1), chainB-seq1(2), chainB-seq2(3),
	// unrelated(4), chainA-seq2(5). Both chains' second halves jump forward
	// to sit right after their anchor, ahead of items that were originally
	// between them and the anchor in the file — the documented tradeoff in
	// chainOrderedIndices: sequence contiguity wins over everything else's
	// original relative position.
	rosterPath, mapPath := writeFixtures(t, `{"roster": [
		{"studentId": "HX-1", "name": "Test Student", "grade": 12,
		 "courseRequests": [
			{"helixCourseUuid": "a1b2c3", "isRequired": true, "chainId": "A", "chainSeq": 1},
			{"helixCourseUuid": "f6a7b8", "isRequired": true},
			{"helixCourseUuid": "c3d4e5", "isRequired": true, "chainId": "B", "chainSeq": 1},
			{"helixCourseUuid": "e5f6a7", "isRequired": true, "chainId": "B", "chainSeq": 2},
			{"helixCourseUuid": "e5f6a7", "isRequired": true},
			{"helixCourseUuid": "a1b2c3", "isRequired": true, "chainId": "A", "chainSeq": 2}
		 ],
		 "alternates": []}
	]}`, courseMap)

	result, err := Load(rosterPath, mapPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Chain A (both seq members map to MTH401 in this fixture) is anchored
	// at index 0 and stays contiguous: its seq-2 member (file index 5)
	// jumps ahead of the unrelated item at index 1 and all of chain B.
	wantOrder := []string{"MTH401", "MTH401", "AT301", "ENG401", "ENG402", "ENG402"}
	got := requestCodes(result.Requests)
	if len(got) != len(wantOrder) {
		t.Fatalf("got %v, want %v", got, wantOrder)
	}
	for i, want := range wantOrder {
		if got[i] != want {
			t.Errorf("Requests[%d].CourseCode = %q, want %q (full order: %v)", i, got[i], want, got)
		}
	}
}

func requestCodes(reqs []domain.CourseRequest) []string {
	codes := make([]string, len(reqs))
	for i, r := range reqs {
		codes[i] = r.CourseCode
	}
	return codes
}

func TestLoad_DanglingUUIDKeepsRawIdentifierWithIssue(t *testing.T) {
	rosterPath, mapPath := writeFixtures(t, `{"roster": [
		{"studentId": "HX-560", "name": "Grace Liu", "grade": 11,
		 "courseRequests": [{"helixCourseUuid": "zz9zz9", "isRequired": true}],
		 "alternates": []}
	]}`, courseMap)

	result, err := Load(rosterPath, mapPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(result.Requests) != 1 {
		t.Fatalf("len(Requests) = %d, want 1", len(result.Requests))
	}
	if result.Requests[0].CourseCode != "zz9zz9" {
		t.Errorf("CourseCode = %q, want raw uuid kept when course_map.json has no entry", result.Requests[0].CourseCode)
	}
	if result.Requests[0].Type != domain.Required {
		t.Errorf("Type = %v, want Required — a dangling reference must not be conflated with a backup", result.Requests[0].Type)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}
}

func TestLoad_MissingFiles(t *testing.T) {
	dir := t.TempDir()
	missingRoster := filepath.Join(dir, "roster_export.json")
	missingMap := filepath.Join(dir, "course_map.json")

	if _, err := Load(missingRoster, missingMap); err == nil {
		t.Fatal("Load() error = nil, want error when both files are missing")
	}

	_, mapPath := writeFixtures(t, `{"roster": []}`, courseMap)
	if _, err := Load(missingRoster, mapPath); err == nil {
		t.Fatal("Load() error = nil, want error when roster is missing")
	}
}

func TestLoad_RealFixture(t *testing.T) {
	result, err := Load(
		filepath.Join("..", "..", "..", "data", "helix", "roster_export.json"),
		filepath.Join("..", "..", "..", "data", "helix", "course_map.json"),
	)
	if err != nil {
		t.Fatalf("Load() real fixture: %v", err)
	}

	if len(result.Students) != 3 {
		t.Errorf("len(Students) = %d, want 3", len(result.Students))
	}
	// Liam: 5 required + 1 alternate; Grace: 3 required (incl. dangling
	// zz9zz9) + 0 alternates; Omar: 2 required + 2 alternates = 13 total.
	if len(result.Requests) != 13 {
		t.Errorf("len(Requests) = %d, want 13", len(result.Requests))
	}
	if len(result.Issues) != 1 {
		t.Errorf("len(Issues) = %d, want 1 (Grace Liu's dangling zz9zz9 uuid)", len(result.Issues))
	}

	var liamRequired []string
	for _, r := range result.Requests {
		if r.StudentID == "helix:HX-558" && r.Type == domain.Required {
			liamRequired = append(liamRequired, r.CourseCode)
		}
	}
	want := []string{"MTH401", "ENG401", "ENG402", "SS402", "SCI402"}
	if len(liamRequired) != len(want) {
		t.Fatalf("Liam's required courses = %v, want %v", liamRequired, want)
	}
	for i, code := range want {
		if liamRequired[i] != code {
			t.Errorf("Liam's required[%d] = %q, want %q (chain CHAIN-7 must keep ENG401 before ENG402)", i, liamRequired[i], code)
		}
	}
}
