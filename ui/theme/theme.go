package theme

import "github.com/charmbracelet/lipgloss"

// Colors defines all the colors used in the application
type Colors struct {
	Background    lipgloss.Color
	Foreground    lipgloss.Color
	ForegroundDim lipgloss.Color

	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color

	BorderFocused   lipgloss.Color
	BorderUnfocused lipgloss.Color

	SelectionBg lipgloss.Color
	SelectionFg lipgloss.Color

	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color
}

type Theme struct {
	Name   string
	Colors Colors

	Header          lipgloss.Style
	Footer          lipgloss.Style
	Title           lipgloss.Style
	BorderFocused   lipgloss.Style
	BorderUnfocused lipgloss.Style

	TableHeader   lipgloss.Style
	TableCell     lipgloss.Style
	TableSelected lipgloss.Style
	TableBorder   lipgloss.Style

	SidebarTitle    lipgloss.Style
	SidebarItem     lipgloss.Style
	SidebarSelected lipgloss.Style
	SidebarActive   lipgloss.Style

	StatusBar lipgloss.Style
}

// Current holds the active theme
var Current *Theme

func init() {
	Current = DefaultTheme()
}

// SetTheme changes the current theme
func SetTheme(t *Theme) {
	Current = t
}

// GetTheme returns the current theme
func GetTheme() *Theme {
	return Current
}

// buildStyles creates all the pre-built styles from colors
func buildStyles(name string, c Colors) *Theme {
	t := &Theme{
		Name:   name,
		Colors: c,
	}

	t.Header = lipgloss.NewStyle().
		Foreground(c.Foreground).
		Background(c.Primary).
		Bold(true).
		Padding(0, 2)

	t.Footer = lipgloss.NewStyle().
		Foreground(c.Foreground).
		Background(c.Primary).
		Padding(0, 2)

	t.Title = lipgloss.NewStyle().
		Foreground(c.Foreground).
		Background(c.Primary).
		Bold(true)

	t.BorderFocused = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.BorderFocused)

	t.BorderUnfocused = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.BorderUnfocused)

	t.TableHeader = lipgloss.NewStyle().
		Foreground(c.Foreground).
		Background(c.Primary).
		Bold(true)

	t.TableCell = lipgloss.NewStyle().
		Foreground(c.Foreground)

	t.TableSelected = lipgloss.NewStyle().
		Foreground(c.SelectionFg).
		Background(c.SelectionBg)

	t.TableBorder = lipgloss.NewStyle().
		Foreground(c.BorderUnfocused)

	t.SidebarTitle = lipgloss.NewStyle().
		Foreground(c.Foreground).
		Background(c.Primary).
		Bold(true)

	t.SidebarItem = lipgloss.NewStyle().
		Foreground(c.ForegroundDim)

	t.SidebarSelected = lipgloss.NewStyle().
		Foreground(c.SelectionFg).
		Background(c.SelectionBg)

	t.SidebarActive = lipgloss.NewStyle().
		Foreground(c.Primary)

	// Status bar
	t.StatusBar = lipgloss.NewStyle().
		Foreground(c.ForegroundDim)

	return t
}

// DefaultTheme returns the default purple theme
func DefaultTheme() *Theme {
	return buildStyles("default", Colors{
		Background:      lipgloss.Color("#1a1a2e"),
		Foreground:      lipgloss.Color("#FAFAFA"),
		ForegroundDim:   lipgloss.Color("#888888"),
		Primary:         lipgloss.Color("#7D56F4"),
		Secondary:       lipgloss.Color("#5A4FCF"),
		Accent:          lipgloss.Color("#9D7BFF"),
		BorderFocused:   lipgloss.Color("#7D56F4"),
		BorderUnfocused: lipgloss.Color("#3C3C3C"),
		SelectionBg:     lipgloss.Color("#5A4FCF"),
		SelectionFg:     lipgloss.Color("#FAFAFA"),
		Success:         lipgloss.Color("#50FA7B"),
		Warning:         lipgloss.Color("#FFB86C"),
		Error:           lipgloss.Color("#FF5555"),
		Info:            lipgloss.Color("#8BE9FD"),
	})
}

// DraculaTheme returns the Dracula color theme
func DraculaTheme() *Theme {
	return buildStyles("dracula", Colors{
		Background:      lipgloss.Color("#282a36"),
		Foreground:      lipgloss.Color("#f8f8f2"),
		ForegroundDim:   lipgloss.Color("#6272a4"),
		Primary:         lipgloss.Color("#bd93f9"),
		Secondary:       lipgloss.Color("#ff79c6"),
		Accent:          lipgloss.Color("#8be9fd"),
		BorderFocused:   lipgloss.Color("#bd93f9"),
		BorderUnfocused: lipgloss.Color("#44475a"),
		SelectionBg:     lipgloss.Color("#44475a"),
		SelectionFg:     lipgloss.Color("#f8f8f2"),
		Success:         lipgloss.Color("#50fa7b"),
		Warning:         lipgloss.Color("#ffb86c"),
		Error:           lipgloss.Color("#ff5555"),
		Info:            lipgloss.Color("#8be9fd"),
	})
}

