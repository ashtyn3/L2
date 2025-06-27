package ui

import (
	"log"
	"time"

	"l2/storage"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/schema"
)

var (
	border = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
)

// NewModel creates a new UI model with initialized components
func NewModel() *Model {
	exists, err := storage.CheckFile(storage.ConversationFile)
	if err != nil {
		log.Fatal(err)
	}
	var history []*schema.Message
	if exists {
		history, err = storage.ReadConversation()
		if err != nil {
			history = []*schema.Message{}
		}
	}

	ti := textarea.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.ShowLineNumbers = false
	ti.SetHeight(1)
	ti.MaxHeight = 1 // Ensure it stays at 1 line
	ti.FocusedStyle.Base = border
	ti.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.NoColor{})
	ti.Prompt = ""

	stats, err := storage.ReadStats()
	if err != nil {
		stats = storage.Stats{TotalTokens: 0}
	}

	return &Model{
		ta:        ti,
		ready:     false,
		tokenChan: make(chan string, 100),
		history:   history,
		stats:     stats,

		// Initialize optimization fields for long responses
		maxHistoryDisplay: 10,                     // Show last 10 messages
		renderBuffer:      5,                      // 5 line buffer for smooth scrolling
		renderThrottle:    100 * time.Millisecond, // Throttle renders to 100ms
	}
}
