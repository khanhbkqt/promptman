package testing

import (
	"testing"
)

func TestMatchKey_Specific(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		requestID string
		want      bool
	}{
		{"exact match", "users/list", "users/list", true},
		{"no match different path", "users/list", "users/get", false},
		{"no match prefix only", "users", "users/list", false},
		{"root level exact", "health", "health", true},
		{"case sensitive", "Users/List", "users/list", false},
		{"empty pattern", "", "users/list", false},
		{"empty request", "users/list", "", false},
		{"both empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchKey(tt.pattern, tt.requestID)
			if got != tt.want {
				t.Errorf("MatchKey(%q, %q) = %v, want %v", tt.pattern, tt.requestID, got, tt.want)
			}
		})
	}
}

func TestMatchKey_Wildcard(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		requestID string
		want      bool
	}{
		{"matches child", "admin/*", "admin/list", true},
		{"matches another child", "admin/*", "admin/get", true},
		{"does not match grandchild", "admin/*", "admin/users/list", false},
		{"does not match unrelated", "admin/*", "users/list", false},
		{"root wildcard matches child", "*", "health", true},
		{"does not match prefix itself by glob", "admin/*", "admin", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchKey(tt.pattern, tt.requestID)
			if got != tt.want {
				t.Errorf("MatchKey(%q, %q) = %v, want %v", tt.pattern, tt.requestID, got, tt.want)
			}
		})
	}
}

func TestMatchKey_Glob(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		requestID string
		want      bool
	}{
		{"glob prefix", "admin/l*", "admin/list", true},
		{"glob no match", "admin/l*", "admin/get", false},
		{"glob with question mark", "admin/?et", "admin/get", true},
		{"glob bracket range", "admin/[lg]et", "admin/get", true},
		{"glob bracket range no match", "admin/[lg]et", "admin/set", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchKey(tt.pattern, tt.requestID)
			if got != tt.want {
				t.Errorf("MatchKey(%q, %q) = %v, want %v", tt.pattern, tt.requestID, got, tt.want)
			}
		})
	}
}

func TestFindBestMatch_Priority(t *testing.T) {
	tests := []struct {
		name      string
		patterns  []string
		requestID string
		wantKey   string
		wantFound bool
	}{
		{
			"specific wins over wildcard and glob",
			[]string{"admin/*", "admin/l*", "admin/list"},
			"admin/list",
			"admin/list",
			true,
		},
		{
			"wildcard wins over glob",
			[]string{"admin/l*", "admin/*"},
			"admin/list",
			"admin/*",
			true,
		},
		{
			"glob only",
			[]string{"admin/l*"},
			"admin/list",
			"admin/l*",
			true,
		},
		{
			"no match returns empty",
			[]string{"users/list", "admin/*"},
			"settings/get",
			"",
			false,
		},
		{
			"empty patterns",
			[]string{},
			"admin/list",
			"",
			false,
		},
		{
			"first specific wins",
			[]string{"admin/list", "admin/list"},
			"admin/list",
			"admin/list",
			true,
		},
		{
			"wildcard matches nested prefix",
			[]string{"api/*"},
			"api/users",
			"api/*",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotFound := FindBestMatch(tt.patterns, tt.requestID)
			if gotKey != tt.wantKey || gotFound != tt.wantFound {
				t.Errorf("FindBestMatch(%v, %q) = (%q, %v), want (%q, %v)",
					tt.patterns, tt.requestID, gotKey, gotFound, tt.wantKey, tt.wantFound)
			}
		})
	}
}

func TestClassifyMatch_InvalidGlob(t *testing.T) {
	// Invalid glob pattern should not panic and should not match.
	got := MatchKey("[invalid", "anything")
	if got {
		t.Error("MatchKey with invalid glob pattern should return false")
	}
}
