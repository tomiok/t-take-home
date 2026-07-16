// Package apex parses Apex Charter's nested requests.json export, resolving
// its local course numbers via course_crosswalk.csv.
package apex

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"timely-take-home/internal/domain"
	"timely-take-home/internal/source"
)

const sourceName = "apex"

// backupSentinelRank is Apex's convention for flagging a backup/alternate
// request: rank 500 is not a real priority position.
const backupSentinelRank = 500

type requestsFile struct {
	Students []struct {
		StateStudentID string `json:"stateStudentId"`
		Name           string `json:"name"`
		GradeLevel     int    `json:"gradeLevel"`
		Requests       []struct {
			CourseNumber string `json:"courseNumber"`
			Rank         int    `json:"rank"`
		} `json:"requests"`
	} `json:"students"`
}

// Load parses Apex's requests.json and course_crosswalk.csv into a
// source.Result.
func Load(requestsPath, crosswalkPath string) (source.Result, error) {
	crosswalk, err := loadCrosswalk(crosswalkPath)
	if err != nil {
		return source.Result{}, err
	}

	data, err := os.ReadFile(requestsPath)
	if err != nil {
		return source.Result{}, fmt.Errorf("reading apex requests %s: %w", requestsPath, err)
	}

	var f requestsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return source.Result{}, fmt.Errorf("parsing apex requests %s: %w", requestsPath, err)
	}

	var result source.Result
	rankCounter := make(map[string]int)

	for _, s := range f.Students {
		studentID := domain.StudentKey(sourceName, s.StateStudentID)
		result.Students = append(result.Students, domain.Student{
			ID:     studentID,
			Name:   s.Name,
			Grade:  s.GradeLevel,
			Source: sourceName,
		})

		for _, req := range s.Requests {
			courseCode, resolved := crosswalk[req.CourseNumber]
			if !resolved {
				courseCode = req.CourseNumber
				result.Issues = append(result.Issues, source.Issue{
					Source:  sourceName,
					Student: s.StateStudentID,
					Detail:  fmt.Sprintf("apex course number %q has no course_crosswalk.csv entry; keeping raw identifier", req.CourseNumber),
				})
			}

			reqType := domain.Required
			rank := req.Rank - 1
			if req.Rank == backupSentinelRank {
				reqType = domain.Elective
				rankKey := studentID + "|" + string(reqType)
				rank = rankCounter[rankKey]
				rankCounter[rankKey] = rank + 1
			}

			result.Requests = append(result.Requests, domain.CourseRequest{
				StudentID:  studentID,
				CourseCode: courseCode,
				Type:       reqType,
				Rank:       rank,
				Source:     sourceName,
			})
		}
	}

	return result, nil
}

func loadCrosswalk(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening apex crosswalk %s: %w", path, err)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing apex crosswalk %s: %w", path, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("apex crosswalk %s: empty file", path)
	}

	crosswalk := make(map[string]string, len(rows)-1)
	for _, row := range rows[1:] {
		crosswalk[row[0]] = row[1]
	}
	return crosswalk, nil
}
