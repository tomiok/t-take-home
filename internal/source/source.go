// Package source defines the shape every SIS adapter produces.
package source

import "timely-take-home/internal/domain"

// Result is what every SIS adapter produces: the students and course
// requests it extracted, plus any row-level issues encountered along the
// way (malformed or ambiguous data that didn't stop parsing).
type Result struct {
	Students []domain.Student
	Requests []domain.CourseRequest
	Issues   []Issue
}

// Issue is a non-fatal problem encountered while parsing a source feed —
// bad input the adapter recovered from rather than one that stopped it.
type Issue struct {
	Source  string
	Student string // raw source-local student ref, for traceability
	Detail  string
}
