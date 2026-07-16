// Package meridian parses Meridian High's flat course_requests.csv export.
package meridian

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"timely-take-home/internal/domain"
	"timely-take-home/internal/source"
)

const sourceName = "meridian"

// Load parses a Meridian course_requests.csv file into a source.Result.
func Load(path string) (source.Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return source.Result{}, fmt.Errorf("opening meridian requests %s: %w", path, err)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return source.Result{}, fmt.Errorf("parsing meridian requests %s: %w", path, err)
	}
	if len(rows) == 0 {
		return source.Result{}, fmt.Errorf("meridian requests %s: empty file", path)
	}

	col, err := columnIndex(rows[0], "student_id", "student_name", "grade", "course_code", "request_type")
	if err != nil {
		return source.Result{}, fmt.Errorf("meridian requests %s: %w", path, err)
	}

	var result source.Result
	seenStudents := make(map[string]bool)
	seenRequests := make(map[string]bool)
	rankCounter := make(map[string]int)

	for _, row := range rows[1:] {
		studentRawID := strings.TrimSpace(row[col["student_id"]])
		courseCode := strings.TrimSpace(row[col["course_code"]])

		if studentRawID == "" || courseCode == "" {
			result.Issues = append(result.Issues, source.Issue{
				Source:  sourceName,
				Student: studentRawID,
				Detail:  "row missing student_id or course_code; skipping",
			})
			continue
		}

		studentID := domain.StudentKey(sourceName, studentRawID)
		if !seenStudents[studentID] {
			grade, _ := strconv.Atoi(strings.TrimSpace(row[col["grade"]]))
			result.Students = append(result.Students, domain.Student{
				ID:     studentID,
				Name:   strings.TrimSpace(row[col["student_name"]]),
				Grade:  grade,
				Source: sourceName,
			})
			seenStudents[studentID] = true
		}

		reqType, ok := parseRequestType(row[col["request_type"]])
		if !ok {
			result.Issues = append(result.Issues, source.Issue{
				Source:  sourceName,
				Student: studentRawID,
				Detail:  fmt.Sprintf("unrecognized request_type %q for course %s; defaulting to elective", row[col["request_type"]], courseCode),
			})
		}

		dedupeKey := studentID + "|" + courseCode + "|" + string(reqType)
		if seenRequests[dedupeKey] {
			result.Issues = append(result.Issues, source.Issue{
				Source:  sourceName,
				Student: studentRawID,
				Detail:  fmt.Sprintf("duplicate request for course %s; keeping first occurrence", courseCode),
			})
			continue
		}
		seenRequests[dedupeKey] = true

		rankKey := studentID + "|" + string(reqType)
		rank := rankCounter[rankKey]
		rankCounter[rankKey] = rank + 1

		result.Requests = append(result.Requests, domain.CourseRequest{
			StudentID:  studentID,
			CourseCode: courseCode,
			Type:       reqType,
			Rank:       rank,
			Source:     sourceName,
		})
	}

	return result, nil
}

// parseRequestType maps Meridian's REQUIRED/ELECTIVE vocabulary to the
// unified RequestType. Anything else (blank, unrecognized) defaults to
// Elective rather than Required, since silently promoting an ambiguous
// request to a graduation requirement is the riskier wrong guess.
func parseRequestType(raw string) (domain.RequestType, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "REQUIRED":
		return domain.Required, true
	case "ELECTIVE":
		return domain.Elective, true
	default:
		return domain.Elective, false
	}
}

func columnIndex(header []string, required ...string) (map[string]int, error) {
	idx := make(map[string]int, len(header))
	for i, name := range header {
		idx[strings.TrimSpace(name)] = i
	}
	for _, name := range required {
		if _, ok := idx[name]; !ok {
			return nil, fmt.Errorf("missing expected column %q", name)
		}
	}
	return idx, nil
}
