package ui

import "github.com/charmbracelet/lipgloss"

// Terminal color palette
const (
	ColorDeepBlue  = "#1a3a5c"
	ColorGold      = "#E5C07B"
	ColorLightBlue = "#7AA2F7"
	ColorGray      = "#888888"
	ColorRed       = "#F7768E"
)

// PDF color palette (RGB values)
var (
	PDFDeepBlue = [3]int{26, 58, 92}
	PDFLightGray = [3]int{245, 245, 245}
	PDFMedGray   = [3]int{180, 180, 180}
	PDFBlack     = [3]int{0, 0, 0}
	PDFDarkGray  = [3]int{100, 100, 100}
)

// Terminal output styles
var (
	StyleSuccess    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorLightBlue))
	StyleWarning    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorGold))
	StyleError      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorRed))
	StylePassphrase = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorGold)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorGold)).
			Padding(0, 2)
	StyleSecondary = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
	StyleLabel     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorDeepBlue))
)
