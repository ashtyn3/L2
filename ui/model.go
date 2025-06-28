package ui

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"l2/storage"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/common-nighthawk/go-figure"
)

var ascii = figure.NewFigure("L2", "banner4", true).Slicify()

// Model represents the main UI model
type Model struct {
	ta              textarea.Model
	hold            viewport.Model
	height          int
	width           int
	ready           bool
	llm             compose.Runnable[[]*schema.Message, []*schema.Message]
	history         []*schema.Message
	streaming       bool
	currentResponse strings.Builder
	tokenChan       chan string
	glam            *glamour.TermRenderer
	stats           storage.Stats
	quit            bool
	thinking        bool

	// Optimization fields for long responses
	maxHistoryDisplay int           // Maximum number of history messages to display
	maxResponseLength int           // Maximum length of current response to display
	renderBuffer      int           // Buffer size for rendering (extra lines above/below viewport)
	lastRenderTime    time.Time     // Track last render time to throttle updates
	renderThrottle    time.Duration // Minimum time between renders
}

// Custom message types for streaming
type streamStartMsg struct{}
type exitMsg struct{}
type tickMsg struct{}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink)
}

// tick returns a command that sends a tick message
func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// AddToHistory adds a message to the conversation history
func (m *Model) AddToHistory(msg *schema.Message) {
	m.history = append(m.history, msg)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		viewportWidth := msg.Width - 2
		viewportHeight := msg.Height - (len(ascii) + 3)

		if viewportWidth < 1 {
			viewportWidth = 1
		}
		if viewportHeight < 1 {
			viewportHeight = 1
		}

		vp := viewport.New(viewportWidth, viewportHeight)
		vp.Style = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).Padding(1)

		m.hold = vp
		m.height = msg.Height
		m.width = msg.Width
		m.ready = true

		glam, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithEmoji(),
			glamour.WithWordWrap(viewportWidth-4),
		)
		if err != nil {
			log.Fatal(err)
		}
		m.glam = glam

		m.updateViewportContent()

	case tickMsg:
		if m.streaming {
			select {
			case token, ok := <-m.tokenChan:
				if !ok {
					m.streaming = false
					m.AddToHistory(schema.AssistantMessage(m.currentResponse.String(), nil))
					m.resetOptimizationParams() // Reset to default values
					// Force a viewport refresh by bypassing throttling
					m.lastRenderTime = time.Time{} // Reset to force immediate update
					m.updateViewportContentInternal()
					storage.WriteConversation(m.history)
					// Add a small delay to ensure UI processes the state change
					return m, tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
						return nil
					})
				}
				m.currentResponse.WriteString(token)
				m.adjustOptimizationParams() // Adjust parameters based on response length
				m.cleanupLongResponse()      // Clean up if response gets too long
				m.updateViewportContent()
				return m, tick()
			default:
				return m, tick()
			}
		}
		return m, nil

	case exitMsg:
		return m, tea.Sequence(tea.ExitAltScreen, tea.Quit)

	case streamStartMsg:
		// Start the ticker for streaming
		return m, tick()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.ta.Focused() {
				m.ta.Blur()
			}
		case tea.KeyEnter:
			if m.streaming {
				return m, nil // Don't allow new input while streaming
			}

			userMessage := m.ta.Value()
			if userMessage == "" {
				return m, nil
			}

			// Add user message to history
			m.AddToHistory(schema.UserMessage(userMessage))

			// Update viewport to show the new message
			m.updateViewportContent()

			// Start streaming response
			m.streaming = true
			m.currentResponse.Reset()
			m.tokenChan = make(chan string, 100) // Buffer for tokens

			// Start streaming in background with the user message
			cmds = append(cmds, m.startStreaming(userMessage))

			m.ta.SetValue("")
			return m, tea.Batch(cmds...)
		case tea.KeyCtrlC:
			storage.WriteConversation(m.history)
			storage.WriteStats(m.stats)
			return m, tea.Sequence(m.Exit())

		default:
			if !m.ta.Focused() {
				cmd = m.ta.Focus()
				cmds = append(cmds, cmd)
			}
		}
	}

	// Update both textarea and viewport
	m.ta, cmd = m.ta.Update(msg)
	cmds = append(cmds, cmd)

	if m.ready {
		m.hold, cmd = m.hold.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
func (m *Model) FinalMsg() tea.Cmd {
	return func() tea.Msg {
		return tea.Quit()
	}
}
func (m *Model) Exit() tea.Cmd {
	return func() tea.Msg {
		// Print any final output to console here
		m.quit = true
		return exitMsg{}
	}
}

func (m *Model) createCondensedHistory() []*schema.Message {
	userMessages := make([]*schema.Message, 0)
	for _, msg := range m.history {
		if msg.Role == "user" || msg.Role == "assistant" {
			userMessages = append(userMessages, msg)
		}
	}

	var contextMessage string
	if len(userMessages) > 10 {
		contextMessage = "CONTEXT: " + m.generateContextSummary(userMessages[:len(userMessages)-1])
	} else if len(userMessages) > 0 {
		contextMessage = "CONTEXT: " + m.formatExistingContext(userMessages)
	} else {
		contextMessage = "CONTEXT: No previous conversation"
	}

	structuredMessage := schema.SystemMessage(contextMessage)

	return []*schema.Message{structuredMessage}
}

func (m *Model) generateContextSummary(messages []*schema.Message) string {
	summaryPrompt := `Please provide a detailed summary of the conlang conversation so far, focusing on:

**CRITICAL INFORMATION TO INCLUDE:**
- **Vocabulary and word definitions** that were discussed or created
- **Phonology rules and sound systems** that were established
- **Grammar rules and structures** that were defined
- **Writing systems or orthography** that were developed
- **Example sentences or translations** that were provided
- **Specific conlang features** (cases, tenses, aspects, etc.)
- **Cultural or naming conventions** that were established
- **Any specific words, roots, or morphemes** that were created
- **Tool call results and data operations** that were performed:
  - Lexicon entries that were added via add_lexicon_entry tool
  - Lexicon data that was retrieved via get_lexicon tool
  - Files that were created or read via file tools
  - Phonology analysis results from analyze_phonology tool
  - Grammar validation results from validate_grammar tool

**Format the summary to include:**
- Key vocabulary with definitions (including any from tool results)
- Phonological rules and sound inventory
- Grammatical structures and patterns
- Writing system details
- Example sentences or phrases
- Any constraints or preferences mentioned
- **Current lexicon state** (what words have been added)
- **Tool operations performed** and their results

**IMPORTANT: Include any lexicon entries that were added through tool calls, as these are part of the established vocabulary.**

Be comprehensive and include specific details rather than generic descriptions.`

	summaryMessages := []*schema.Message{schema.SystemMessage(summaryPrompt)}
	summaryMessages = append(summaryMessages, messages...)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := m.llm.Invoke(ctx, summaryMessages)
	if err != nil {
		log.Printf("Error generating context summary: %v", err)
		return m.formatExistingContext(messages[len(messages)-5:])
	}

	return response[0].Content
}

func (m *Model) formatExistingContext(messages []*schema.Message) string {
	if len(messages) == 0 {
		return "No previous conversation"
	}

	if len(messages) > 10 {
		messages = messages[len(messages)-10:]
	}

	context := strings.Builder{}
	context.WriteString("Previous conversation includes:\n\n")

	for _, msg := range messages {
		role := "User"
		if msg.Role == "assistant" {
			role = "Assistant"
		}

		content := msg.Content
		if strings.Contains(content, "[Tool Call:") {
			content = strings.ReplaceAll(content, "[Tool Call:", "**[Tool Call:")
			content = strings.ReplaceAll(content, "]", "]**")
		}

		context.WriteString(fmt.Sprintf("**%s:** %s\n\n", role, content))
	}

	return context.String()
}

// startStreaming starts the streaming process
func (m *Model) startStreaming(userMessage string) tea.Cmd {
	return func() tea.Msg {

		contextMessages := m.createCondensedHistory()

		requestMessage := schema.UserMessage("REQUEST: " + userMessage)

		messages := append(contextMessages, requestMessage)

		systemMessages := make([]*schema.Message, 0)
		for _, msg := range m.history {
			if msg.Role == "system" {
				systemMessages = append(systemMessages, msg)
			}
		}
		if len(systemMessages) > 0 {
			messages = append(systemMessages, messages...)
		}

		response, err := m.llm.Stream(context.Background(), messages)
		if err != nil {
			log.Printf("Streaming error: %v", err)
			m.thinking = false
			m.streaming = false
			m.updateViewportContent()
			return nil
		}

		m.tokenChan = make(chan string, 100)

		go func() {
			defer close(m.tokenChan)
			defer func() {
				m.thinking = false
			}()

			for {
				msg, err := response.Recv()
				if err == io.EOF {
					break
				} else if err != nil {
					log.Printf("Error receiving message: %v", err)
					break
				}

				if len(msg) > 0 {
					message := msg[0]

					if len(message.ToolCalls) > 0 {
						for _, toolCall := range message.ToolCalls {
							toolInfo := fmt.Sprintf("\n[Tool Call: %s]\n", toolCall.Function.Name)
							m.tokenChan <- toolInfo
						}
					}

					if message.Content != "" {
						m.tokenChan <- message.Content
					}

					m.UpdateStats(storage.Stats{TotalTokens: m.stats.TotalTokens + 1})
				}
			}
		}()

		return streamStartMsg{}
	}
}

// adjustOptimizationParams dynamically adjusts optimization parameters based on response length
func (m *Model) adjustOptimizationParams() {
	responseLength := m.currentResponse.Len()

	if responseLength > 10000 {
		m.renderThrottle = 200 * time.Millisecond
	} else if responseLength > 5000 {
		m.renderThrottle = 150 * time.Millisecond
	} else {
		m.renderThrottle = 100 * time.Millisecond
	}

	m.maxResponseLength = 0
}

// cleanupLongResponse is now disabled - no truncation
func (m *Model) cleanupLongResponse() {
}

// updateViewportContent updates the viewport with current content
func (m *Model) updateViewportContent() {
	if time.Since(m.lastRenderTime) < m.renderThrottle {
		return
	}
	m.lastRenderTime = time.Now()

	m.updateViewportContentInternal()
}

// updateViewportContentInternal does the actual viewport update without throttling
func (m *Model) updateViewportContentInternal() {
	logs := strings.Builder{}

	historyToShow := m.history
	if len(historyToShow) > m.maxHistoryDisplay {
		historyToShow = historyToShow[len(historyToShow)-m.maxHistoryDisplay:]
		logs.WriteString(fmt.Sprintf("... (showing last %d messages) ...\n\n", m.maxHistoryDisplay))
	}

	for _, msg := range historyToShow {
		role := string(msg.Role)
		if role == "user" {
			logs.WriteString("👤 User: " + msg.Content + "\n\n")
		} else if role == "assistant" {
			logs.WriteString("🤖 Assistant: " + msg.Content + "\n\n")
		} else if role == "system" {
			continue
		}
	}

	if m.streaming {
		logs.WriteString("=== Streaming Response ===\n\n")
		currentResponse := m.currentResponse.String()

		logs.WriteString(currentResponse)
		if m.currentResponse.Len() > 0 {
			logs.WriteString("▌")
		}
	}

	logsStr := logs.String()
	rendered, err := m.glam.Render(logsStr)
	if err != nil {
		log.Printf("Rendering error: %v", err)
		m.hold.SetContent(logsStr)
	} else {
		m.hold.SetContent(rendered)
	}

	if m.ready && m.hold.Height > 1 && len(m.hold.View()) > 0 {
		if m.hold.Height > 0 && m.hold.Width > 0 {
			m.hold.GotoBottom()
		}
	}
}

// View implements tea.Model.
func (m *Model) View() string {
	if m.quit {
		return ""
	}
	if !m.ready {
		return "Initializing..."
	}

	centerStyle := lipgloss.NewStyle().AlignHorizontal(lipgloss.Center)

	m.ta.SetWidth(m.width - 2)

	var doc []string

	if m.height > 20 {
		doc = []string{}

		maxLength := 0
		for _, row := range ascii {
			trimmedRow := strings.TrimLeft(row, " ")
			if len(trimmedRow) > maxLength {
				maxLength = len(trimmedRow)
			}
		}

		colors := []lipgloss.Color{
			lipgloss.Color("#FF6B6B"), // Red
			lipgloss.Color("#4ECDC4"), // Teal
			lipgloss.Color("#45B7D1"), // Blue
			lipgloss.Color("#96CEB4"), // Green
			lipgloss.Color("#FFEAA7"), // Yellow
			lipgloss.Color("#DDA0DD"), // Plum
			lipgloss.Color("#98D8C8"), // Mint
		}

		for i, row := range ascii {
			trimmedRow := strings.TrimLeft(row, " ")
			paddedRow := trimmedRow + strings.Repeat(" ", maxLength-len(trimmedRow))
			colorStyle := lipgloss.NewStyle().Foreground(colors[i%len(colors)])
			coloredRow := colorStyle.Render(paddedRow)
			doc = append(doc, centerStyle.Width(m.width).Render(coloredRow))
		}
		doc = append(doc, centerStyle.Width(m.width).Render(m.hold.View()))
		doc = append(doc, centerStyle.Width(m.width).Render(m.ta.View()))
	} else {
		doc = []string{
			centerStyle.Width(m.width).Render(m.hold.View()),
			centerStyle.Width(m.width).Render(m.ta.View()),
		}
	}

	return lipgloss.JoinVertical(lipgloss.Top, doc...)
}

func (m *Model) SetPrompts() {
	system, err := storage.ReadSystem()
	if err != nil {
		log.Fatal(err)
	}
	m.AddToHistory(schema.SystemMessage(system))
}

// SetLLM sets the LLM client
func (m *Model) SetLLM(llm compose.Runnable[[]*schema.Message, []*schema.Message]) {
	m.llm = llm
	m.SetPrompts()
}

// SetHistory sets the conversation history
func (m *Model) SetHistory(history []*schema.Message) {
	m.history = history
}
func (m *Model) SetStats() {
	stats, err := storage.ReadStats()
	if err != nil {
		log.Fatal(err)
	}
	m.stats = stats
}

func (m *Model) UpdateStats(newStats storage.Stats) {
	m.stats = newStats
	storage.WriteStats(m.stats)
}

func (m *Model) GetStats() storage.Stats {
	return m.stats
}

// resetOptimizationParams resets optimization parameters to default values
func (m *Model) resetOptimizationParams() {
	m.maxHistoryDisplay = 10
	m.maxResponseLength = 0 // Disable truncation
	m.renderBuffer = 5
	m.renderThrottle = 100 * time.Millisecond
}
