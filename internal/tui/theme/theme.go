// Package theme defines the Theme interface and its implementations for the EKVS TUI.
package theme

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Theme groups all lipgloss styles used by the application.
type Theme interface {
	// Semantic colours
	PrimaryColor() lipgloss.TerminalColor
	SecondaryColor() lipgloss.TerminalColor
	BackgroundColor() lipgloss.TerminalColor
	ErrorColor() lipgloss.TerminalColor

	// Predefined styles
	TitleStyle() lipgloss.Style
	MenuItemStyle() lipgloss.Style
	SelectedMenuItemStyle() lipgloss.Style
	StatusBarStyle() lipgloss.Style
	ErrorStyle() lipgloss.Style

	// UX-polish styles
	SpinnerStyle() lipgloss.Style     // animated loading indicator frame
	FooterStyle() lipgloss.Style      // fixed keyboard-hints bar
	ModalStyle() lipgloss.Style       // blocking error dialog box
	DetailStyle() lipgloss.Style      // profile detail panel label/value text
	TableHeaderStyle() lipgloss.Style // secrets table header row and separator
}

// NewTheme returns the Theme corresponding to the given name.
// Valid names: "adaptive", "hacker".
// Returns an error for unrecognised names.
func NewTheme(name string) (Theme, error) {
	switch name {
	case "adaptive":
		return AdaptiveTheme{}, nil
	case "hacker":
		return HackerTheme{}, nil
	default:
		return nil, fmt.Errorf("unknown theme: %q (valid values: adaptive, hacker)", name)
	}
}

// ---------------------------------------------------------------------------
// AdaptiveTheme
// ---------------------------------------------------------------------------

var (
	adaptivePrimary    = lipgloss.AdaptiveColor{Light: "#1A1A2E", Dark: "#E0E0E0"}
	adaptiveSecondary  = lipgloss.AdaptiveColor{Light: "#5A5A8A", Dark: "#A0A0C0"}
	adaptiveBackground = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1A1A1A"}
	adaptiveError      = lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF5555"}
	adaptiveSelected   = lipgloss.AdaptiveColor{Light: "#0055AA", Dark: "#88C0D0"}
	adaptiveStatus     = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}
)

// AdaptiveTheme adapts to the terminal's light/dark colour scheme.
type AdaptiveTheme struct{}

func (AdaptiveTheme) PrimaryColor() lipgloss.TerminalColor    { return adaptivePrimary }
func (AdaptiveTheme) SecondaryColor() lipgloss.TerminalColor  { return adaptiveSecondary }
func (AdaptiveTheme) BackgroundColor() lipgloss.TerminalColor { return adaptiveBackground }
func (AdaptiveTheme) ErrorColor() lipgloss.TerminalColor      { return adaptiveError }

func (AdaptiveTheme) TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(adaptivePrimary).
		MarginBottom(1)
}

func (AdaptiveTheme) MenuItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(adaptiveSecondary).
		PaddingLeft(4)
}

func (AdaptiveTheme) SelectedMenuItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(adaptiveSelected).
		PaddingLeft(2)
}

func (AdaptiveTheme) StatusBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(adaptiveStatus).
		MarginTop(1)
}

func (AdaptiveTheme) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(adaptiveError)
}

func (AdaptiveTheme) SpinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(adaptiveSecondary)
}

func (AdaptiveTheme) FooterStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(adaptiveStatus).
		MarginTop(1)
}

func (AdaptiveTheme) ModalStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(adaptivePrimary).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(adaptiveError).
		Padding(1, 2)
}

func (AdaptiveTheme) DetailStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(adaptiveSecondary).
		PaddingLeft(2)
}

func (AdaptiveTheme) TableHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(adaptivePrimary)
}

// ---------------------------------------------------------------------------
// HackerTheme — green on black, Matrix-inspired
// ---------------------------------------------------------------------------

const (
	hackerGreen      = lipgloss.Color("#00FF41")
	hackerGreenDim   = lipgloss.Color("#008F11")
	hackerBackground = lipgloss.Color("#0D0208")
	hackerRed        = lipgloss.Color("#FF0000")
)

// HackerTheme is a green-on-black Matrix-inspired theme.
type HackerTheme struct{}

func (HackerTheme) PrimaryColor() lipgloss.TerminalColor    { return hackerGreen }
func (HackerTheme) SecondaryColor() lipgloss.TerminalColor  { return hackerGreenDim }
func (HackerTheme) BackgroundColor() lipgloss.TerminalColor { return hackerBackground }
func (HackerTheme) ErrorColor() lipgloss.TerminalColor      { return hackerRed }

func (HackerTheme) TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(hackerGreen).
		Background(hackerBackground).
		MarginBottom(1)
}

func (HackerTheme) MenuItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(hackerGreenDim).
		Background(hackerBackground).
		PaddingLeft(4)
}

func (HackerTheme) SelectedMenuItemStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(hackerGreen).
		Background(hackerBackground).
		PaddingLeft(2)
}

func (HackerTheme) StatusBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(hackerGreenDim).
		Background(hackerBackground).
		MarginTop(1)
}

func (HackerTheme) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(hackerRed).
		Background(hackerBackground)
}

func (HackerTheme) SpinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(hackerGreen).
		Background(hackerBackground)
}

func (HackerTheme) FooterStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(hackerGreenDim).
		Background(hackerBackground).
		MarginTop(1)
}

func (HackerTheme) ModalStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(hackerGreen).
		Background(hackerBackground).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(hackerRed).
		Padding(1, 2)
}

func (HackerTheme) DetailStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(hackerGreenDim).
		Background(hackerBackground).
		PaddingLeft(2)
}

func (HackerTheme) TableHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(hackerGreen).
		Background(hackerBackground)
}
