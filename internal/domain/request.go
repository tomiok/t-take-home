package domain

// RequestType is the normalized required/elective distinction every source
// feed's own vocabulary (REQUIRED/ELECTIVE, rank 500, primary/alternate file,
// isRequired) maps into.
type RequestType string

const (
	Required RequestType = "required"
	Elective RequestType = "elective"
)

// Student is a district student, identified by a source-qualified ID since
// raw IDs (a local numeric ID, an email, a state ID) are only unique within
// their own SIS. Use StudentKey to build that ID consistently.
type Student struct {
	ID     string
	Name   string
	Grade  int
	Source string
}

// StudentKey builds the source-qualified student ID used as Student.ID and
// CourseRequest.StudentID, so every adapter joins students the same way
// instead of each inventing its own qualification scheme.
func StudentKey(source, rawID string) string {
	return source + ":" + rawID
}

// CourseRequest is the unified shape every SIS adapter produces, regardless
// of the source feed's own format.
//
// CourseCode holds the catalog code when the source's own identifier could
// be resolved to one; otherwise it holds the unresolved source-local
// identifier (a raw course number or UUID) so downstream validation can
// still report which request is broken and with what value.
type CourseRequest struct {
	StudentID  string
	CourseCode string
	Type       RequestType
	// Rank orders a student's requests within their RequestType group;
	// lower ranks come first. Its meaning (explicit priority, chain
	// sequence, or plain file order) is decided per-source by the adapter
	// that produces it.
	Rank   int
	Source string
	// Terms is source-specific scheduling metadata (e.g. Cornerstone's
	// FULL/S1/Q1;Q2); empty for sources that don't express it.
	Terms string
}
