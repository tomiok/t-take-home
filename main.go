package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"timely-take-home/internal/catalog"
	"timely-take-home/internal/pipeline"
	"timely-take-home/internal/report"
	"timely-take-home/internal/validation"
)

func main() {
	args := os.Args[1:]
	cmd := "run"
	if len(args) > 0 {
		switch args[0] {
		case "run", "student", "validate":
			cmd = args[0]
			args = args[1:]
		}
	}

	switch cmd {
	case "student":
		runStudent(args)
	case "validate":
		runValidate(args)
	default:
		runPipeline(args)
	}
}

func runPipeline(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	dataDir := fs.String("data", "data", "path to the data directory containing catalog/, meridian/, apex/, cornerstone/, helix/")
	outPath := fs.String("out", filepath.Join("output", "unified_requests.json"), "path to write the unified course-request output to")
	fs.Parse(args)

	out, err := pipeline.Run(*dataDir)
	if err != nil {
		fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fatal(err)
	}
	if err := pipeline.Write(out, *outPath); err != nil {
		fatal(err)
	}

	fmt.Printf("wrote %d students, %d requests, %d issues to %s\n", len(out.Students), len(out.Requests), len(out.Issues), *outPath)
}

func runStudent(args []string) {
	fs := flag.NewFlagSet("student", flag.ExitOnError)
	dataDir := fs.String("data", "data", "path to the data directory")
	fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: student [-data DIR] <student-id>  (e.g. meridian:10042)")
		os.Exit(1)
	}
	studentID := fs.Arg(0)

	out, cat := loadPipelineAndCatalog(*dataDir)

	summary, found := report.BuildStudentSummary(out, cat, studentID)
	if !found {
		fmt.Fprintf(os.Stderr, "no student found with id %q\n", studentID)
		os.Exit(1)
	}

	name := summary.Student.Name
	if name == "" {
		name = "(name unavailable)"
	}
	fmt.Printf("%s (%s), grade %d\n\n", name, summary.Student.ID, summary.Student.Grade)

	printCourseViews("Required", summary.Required)
	fmt.Println()
	printCourseViews("Elective / backup", summary.Elective)
}

func printCourseViews(label string, views []report.CourseView) {
	fmt.Printf("%s:\n", label)
	if len(views) == 0 {
		fmt.Println("  (none)")
		return
	}
	for i, v := range views {
		if v.Resolved {
			fmt.Printf("  %d. %s - %s\n", i+1, v.CourseCode, v.CourseName)
		} else {
			fmt.Printf("  %d. %s - (unknown course code, not in catalog)\n", i+1, v.CourseCode)
		}
	}
}

func runValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	dataDir := fs.String("data", "data", "path to the data directory")
	fs.Parse(args)

	out, cat := loadPipelineAndCatalog(*dataDir)
	result := validation.Validate(cat, out.Requests)

	fmt.Printf("Parsing issues (%d):\n", len(out.Issues))
	for _, issue := range out.Issues {
		fmt.Printf("  [%s] student %s: %s\n", issue.Source, issue.Student, issue.Detail)
	}

	fmt.Printf("\nUnknown course codes (%d):\n", len(result.UnknownCourses))
	for _, f := range result.UnknownCourses {
		fmt.Printf("  student %s requested unknown course %q\n", f.StudentID, f.CourseCode)
	}

	fmt.Printf("\nMissing corequisites (%d):\n", len(result.MissingCorequisites))
	for _, f := range result.MissingCorequisites {
		fmt.Printf("  student %s has %s but not its corequisite %s\n", f.StudentID, f.HasCourse, f.MissingCourse)
	}
}

func loadPipelineAndCatalog(dataDir string) (pipeline.Output, *catalog.Catalog) {
	out, err := pipeline.Run(dataDir)
	if err != nil {
		fatal(err)
	}
	cat, err := catalog.Load(filepath.Join(dataDir, "catalog", "courses.json"))
	if err != nil {
		fatal(err)
	}
	return out, cat
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
