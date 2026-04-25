package main

import (
	"flag"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
	tuiconfig "ekvs/internal/tui/config"
	"ekvs/internal/tui/root"
	"ekvs/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	const defaultConfig = "ekvs-tui.yaml"
	configPath := flag.String("config", defaultConfig, "path to the YAML configuration file")
	flag.Parse()

	explicit := *configPath != defaultConfig

	cf, err := tuiconfig.LoadFromFile(*configPath, explicit)
	if err != nil {
		log.Fatalf("error loading configuration: %v", err)
	}

	defaultTheme, _ := theme.NewTheme("adaptive")

	// Pre-warm lipgloss's dark-background detection while stdin is still in
	// cooked mode. lipgloss v1 (via termenv) queries the terminal with OSC 11
	// and caches the result; if this is called for the first time inside
	// View() — after bubbletea has taken over stdin via epoll — termenv can
	// no longer read the response and blocks until its timeout (~5-6 s).
	_ = lipgloss.HasDarkBackground()

	m := root.New(cf, defaultTheme)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Printf("TUI error: %v", err)
		os.Exit(1)
	}
}
