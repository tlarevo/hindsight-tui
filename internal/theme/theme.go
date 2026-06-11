package theme

import gloss "charm.land/lipgloss/v2"

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
	}
}
