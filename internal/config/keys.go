package config

// KeyMap maps actions to key binding strings.
// Key strings match bubbletea's tea.KeyMsg.String() output.
// Examples: "ctrl+c", "tab", "enter", "q", " " (space).
// See: https://pkg.go.dev/github.com/charmbracelet/bubbletea#KeyMsg
type KeyMap struct {
	Up          []string
	Down        []string
	PaneLeft    []string
	PaneRight   []string
	TreeNav     []string
	Collapse    []string
	CycleFilter []string
	CycleTime   []string
	Search      []string
	Refresh     []string
	Launch      []string
	Quit        []string
}

// Matches checks if a key string matches any of the provided bindings.
// It normalizes each binding via NormalizeKey() before comparison.
// Special case: if keyStr is " " (space) and any binding is "space", returns true.
func Matches(keyStr string, bindings []string) bool {
	for _, binding := range bindings {
		normalized := NormalizeKey(binding)
		if keyStr == normalized {
			return true
		}
	}
	return false
}

// NormalizeKey converts "space" to " ", all others unchanged.
func NormalizeKey(key string) string {
	if key == "space" {
		return " "
	}
	return key
}

// DefaultKeyMap returns the default key bindings matching current hardcoded values.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:          []string{"k", "up"},
		Down:        []string{"j", "down"},
		PaneLeft:    []string{"h"},
		PaneRight:   []string{"l", "right"},
		TreeNav:     []string{"left"},
		Collapse:    []string{" "},
		CycleFilter: []string{"tab"},
		CycleTime:   []string{"t"},
		Search:      []string{"/"},
		Refresh:     []string{"r"},
		Launch:      []string{"enter"},
		Quit:        []string{"q", "ctrl+c"},
	}
}
