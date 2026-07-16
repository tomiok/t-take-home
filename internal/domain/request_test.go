package domain

import "testing"

func TestStudentKey(t *testing.T) {
	tests := []struct {
		name   string
		source string
		rawID  string
		want   string
	}{
		{name: "meridian numeric id", source: "meridian", rawID: "10042", want: "meridian:10042"},
		{name: "cornerstone email", source: "cornerstone", rawID: "marcus.thompson@cornerstone.edu", want: "cornerstone:marcus.thompson@cornerstone.edu"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StudentKey(tt.source, tt.rawID); got != tt.want {
				t.Errorf("StudentKey(%q, %q) = %q, want %q", tt.source, tt.rawID, got, tt.want)
			}
		})
	}
}
