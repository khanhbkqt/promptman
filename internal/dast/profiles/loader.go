package profiles

import (
	"sort"
	"sync"

	"github.com/khanhnguyen/promptman/internal/dast"
)

// Profile represents a named collection of DAST rules.
type Profile struct {
	Name        string      // profile identifier: quick, standard, thorough
	Description string      // human-readable description
	Rules       []dast.Rule // ordered list of rules in this profile
}

var (
	mu       sync.RWMutex
	registry = make(map[string]*Profile)
)

// Register adds a profile to the global registry.
// It is typically called from init() functions in profile definition files.
func Register(p *Profile) {
	mu.Lock()
	defer mu.Unlock()
	registry[p.Name] = p
}

// Get retrieves a profile by name. Returns nil if not found.
func Get(name string) *Profile {
	mu.RLock()
	defer mu.RUnlock()
	return registry[name]
}

// ListProfiles returns metadata for all registered profiles,
// sorted alphabetically by name.
func ListProfiles() []dast.ProfileInfo {
	mu.RLock()
	defer mu.RUnlock()

	infos := make([]dast.ProfileInfo, 0, len(registry))
	for _, p := range registry {
		infos = append(infos, dast.ProfileInfo{
			Name:        p.Name,
			Description: p.Description,
			RuleCount:   len(p.Rules),
		})
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	return infos
}

// Names returns all registered profile names sorted alphabetically.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
