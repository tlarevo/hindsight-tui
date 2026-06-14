package theme

import (
	"fmt"
	"image/color"
	"strings"

	gloss "charm.land/lipgloss/v2"
)

// Palette holds every semantic style the TUI needs.
type Palette struct {
	Header          gloss.Style
	Border          gloss.Style
	SidebarSelected gloss.Style
	Success         gloss.Style
	Warning         gloss.Style
	Error           gloss.Style
	Muted           gloss.Style
	TableHeader     gloss.Style
	FormLabel       gloss.Style

	// Extended semantic styles
	Primary       gloss.Style // Main accent color for titles and highlights
	Accent        gloss.Style // Secondary accent for subtle emphasis
	PanelTitle    gloss.Style // Styled panel title text
	FocusedLabel  gloss.Style // Label style when focused
	TabActive     gloss.Style // Active tab indicator
	TabInactive   gloss.Style // Inactive tab indicator
	StatusGood    gloss.Style // Healthy/running status
	StatusBad     gloss.Style // Degraded status
	StatusNeutral gloss.Style // Neutral status text
	Footer        gloss.Style // Footer bar styling
	Spinner       gloss.Style // Loading spinner
	Selection     gloss.Style // Selected item highlight
	Code          gloss.Style // Code/monospace text
	HeaderKey     gloss.Style // Header key label
	HeaderValue   gloss.Style // Header value

	// Raw colors for borders where Style getters aren't available
	PanelBorderColor   color.Color
	FocusedBorderColor color.Color
}

// Panel renders a titled, bordered panel. Pass width=0 to auto-size.
func (p Palette) Panel(title, body string, width int) string {
	style := gloss.NewStyle().
		Border(gloss.RoundedBorder()).
		Padding(0, 1).
		BorderForeground(p.PanelBorderColor)

	if width > 0 {
		style = style.Width(width)
	}

	var content string
	if title != "" {
		titleStyled := p.PanelTitle.Render(title)
		sep := p.Separator(max(0, width-4))
		content = strings.Join([]string{titleStyled, sep, body}, "\n")
	} else {
		content = body
	}

	return style.Render(content)
}

// FocusedPanel renders a panel with an accent border to indicate focus.
func (p Palette) FocusedPanel(title, body string, width int) string {
	style := gloss.NewStyle().
		Border(gloss.RoundedBorder()).
		Padding(0, 1).
		BorderForeground(p.FocusedBorderColor)

	if width > 0 {
		style = style.Width(width)
	}

	var content string
	if title != "" {
		titleStyled := p.FocusedLabel.Render(title)
		sep := p.Separator(max(0, width-4))
		content = strings.Join([]string{titleStyled, sep, body}, "\n")
	} else {
		content = body
	}

	return style.Render(content)
}

// Separator returns a horizontal divider line of the given width.
func (p Palette) Separator(width int) string {
	if width <= 0 {
		return ""
	}
	return p.Muted.Render(strings.Repeat("─", width))
}

// StatusLabel renders a key: value pair with semantic coloring.
// kind is "good", "bad", "neutral", or "" (default muted).
func (p Palette) StatusLabel(key, value, kind string) string {
	keyStyled := p.HeaderKey.Render(key + ": ")
	var valueStyled string
	switch kind {
	case "good":
		valueStyled = p.StatusGood.Render(value)
	case "bad":
		valueStyled = p.StatusBad.Render(value)
	case "neutral":
		valueStyled = p.StatusNeutral.Render(value)
	default:
		valueStyled = p.Muted.Render(value)
	}
	return keyStyled + valueStyled
}

// StatusLabelWithIcon renders ● + key: value with semantic coloring.
func (p Palette) StatusLabelWithIcon(key, value, kind string) string {
	var icon string
	switch kind {
	case "good":
		icon = p.StatusGood.Render("●")
	case "bad":
		icon = p.StatusBad.Render("●")
	case "neutral":
		icon = p.StatusNeutral.Render("◐")
	default:
		icon = p.Muted.Render("○")
	}
	return fmt.Sprintf("%s %s", icon, p.StatusLabel(key, value, kind))
}

func Resolve(name string) Palette {
	switch name {
	case "dark":
		return darkPalette()
	case "light":
		return lightPalette()
	case "auto":
		fallthrough
	default:
		return darkPalette()
	}
}

