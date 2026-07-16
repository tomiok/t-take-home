package domain

// Course is a single catalog entry — the source of truth every SIS feed's
// course reference must ultimately resolve against.
type Course struct {
	Code          string   `json:"code"`
	Name          string   `json:"name"`
	Department    string   `json:"department"`
	Grades        []int    `json:"grades"`
	Prerequisites []string `json:"prerequisites"`
	Corequisites  []string `json:"corequisites"`
}
