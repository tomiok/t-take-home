// Package cornerstone parses Cornerstone Prep's two CSV exports —
// primary_requests.csv (required) and alternate_requests.csv (elective) —
// the file a request appears in is itself the required/elective signal.
package cornerstone

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"timely-take-home/internal/domain"
	"timely-take-home/internal/source"
)

const sourceName = "cornerstone"

// Load parses Cornerstone's primary and alternate request CSVs into a
// source.Result. Cornerstone has no student name/grade in either file, so
// Student.Name and Student.Grade are left zero-valued.
func Load(primaryPath, alternatePath string) (source.Result, error) {
	var result source.Result
	seenStudents := make(map[string]bool)
	rankCounter := make(map[string]int)

	if err := loadFile(primaryPath, domain.Required, &result, seenStudents, rankCounter); err != nil {
		return source.Result{}, err
	}
	if err := loadFile(alternatePath, domain.Elective, &result, seenStudents, rankCounter); err != nil {
		return source.Result{}, err
	}

	return result, nil
}

func loadFile(path string, reqType domain.RequestType, result *source.Result, seenStudents map[string]bool, rankCounter map[string]int) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening cornerstone file %s: %w", path, err)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return fmt.Errorf("parsing cornerstone file %s: %w", path, err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("cornerstone file %s: empty file", path)
	}

	for _, row := range rows[1:] {
		rawEmail := row[0]
		courseCode := strings.TrimSpace(row[1])
		terms := ""
		if len(row) > 2 {
			terms = strings.TrimSpace(row[2])
		}

		// Raw emails have case/whitespace noise (e.g. " Marcus.Thompson@cornerstone.edu ");
		// normalize before using as the join key, or the same student
		// splits into two.
		email := strings.ToLower(strings.TrimSpace(rawEmail))
		if email == "" || courseCode == "" {
			result.Issues = append(result.Issues, source.Issue{
				Source:  sourceName,
				Student: rawEmail,
				Detail:  "row missing student_email or course_code; skipping",
			})
			continue
		}

		studentID := domain.StudentKey(sourceName, email)
		if !seenStudents[studentID] {
			result.Students = append(result.Students, domain.Student{
				ID:     studentID,
				Source: sourceName,
			})
			seenStudents[studentID] = true
		}

		rankKey := studentID + "|" + string(reqType)
		rank := rankCounter[rankKey]
		rankCounter[rankKey] = rank + 1

		result.Requests = append(result.Requests, domain.CourseRequest{
			StudentID:  studentID,
			CourseCode: courseCode,
			Type:       reqType,
			Rank:       rank,
			Source:     sourceName,
			Terms:      terms,
		})
	}

	return nil
}