func darkPalette() Palette {
	return Palette{
		Header:          gloss.NewStyle().Bold(true).Foreground(gloss.Color("86")).Padding(0, 1),
		Border:          gloss.NewStyle().BorderForeground(gloss.Color("63")),
		SidebarSelected: gloss.NewStyle().Bold(true).Foreground(gloss.Color("230")).Background(gloss.Color("63")).Padding(0, 1),
		Success:         gloss.NewStyle().Foreground(gloss.Color("42")).Bold(true),
		Warning:         gloss.NewStyle().Foreground(gloss.Color("214")).Bold(true),
		Error:           gloss.NewStyle().Foreground(gloss.Color("203")).Bold(true),
		Muted:           gloss.NewStyle().Foreground(gloss.Color("244")),
		TableHeader:     gloss.NewStyle().Bold(true).Foreground(gloss.Color("117")),
		FormLabel:       gloss.NewStyle().Bold(true).Foreground(gloss.Color("153")),

		Primary:       gloss.NewStyle().Foreground(gloss.Color("111")).Bold(true),
		Accent:        gloss.NewStyle().Foreground(gloss.Color("141")),
		PanelTitle:    gloss.NewStyle().Bold(true).Foreground(gloss.Color("111")),
		FocusedLabel:  gloss.NewStyle().Bold(true).Foreground(gloss.Color("141")),
		TabActive:     gloss.NewStyle().Bold(true).Foreground(gloss.Color("230")).Background(gloss.Color("63")).Padding(0, 1),
		TabInactive:   gloss.NewStyle().Foreground(gloss.Color("244")).Padding(0, 1),
		StatusGood:    gloss.NewStyle().Foreground(gloss.Color("42")).Bold(true),
		StatusBad:     gloss.NewStyle().Foreground(gloss.Color("203")).Bold(true),
		StatusNeutral: gloss.NewStyle().Foreground(gloss.Color("214")).Bold(true),
		Footer:        gloss.NewStyle().Foreground(gloss.Color("244")).Padding(0, 1),
		Spinner:       gloss.NewStyle().Foreground(gloss.Color("141")),
		Selection:     gloss.NewStyle().Foreground(gloss.Color("230")).Background(gloss.Color("23")).Padding(0, 1),
		Code:          gloss.NewStyle().Foreground(gloss.Color("150")).Background(gloss.Color("235")).Padding(0, 1),
		HeaderKey:     gloss.NewStyle().Foreground(gloss.Color("244")),
		HeaderValue:   gloss.NewStyle().Foreground(gloss.Color("111")).Bold(true),

		PanelBorderColor:   gloss.Color("60"),
		FocusedBorderColor: gloss.Color("141"),
	}
}

func lightPalette() Palette {
	return Palette{
		Header:          gloss.NewStyle().Bold(true).Foreground(gloss.Color("25")).Padding(0, 1),
		Border:          gloss.NewStyle().BorderForeground(gloss.Color("69")),
		SidebarSelected: gloss.NewStyle().Bold(true).Foreground(gloss.Color("255")).Background(gloss.Color("33")).Padding(0, 1),
		Success:         gloss.NewStyle().Foreground(gloss.Color("28")).Bold(true),
		Warning:         gloss.NewStyle().Foreground(gloss.Color("166")).Bold(true),
		Error:           gloss.NewStyle().Foreground(gloss.Color("160")).Bold(true),
		Muted:           gloss.NewStyle().Foreground(gloss.Color("245")),
		TableHeader:     gloss.NewStyle().Bold(true).Foreground(gloss.Color("25")),
		FormLabel:       gloss.NewStyle().Bold(true).Foreground(gloss.Color("60")),

		Primary:       gloss.NewStyle().Foreground(gloss.Color("27")).Bold(true),
		Accent:        gloss.NewStyle().Foreground(gloss.Color("99")),
		PanelTitle:    gloss.NewStyle().Bold(true).Foreground(gloss.Color("27")),
		FocusedLabel:  gloss.NewStyle().Bold(true).Foreground(gloss.Color("99")),
		TabActive:     gloss.NewStyle().Bold(true).Foreground(gloss.Color("255")).Background(gloss.Color("27")).Padding(0, 1),
		TabInactive:   gloss.NewStyle().Foreground(gloss.Color("245")).Padding(0, 1),
		StatusGood:    gloss.NewStyle().Foreground(gloss.Color("28")).Bold(true),
		StatusBad:     gloss.NewStyle().Foreground(gloss.Color("160")).Bold(true),
		StatusNeutral: gloss.NewStyle().Foreground(gloss.Color("166")).Bold(true),
		Footer:        gloss.NewStyle().Foreground(gloss.Color("245")).Padding(0, 1),
		Spinner:       gloss.NewStyle().Foreground(gloss.Color("99")),
		Selection:     gloss.NewStyle().Foreground(gloss.Color("255")).Background(gloss.Color("25")).Padding(0, 1),
		Code:          gloss.NewStyle().Foreground(gloss.Color("25")).Background(gloss.Color("254")).Padding(0, 1),
		HeaderKey:     gloss.NewStyle().Foreground(gloss.Color("245")),
		HeaderValue:   gloss.NewStyle().Foreground(gloss.Color("27")).Bold(true),

		PanelBorderColor:   gloss.Color("111"),
		FocusedBorderColor: gloss.Color("99"),
	}
}
