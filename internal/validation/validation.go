// Package validation checks unified course requests against the catalog:
// every request must point at a real course, and a student who requests
// one half of a corequisite pair should request the other half too.
package validation

import (
	"sort"

	"timely-take-home/internal/catalog"
	"timely-take-home/internal/domain"
)

// UnknownCourseFinding flags a request whose CourseCode doesn't resolve to
// any course in the catalog.
type UnknownCourseFinding struct {
	StudentID  string
	CourseCode string
}

// MissingCorequisiteFinding flags a student who has requested HasCourse but
// not its corequisite MissingCourse.
type MissingCorequisiteFinding struct {
	StudentID     string
	HasCourse     string
	MissingCourse string
}

// Report holds every validation finding across a set of requests.
type Report struct {
	UnknownCourses      []UnknownCourseFinding
	MissingCorequisites []MissingCorequisiteFinding
}

// Validate checks requests against cat: every CourseCode should resolve to
// a real course, and every corequisite pair a student has one half of
// should have its other half too.
//
// The catalog declares corequisites one-directionally (e.g. SCI301 lists
// MTH202 but not vice versa) — this is the layer that treats a declared
// pair as symmetric in either direction, as intended when the catalog
// loader was kept faithful to the raw (asymmetric) source data.
func Validate(cat *catalog.Catalog, requests []domain.CourseRequest) Report {
	var report Report

	studentCourses := make(map[string]map[string]bool)
	for _, req := range requests {
		if _, ok := cat.Lookup(req.CourseCode); !ok {
			report.UnknownCourses = append(report.UnknownCourses, UnknownCourseFinding{
				StudentID:  req.StudentID,
				CourseCode: req.CourseCode,
			})
			continue
		}
		if studentCourses[req.StudentID] == nil {
			studentCourses[req.StudentID] = make(map[string]bool)
		}
		studentCourses[req.StudentID][req.CourseCode] = true
	}

	pairs := symmetricCorequisites(cat)

	for _, studentID := range sortedKeys(studentCourses) {
		courses := studentCourses[studentID]
		for _, code := range sortedSet(courses) {
			for _, paired := range pairs[code] {
				if !courses[paired] {
					report.MissingCorequisites = append(report.MissingCorequisites, MissingCorequisiteFinding{
						StudentID:     studentID,
						HasCourse:     code,
						MissingCourse: paired,
					})
				}
			}
		}
	}

	return report
}

// symmetricCorequisites builds a course code -> paired codes map, treating
// every declared corequisite relationship as symmetric regardless of which
// direction the catalog happened to declare it in.
func symmetricCorequisites(cat *catalog.Catalog) map[string][]string {
	pairs := make(map[string]map[string]bool)
	add := func(a, b string) {
		if pairs[a] == nil {
			pairs[a] = make(map[string]bool)
		}
		pairs[a][b] = true
	}

	for _, course := range cat.All() {
		for _, coreq := range course.Corequisites {
			add(course.Code, coreq)
			add(coreq, course.Code)
		}
	}

	result := make(map[string][]string, len(pairs))
	for code, set := range pairs {
		result[code] = sortedSet(set)
	}
	return result
}

func sortedKeys(m map[string]map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedSet(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