// NordTheme returns the Nord color theme
func NordTheme() *Theme {
	return buildStyles("nord", Colors{
		Background:      lipgloss.Color("#2e3440"),
		Foreground:      lipgloss.Color("#eceff4"),
		ForegroundDim:   lipgloss.Color("#4c566a"),
		Primary:         lipgloss.Color("#5e81ac"),
		Secondary:       lipgloss.Color("#81a1c1"),
		Accent:          lipgloss.Color("#88c0d0"),
		BorderFocused:   lipgloss.Color("#88c0d0"),
		BorderUnfocused: lipgloss.Color("#3b4252"),
		SelectionBg:     lipgloss.Color("#434c5e"),
		SelectionFg:     lipgloss.Color("#eceff4"),
		Success:         lipgloss.Color("#a3be8c"),
		Warning:         lipgloss.Color("#ebcb8b"),
		Error:           lipgloss.Color("#bf616a"),
		Info:            lipgloss.Color("#81a1c1"),
	})
}

// GruvboxTheme returns the Gruvbox dark theme
func GruvboxTheme() *Theme {
	return buildStyles("gruvbox", Colors{
		Background:      lipgloss.Color("#282828"),
		Foreground:      lipgloss.Color("#ebdbb2"),
		ForegroundDim:   lipgloss.Color("#928374"),
		Primary:         lipgloss.Color("#fe8019"),
		Secondary:       lipgloss.Color("#fabd2f"),
		Accent:          lipgloss.Color("#8ec07c"),
		BorderFocused:   lipgloss.Color("#fe8019"),
		BorderUnfocused: lipgloss.Color("#3c3836"),
		SelectionBg:     lipgloss.Color("#504945"),
		SelectionFg:     lipgloss.Color("#ebdbb2"),
		Success:         lipgloss.Color("#b8bb26"),
		Warning:         lipgloss.Color("#fabd2f"),
		Error:           lipgloss.Color("#fb4934"),
		Info:            lipgloss.Color("#83a598"),
	})
}

// TokyoNightTheme returns the Tokyo Night theme
func TokyoNightTheme() *Theme {
	return buildStyles("tokyo-night", Colors{
		Background:      lipgloss.Color("#1a1b26"),
		Foreground:      lipgloss.Color("#c0caf5"),
		ForegroundDim:   lipgloss.Color("#565f89"),
		Primary:         lipgloss.Color("#7aa2f7"),
		Secondary:       lipgloss.Color("#bb9af7"),
		Accent:          lipgloss.Color("#7dcfff"),
		BorderFocused:   lipgloss.Color("#7aa2f7"),
		BorderUnfocused: lipgloss.Color("#3b4261"),
		SelectionBg:     lipgloss.Color("#33467c"),
		SelectionFg:     lipgloss.Color("#c0caf5"),
		Success:         lipgloss.Color("#9ece6a"),
		Warning:         lipgloss.Color("#e0af68"),
		Error:           lipgloss.Color("#f7768e"),
		Info:            lipgloss.Color("#7dcfff"),
	})
}

// CatppuccinTheme returns the Catppuccin Mocha theme
func CatppuccinTheme() *Theme {
	return buildStyles("catppuccin", Colors{
		Background:      lipgloss.Color("#1e1e2e"),
		Foreground:      lipgloss.Color("#cdd6f4"),
		ForegroundDim:   lipgloss.Color("#6c7086"),
		Primary:         lipgloss.Color("#cba6f7"),
		Secondary:       lipgloss.Color("#f5c2e7"),
		Accent:          lipgloss.Color("#94e2d5"),
		BorderFocused:   lipgloss.Color("#cba6f7"),
		BorderUnfocused: lipgloss.Color("#313244"),
		SelectionBg:     lipgloss.Color("#45475a"),
		SelectionFg:     lipgloss.Color("#cdd6f4"),
		Success:         lipgloss.Color("#a6e3a1"),
		Warning:         lipgloss.Color("#f9e2af"),
		Error:           lipgloss.Color("#f38ba8"),
		Info:            lipgloss.Color("#89dceb"),
	})
}

// MonokaiTheme returns the Monokai theme
func MonokaiTheme() *Theme {
	return buildStyles("monokai", Colors{
		Background:      lipgloss.Color("#272822"),
		Foreground:      lipgloss.Color("#f8f8f2"),
		ForegroundDim:   lipgloss.Color("#75715e"),
		Primary:         lipgloss.Color("#f92672"),
		Secondary:       lipgloss.Color("#ae81ff"),
		Accent:          lipgloss.Color("#a6e22e"),
		BorderFocused:   lipgloss.Color("#f92672"),
		BorderUnfocused: lipgloss.Color("#49483e"),
		SelectionBg:     lipgloss.Color("#49483e"),
		SelectionFg:     lipgloss.Color("#f8f8f2"),
		Success:         lipgloss.Color("#a6e22e"),
		Warning:         lipgloss.Color("#e6db74"),
		Error:           lipgloss.Color("#f92672"),
		Info:            lipgloss.Color("#66d9ef"),
	})
}

// GetAvailableThemes returns a list of all available theme names
func GetAvailableThemes() []string {
	return []string{
		"default",
		"dracula",
		"nord",
		"gruvbox",
		"tokyo-night",
		"catppuccin",
		"monokai",
	}
}

// GetThemeByName returns a theme by its name
func GetThemeByName(name string) *Theme {
	switch name {
	case "dracula":
		return DraculaTheme()
	case "nord":
		return NordTheme()
	case "gruvbox":
		return GruvboxTheme()
	case "tokyo-night":
		return TokyoNightTheme()
	case "catppuccin":
		return CatppuccinTheme()
	case "monokai":
		return MonokaiTheme()
	default:
		return DefaultTheme()
	}
}
