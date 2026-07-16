// Package helix parses Helix Academy's denormalized roster_export.json,
// resolving its course UUIDs via course_map.json.
package helix

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"timely-take-home/internal/domain"
	"timely-take-home/internal/source"
)

const sourceName = "helix"

type rosterFile struct {
	Roster []rosterStudent `json:"roster"`
}

type rosterStudent struct {
	StudentID      string          `json:"studentId"`
	Name           string          `json:"name"`
	Grade          int             `json:"grade"`
	CourseRequests []courseRequest `json:"courseRequests"`
	Alternates     []struct {
		HelixCourseUUID string `json:"helixCourseUuid"`
	} `json:"alternates"`
}

type courseRequest struct {
	HelixCourseUUID string `json:"helixCourseUuid"`
	IsRequired      bool   `json:"isRequired"`
	ChainID         string `json:"chainId"`
	ChainSeq        int    `json:"chainSeq"`
}

// Load parses Helix's roster_export.json and course_map.json into a
// source.Result.
func Load(rosterPath, courseMapPath string) (source.Result, error) {
	courseMap, err := loadCourseMap(courseMapPath)
	if err != nil {
		return source.Result{}, err
	}

	data, err := os.ReadFile(rosterPath)
	if err != nil {
		return source.Result{}, fmt.Errorf("reading helix roster %s: %w", rosterPath, err)
	}

	var f rosterFile
	if err := json.Unmarshal(data, &f); err != nil {
		return source.Result{}, fmt.Errorf("parsing helix roster %s: %w", rosterPath, err)
	}

	var result source.Result

	for _, s := range f.Roster {
		studentID := domain.StudentKey(sourceName, s.StudentID)
		result.Students = append(result.Students, domain.Student{
			ID:     studentID,
			Name:   s.Name,
			Grade:  s.Grade,
			Source: sourceName,
		})

		rankCounter := make(map[domain.RequestType]int)

		for _, idx := range chainOrderedIndices(s.CourseRequests) {
			cr := s.CourseRequests[idx]
			reqType := domain.Elective
			if cr.IsRequired {
				reqType = domain.Required
			}

			courseCode := resolveCourse(cr.HelixCourseUUID, courseMap, &result, s.StudentID)

			result.Requests = append(result.Requests, domain.CourseRequest{
				StudentID:  studentID,
				CourseCode: courseCode,
				Type:       reqType,
				Rank:       rankCounter[reqType],
				Source:     sourceName,
			})
			rankCounter[reqType]++
		}

		for _, alt := range s.Alternates {
			courseCode := resolveCourse(alt.HelixCourseUUID, courseMap, &result, s.StudentID)

			result.Requests = append(result.Requests, domain.CourseRequest{
				StudentID:  studentID,
				CourseCode: courseCode,
				Type:       domain.Elective,
				Rank:       rankCounter[domain.Elective],
				Source:     sourceName,
			})
			rankCounter[domain.Elective]++
		}
	}

	return result, nil
}

// resolveCourse maps a Helix course UUID to its catalog code, logging an
// Issue and keeping the raw UUID when course_map.json has no entry for it
// (e.g. a dangling reference) rather than dropping the request.
func resolveCourse(uuid string, courseMap map[string]string, result *source.Result, studentRawID string) string {
	code, ok := courseMap[uuid]
	if ok {
		return code
	}
	result.Issues = append(result.Issues, source.Issue{
		Source:  sourceName,
		Student: studentRawID,
		Detail:  fmt.Sprintf("helix course uuid %q has no course_map.json entry; keeping raw identifier", uuid),
	})
	return uuid
}

// chainOrderedIndices returns request indices ordered so that a multi-term
// sequence (shared chainId) stays contiguous and in chainSeq order, anchored
// at the earliest file position any of its members appear at. Requests
// without a chainId keep plain file order.
func chainOrderedIndices(requests []courseRequest) []int {
	anchor := make(map[string]int)
	for i, r := range requests {
		if r.ChainID == "" {
			continue
		}
		if existing, ok := anchor[r.ChainID]; !ok || i < existing {
			anchor[r.ChainID] = i
		}
	}

	indices := make([]int, len(requests))
	for i := range requests {
		indices[i] = i
	}

	sortKey := func(i int) int {
		if r := requests[i]; r.ChainID != "" {
			return anchor[r.ChainID]
		}
		return i
	}

	sort.SliceStable(indices, func(a, b int) bool {
		ia, ib := indices[a], indices[b]
		keyA, keyB := sortKey(ia), sortKey(ib)
		if keyA != keyB {
			return keyA < keyB
		}
		return requests[ia].ChainSeq < requests[ib].ChainSeq
	})

	return indices
}

func loadCourseMap(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading helix course map %s: %w", path, err)
	}

	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing helix course map %s: %w", path, err)
	}
	return m, nil
}
