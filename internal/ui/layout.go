package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	gloss "charm.land/lipgloss/v2"
)

func Clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func Panel(title, body string, width int) string {
	style := gloss.NewStyle().Border(gloss.RoundedBorder()).Padding(0, 1)
	if width > 0 {
		style = style.Width(width)
	}
	if title != "" {
		body = title + "\n" + body
	}
	return style.Render(body)
}

func TwoColumn(left, right string, totalWidth int) string {
	if totalWidth <= 0 {
		return left + "\n" + right
	}
	leftWidth := totalWidth / 2
	rightWidth := totalWidth - leftWidth
	leftStyle := gloss.NewStyle().Width(leftWidth)
	rightStyle := gloss.NewStyle().Width(rightWidth)
	return gloss.JoinHorizontal(gloss.Top, leftStyle.Render(left), rightStyle.Render(right))
}

func PrettyJSON(value any) string {
	if value == nil {
		return "{}"
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("<unrenderable: %v>", err)
	}
	return string(data)
}

func Lines(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, "\n")
}

func TruncateRunes(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if gloss.Width(value) <= width {
		return value
	}
	runes := []rune(value)
	if width <= 1 {
		return string(runes[:1])
	}
	return string(runes[:width-1]) + "…"
}
