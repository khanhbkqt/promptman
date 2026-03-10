package variable

// MergeScopes merges multiple variable maps into one.
// Later scopes override earlier scopes.
//
//	merged := MergeScopes(envVars, collectionDefaults, requestOverrides)
//	// requestOverrides wins > collectionDefaults > envVars
func MergeScopes(scopes ...map[string]any) map[string]any {
	result := make(map[string]any)
	for _, scope := range scopes {
		for k, v := range scope {
			result[k] = v
		}
	}
	return result
}
