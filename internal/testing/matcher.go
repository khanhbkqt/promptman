package testing

import (
	"path/filepath"
	"strings"
)

// matchType classifies how a pattern matches a request ID.
type matchType int

const (
	matchNone     matchType = iota // no match
	matchGlob                      // glob pattern (e.g., "admin/l*")
	matchWildcard                  // trailing /* wildcard (e.g., "admin/*")
	matchSpecific                  // exact string match
)

// MatchKey reports whether pattern matches the given requestID.
//
// Three pattern types are supported (highest to lowest priority):
//   - Specific: exact string equality ("users/list" matches "users/list")
//   - Wildcard: trailing /* matches all under a prefix ("admin/*" matches "admin/list")
//   - Glob: filepath.Match-style glob patterns ("admin/l*" matches "admin/list")
func MatchKey(pattern, requestID string) bool {
	return classifyMatch(pattern, requestID) != matchNone
}

// FindBestMatch returns the best-matching pattern for a requestID from
// the given set of patterns. Priority order: specific > wildcard > glob.
// If no pattern matches, it returns ("", false).
func FindBestMatch(patterns []string, requestID string) (string, bool) {
	bestType := matchNone
	bestPattern := ""

	for _, p := range patterns {
		mt := classifyMatch(p, requestID)
		if mt > bestType {
			bestType = mt
			bestPattern = p
		}
		// If we found a specific match, it's the highest priority — stop early.
		if bestType == matchSpecific {
			return bestPattern, true
		}
	}

	if bestType == matchNone {
		return "", false
	}
	return bestPattern, true
}

// classifyMatch determines how pattern matches requestID.
func classifyMatch(pattern, requestID string) matchType {
	// Specific: exact string match (highest priority).
	if pattern == requestID {
		return matchSpecific
	}

	// Wildcard: trailing /* matches direct children under the prefix.
	// "admin/*" matches "admin/list" but NOT "admin/users/list" (grandchild)
	// and NOT "admin" itself (the prefix).
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		if strings.HasPrefix(requestID, prefix+"/") {
			// Ensure it's a direct child: no further "/" after the prefix.
			remainder := requestID[len(prefix)+1:]
			if !strings.Contains(remainder, "/") {
				return matchWildcard
			}
		}
		// Wildcard patterns are handled entirely here — don't fall through
		// to glob, which would match differently (e.g., grandchildren).
		return matchNone
	}

	// Glob: filepath.Match for general glob patterns.
	matched, err := filepath.Match(pattern, requestID)
	if err == nil && matched {
		return matchGlob
	}

	return matchNone
}
