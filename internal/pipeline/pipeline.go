// Package pipeline runs all four SIS adapters against a data directory and
// combines their output into one unified set the application works with.
package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"timely-take-home/internal/domain"
	"timely-take-home/internal/source"
	"timely-take-home/internal/source/apex"
	"timely-take-home/internal/source/cornerstone"
	"timely-take-home/internal/source/helix"
	"timely-take-home/internal/source/meridian"
)

// Output is the unified course-request data the application works with,
// independent of which SIS each student/request came from.
//
// No cross-source student matching happens here: each of the four SIS
// feeds describes a disjoint set of students (one school per SIS), and
// domain.StudentKey already namespaces every ID by source, so there's no
// possibility of two sources colliding on the same Student.ID.
type Output struct {
	Students []domain.Student       `json:"students"`
	Requests []domain.CourseRequest `json:"requests"`
	Issues   []source.Issue         `json:"issues"`
}

// Run loads all four SIS feeds from the conventional layout under dataDir
// (dataDir/meridian, dataDir/apex, dataDir/cornerstone, dataDir/helix) and
// combines them into one Output.
func Run(dataDir string) (Output, error) {
	meridianResult, err := meridian.Load(filepath.Join(dataDir, "meridian", "course_requests.csv"))
	if err != nil {
		return Output{}, fmt.Errorf("loading meridian: %w", err)
	}

	apexResult, err := apex.Load(
		filepath.Join(dataDir, "apex", "requests.json"),
		filepath.Join(dataDir, "apex", "course_crosswalk.csv"),
	)
	if err != nil {
		return Output{}, fmt.Errorf("loading apex: %w", err)
	}

	cornerstoneResult, err := cornerstone.Load(
		filepath.Join(dataDir, "cornerstone", "primary_requests.csv"),
		filepath.Join(dataDir, "cornerstone", "alternate_requests.csv"),
	)
	if err != nil {
		return Output{}, fmt.Errorf("loading cornerstone: %w", err)
	}

	helixResult, err := helix.Load(
		filepath.Join(dataDir, "helix", "roster_export.json"),
		filepath.Join(dataDir, "helix", "course_map.json"),
	)
	if err != nil {
		return Output{}, fmt.Errorf("loading helix: %w", err)
	}

	var out Output
	for _, result := range []source.Result{meridianResult, apexResult, cornerstoneResult, helixResult} {
		out.Students = append(out.Students, result.Students...)
		out.Requests = append(out.Requests, result.Requests...)
		out.Issues = append(out.Issues, result.Issues...)
	}

	return out, nil
}

// Write marshals the unified Output as indented JSON to path.
func Write(out Output, path string) error {
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling unified output: %w", err)
	}
	if err = os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing unified output to %s: %w", path, err)
	}
	return nil
}
