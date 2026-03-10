package variable

import "strings"

// token represents a parsed segment of a template string.
type token struct {
	isVar bool   // true if this is a {{variable}} reference
	value string // variable name (if isVar) or literal text
}

// parse tokenizes a template string into literal and variable tokens.
// It handles the escape sequence \{\{ ... \}\} to produce literal {{ }}.
func parse(template string) []token {
	var tokens []token
	i := 0
	n := len(template)

	for i < n {
		// Check for escaped opening: \{\{
		if i+3 < n && template[i:i+4] == `\{\{` {
			// Find matching escaped closing: \}\}
			closeIdx := strings.Index(template[i+4:], `\}\}`)
			if closeIdx >= 0 {
				// Produce literal {{ content }}
				inner := template[i+4 : i+4+closeIdx]
				tokens = append(tokens, token{isVar: false, value: "{{" + inner + "}}"})
				i = i + 4 + closeIdx + 4 // skip past \}\}
				continue
			}
			// No matching close — treat as literal
			tokens = append(tokens, token{isVar: false, value: template[i : i+4]})
			i += 4
			continue
		}

		// Check for variable opening: {{
		if i+1 < n && template[i] == '{' && template[i+1] == '{' {
			// Find closing }}
			closeIdx := strings.Index(template[i+2:], "}}")
			if closeIdx >= 0 {
				varName := strings.TrimSpace(template[i+2 : i+2+closeIdx])
				if varName != "" {
					tokens = append(tokens, token{isVar: true, value: varName})
				} else {
					// Empty variable name — treat as literal
					tokens = append(tokens, token{isVar: false, value: "{{}}"})
				}
				i = i + 2 + closeIdx + 2
				continue
			}
			// No closing }} — treat as literal
			tokens = append(tokens, token{isVar: false, value: "{{"})
			i += 2
			continue
		}

		// Collect literal characters
		start := i
		for i < n {
			if i+1 < n && template[i] == '{' && template[i+1] == '{' {
				break
			}
			if i+3 < n && template[i:i+4] == `\{\{` {
				break
			}
			i++
		}
		tokens = append(tokens, token{isVar: false, value: template[start:i]})
	}

	return tokens
}

// containsVariable checks if a string contains any {{variable}} pattern.
func containsVariable(s string) bool {
	idx := strings.Index(s, "{{")
	if idx < 0 {
		return false
	}
	return strings.Index(s[idx+2:], "}}") >= 0
}
