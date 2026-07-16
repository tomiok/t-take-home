package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"timely-take-home/internal/pipeline"
)

func main() {
	dataDir := flag.String("data", "data", "path to the data directory containing catalog/, meridian/, apex/, cornerstone/, helix/")
	outPath := flag.String("out", filepath.Join("output", "unified_requests.json"), "path to write the unified course-request output to")
	flag.Parse()

	out, err := pipeline.Run(*dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if err := pipeline.Write(out, *outPath); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %d students, %d requests, %d issues to %s\n", len(out.Students), len(out.Requests), len(out.Issues), *outPath)
}
