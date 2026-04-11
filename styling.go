package main

import "charm.land/lipgloss/v2"

type styles struct {
	hintStyle lipgloss.Style
}

func newStyles(isDark bool) (s *styles) {
	s = new(styles)
	lightDark := lipgloss.LightDark(isDark)
	s.hintStyle = lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("#c6c6c6"), lipgloss.Color("#686868")))
	return s
}
