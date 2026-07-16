package catalog

import (
	"encoding/json"
	"fmt"
	"os"

	"timely-take-home/internal/domain"
)

// Catalog is the loaded district course catalog, indexed by course code.
type Catalog struct {
	courses map[string]domain.Course
}

type file struct {
	Courses []domain.Course `json:"courses"`
}

// Load reads and indexes the catalog from a courses.json file.
func Load(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading catalog %s: %w", path, err)
	}

	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing catalog %s: %w", path, err)
	}

	courses := make(map[string]domain.Course, len(f.Courses))
	for _, c := range f.Courses {
		courses[c.Code] = c
	}

	return &Catalog{courses: courses}, nil
}

// Lookup returns the course for a given catalog code, and whether it exists.
func (c *Catalog) Lookup(code string) (domain.Course, bool) {
	course, ok := c.courses[code]
	return course, ok
}

// Len returns the number of courses in the catalog.
func (c *Catalog) Len() int {
	return len(c.courses)
}
