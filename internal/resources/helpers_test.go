package resources

import (
	"testing"
)

func TestParseImportID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSite string
		wantRes  string
		wantOK   bool
	}{
		{
			name:     "valid two-part ID",
			input:    "site123/resource456",
			wantSite: "site123",
			wantRes:  "resource456",
			wantOK:   true,
		},
		{
			name:     "valid with real Omada IDs",
			input:    "696a40fd49039e1d13a9c3f9/696a4b9149039e1d13a9c5f4",
			wantSite: "696a40fd49039e1d13a9c3f9",
			wantRes:  "696a4b9149039e1d13a9c5f4",
			wantOK:   true,
		},
		{
			name:     "valid with MAC address",
			input:    "696a40fd49039e1d13a9c3f9/9C-A2-F4-00-08-12",
			wantSite: "696a40fd49039e1d13a9c3f9",
			wantRes:  "9C-A2-F4-00-08-12",
			wantOK:   true,
		},
		{
			name:   "empty string",
			input:  "",
			wantOK: false,
		},
		{
			name:   "no separator",
			input:  "site123resource456",
			wantOK: false,
		},
		{
			name:   "empty site ID",
			input:  "/resource456",
			wantOK: false,
		},
		{
			name:   "empty resource ID",
			input:  "site123/",
			wantOK: false,
		},
		{
			name:     "three parts — only splits on first slash",
			input:    "site123/part1/part2",
			wantSite: "site123",
			wantRes:  "part1/part2",
			wantOK:   true,
		},
		{
			name:   "just a slash",
			input:  "/",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSite, gotRes, gotOK := parseImportID(tt.input)
			if gotOK != tt.wantOK {
				t.Fatalf("parseImportID(%q) ok = %v, want %v", tt.input, gotOK, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if gotSite != tt.wantSite {
				t.Errorf("parseImportID(%q) siteID = %q, want %q", tt.input, gotSite, tt.wantSite)
			}
			if gotRes != tt.wantRes {
				t.Errorf("parseImportID(%q) resourceID = %q, want %q", tt.input, gotRes, tt.wantRes)
			}
		})
	}
}

func TestParseImportID3(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantSite  string
		wantPart1 string
		wantPart2 string
		wantOK    bool
	}{
		{
			name:      "valid three-part ID",
			input:     "site123/group456/ssid789",
			wantSite:  "site123",
			wantPart1: "group456",
			wantPart2: "ssid789",
			wantOK:    true,
		},
		{
			name:      "valid with real Omada IDs",
			input:     "696a40fd49039e1d13a9c3f9/696a40fd49039e1d13a9c412/696a4c3549039e1d13a9c61b",
			wantSite:  "696a40fd49039e1d13a9c3f9",
			wantPart1: "696a40fd49039e1d13a9c412",
			wantPart2: "696a4c3549039e1d13a9c61b",
			wantOK:    true,
		},
		{
			name:   "empty string",
			input:  "",
			wantOK: false,
		},
		{
			name:   "only two parts",
			input:  "site123/group456",
			wantOK: false,
		},
		{
			name:   "empty site ID",
			input:  "/group456/ssid789",
			wantOK: false,
		},
		{
			name:   "empty part1",
			input:  "site123//ssid789",
			wantOK: false,
		},
		{
			name:   "empty part2",
			input:  "site123/group456/",
			wantOK: false,
		},
		{
			name:   "no separators",
			input:  "site123group456ssid789",
			wantOK: false,
		},
		{
			name:      "four parts — last two joined in part2",
			input:     "site/group/ssid/extra",
			wantSite:  "site",
			wantPart1: "group",
			wantPart2: "ssid/extra",
			wantOK:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSite, gotPart1, gotPart2, gotOK := parseImportID3(tt.input)
			if gotOK != tt.wantOK {
				t.Fatalf("parseImportID3(%q) ok = %v, want %v", tt.input, gotOK, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if gotSite != tt.wantSite {
				t.Errorf("parseImportID3(%q) siteID = %q, want %q", tt.input, gotSite, tt.wantSite)
			}
			if gotPart1 != tt.wantPart1 {
				t.Errorf("parseImportID3(%q) part1 = %q, want %q", tt.input, gotPart1, tt.wantPart1)
			}
			if gotPart2 != tt.wantPart2 {
				t.Errorf("parseImportID3(%q) part2 = %q, want %q", tt.input, gotPart2, tt.wantPart2)
			}
		})
	}
}
