package main

import (
	"fmt"
	"log"

	"l2/config"
	"l2/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func exitStats(m *ui.Model) string {
	style := lipgloss.NewStyle().Border(lipgloss.ThickBorder()).Padding(1)
	header := lipgloss.NewStyle().Bold(true).Render("Session stats:")
	stats := m.GetStats()
	return style.Render(fmt.Sprintf("%s\nTotal tokens used: %d\n", header, stats.TotalTokens))
}

func main() {
	client := config.NewLLMClient()

	m := ui.NewModel()
	m.SetLLM(client)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
	fmt.Print(exitStats(m) + "\n\n")
}
