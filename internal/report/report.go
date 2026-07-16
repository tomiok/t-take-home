// Package report builds the application-facing views the CLI prints:
// a single student's requests, split and ordered, with catalog names
// resolved where possible.
package report

import (
	"sort"

	"timely-take-home/internal/catalog"
	"timely-take-home/internal/domain"
	"timely-take-home/internal/pipeline"
)

// CourseView is a single course reference resolved (or not) against the
// catalog, ready to print.
type CourseView struct {
	CourseCode string
	CourseName string
	Resolved   bool
}

// StudentSummary is one student's course requests, split into required and
// elective/backup, each in sensible (Rank) order.
type StudentSummary struct {
	Student  domain.Student
	Required []CourseView
	Elective []CourseView
}

// BuildStudentSummary finds studentID in out and returns their requests
// split by RequestType and ordered by Rank, with each CourseCode resolved
// against cat. Returns false if no student with that ID exists.
func BuildStudentSummary(out pipeline.Output, cat *catalog.Catalog, studentID string) (StudentSummary, bool) {
	var student domain.Student
	found := false
	for _, s := range out.Students {
		if s.ID == studentID {
			student = s
			found = true
			break
		}
	}
	if !found {
		return StudentSummary{}, false
	}

	var required, elective []domain.CourseRequest
	for _, r := range out.Requests {
		if r.StudentID != studentID {
			continue
		}
		if r.Type == domain.Required {
			required = append(required, r)
		} else {
			elective = append(elective, r)
		}
	}

	byRank := func(reqs []domain.CourseRequest) {
		sort.SliceStable(reqs, func(i, j int) bool { return reqs[i].Rank < reqs[j].Rank })
	}
	byRank(required)
	byRank(elective)

	return StudentSummary{
		Student:  student,
		Required: toViews(required, cat),
		Elective: toViews(elective, cat),
	}, true
}

func toViews(reqs []domain.CourseRequest, cat *catalog.Catalog) []CourseView {
	views := make([]CourseView, 0, len(reqs))
	for _, r := range reqs {
		course, ok := cat.Lookup(r.CourseCode)
		view := CourseView{CourseCode: r.CourseCode, Resolved: ok}
		if ok {
			view.CourseName = course.Name
		}
		views = append(views, view)
	}
	return views
}
