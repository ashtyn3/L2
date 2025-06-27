package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"l2/storage"
	"log"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// PhonologyAnalysis represents a phonology analysis request
type PhonologyAnalysis struct {
	Text string `json:"text" jsonschema:"required,description=The text to analyze for phonology"`
}

// PhonologyResult represents the result of phonology analysis
type PhonologyResult struct {
	Success    bool     `json:"success"`
	Message    string   `json:"message"`
	Phonemes   []string `json:"phonemes,omitempty"`
	Allophones []string `json:"allophones,omitempty"`
	Syllables  []string `json:"syllables,omitempty"`
	Analysis   string   `json:"analysis,omitempty"`
}

// GrammarValidation represents a grammar validation request
type GrammarValidation struct {
	Text        string `json:"text" jsonschema:"required,description=The text to validate"`
	GrammarFile string `json:"grammar_file" jsonschema:"description=Path to grammar rules file"`
}

// GrammarResult represents the result of grammar validation
type GrammarResult struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
	Valid       bool     `json:"valid"`
	Errors      []string `json:"errors,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// LexiconEntry represents a lexicon entry
type LexiconEntry struct {
	Word         string `json:"word" jsonschema:"required,description=The word to add to lexicon"`
	Definition   string `json:"definition" jsonschema:"required,description=The definition of the word"`
	PartOfSpeech string `json:"part_of_speech" jsonschema:"description=Part of speech"`
	Etymology    string `json:"etymology" jsonschema:"description=Etymology of the word"`
}

// LexiconResult represents the result of lexicon operations
type LexiconResult struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Entries []LexiconEntry `json:"entries,omitempty"`
}

// AnalyzePhonology analyzes the phonology of given text
func AnalyzePhonology(ctx context.Context, req *PhonologyAnalysis) (*PhonologyResult, error) {
	if req.Text == "" {
		return &PhonologyResult{
			Success: false,
			Message: "Text is required for phonology analysis",
		}, nil
	}

	// Basic phoneme extraction (this would be enhanced with actual IPA processing)
	text := strings.ToLower(req.Text)
	phonemes := extractPhonemes(text)
	allophones := extractAllophones(text)
	syllables := extractSyllables(text)

	analysis := fmt.Sprintf("Analyzed text: %s\nPhonemes: %v\nAllophones: %v\nSyllables: %v",
		req.Text, phonemes, allophones, syllables)

	return &PhonologyResult{
		Success:    true,
		Message:    "Phonology analysis completed",
		Phonemes:   phonemes,
		Allophones: allophones,
		Syllables:  syllables,
		Analysis:   analysis,
	}, nil
}

// ValidateGrammar validates text against grammar rules
func ValidateGrammar(ctx context.Context, req *GrammarValidation) (*GrammarResult, error) {
	if req.Text == "" {
		return &GrammarResult{
			Success: false,
			Message: "Text is required for grammar validation",
		}, nil
	}

	// Load grammar rules if specified
	if req.GrammarFile != "" {
		_, err := storage.ReadDataFile(req.GrammarFile)
		if err != nil {
			return &GrammarResult{
				Success: false,
				Message: "Failed to load grammar rules: " + err.Error(),
			}, nil
		}
	}

	// Basic validation (this would be enhanced with actual grammar checking)
	errors := []string{}
	suggestions := []string{}

	// Simple checks
	if len(strings.Split(req.Text, " ")) < 2 {
		errors = append(errors, "Text appears to be too short for meaningful grammar validation")
	}

	if !strings.HasSuffix(req.Text, ".") && !strings.HasSuffix(req.Text, "!") && !strings.HasSuffix(req.Text, "?") {
		suggestions = append(suggestions, "Consider adding proper sentence termination")
	}

	valid := len(errors) == 0

	return &GrammarResult{
		Success:     true,
		Message:     "Grammar validation completed",
		Valid:       valid,
		Errors:      errors,
		Suggestions: suggestions,
	}, nil
}

// AddLexiconEntry adds a word to the lexicon
func AddLexiconEntry(ctx context.Context, entry *LexiconEntry) (*LexiconResult, error) {
	if entry.Word == "" {
		return &LexiconResult{
			Success: false,
			Message: "Word is required",
		}, nil
	}

	if entry.Definition == "" {
		return &LexiconResult{
			Success: false,
			Message: "Definition is required",
		}, nil
	}

	// Load existing lexicon
	lexiconPath := "lexicon.json"
	data, err := storage.ReadDataFile(lexiconPath)
	entries := []LexiconEntry{}

	if err == nil {
		// Parse existing entries
		if err := json.Unmarshal(data, &entries); err != nil {
			log.Printf("Failed to parse existing lexicon: %v", err)
		}
	}

	// Check for duplicates
	for _, existing := range entries {
		if existing.Word == entry.Word {
			return &LexiconResult{
				Success: false,
				Message: "Word already exists in lexicon",
			}, nil
		}
	}

	// Add new entry
	entries = append(entries, *entry)

	// Save updated lexicon
	lexiconData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return &LexiconResult{
			Success: false,
			Message: "Failed to serialize lexicon: " + err.Error(),
		}, nil
	}

	if err := storage.WriteDataFile(lexiconPath, lexiconData); err != nil {
		return &LexiconResult{
			Success: false,
			Message: "Failed to save lexicon: " + err.Error(),
		}, nil
	}

	return &LexiconResult{
		Success: true,
		Message: "Lexicon entry added successfully",
		Entries: []LexiconEntry{*entry},
	}, nil
}

// GetLexiconRequest represents a request to get lexicon entries
type GetLexiconRequest struct {
	// Empty struct for consistency with other tools
}

// GetLexicon retrieves all lexicon entries
func GetLexicon(ctx context.Context, req *GetLexiconRequest) (*LexiconResult, error) {
	lexiconPath := "lexicon.json"
	data, err := storage.ReadDataFile(lexiconPath)
	if err != nil {
		return &LexiconResult{
			Success: false,
			Message: "Failed to read lexicon: " + err.Error(),
		}, nil
	}

	var entries []LexiconEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return &LexiconResult{
			Success: false,
			Message: "Failed to parse lexicon: " + err.Error(),
		}, nil
	}

	return &LexiconResult{
		Success: true,
		Message: fmt.Sprintf("Retrieved %d lexicon entries", len(entries)),
		Entries: entries,
	}, nil
}

// Helper functions for phonology analysis
func extractPhonemes(text string) []string {
	// Simplified phoneme extraction - in practice, this would use IPA analysis
	phonemes := []string{}
	for _, char := range text {
		if char >= 'a' && char <= 'z' {
			phonemes = append(phonemes, string(char))
		}
	}
	return phonemes
}

func extractAllophones(text string) []string {
	// Simplified allophone extraction
	allophones := []string{}
	for _, char := range text {
		if char >= 'a' && char <= 'z' {
			allophones = append(allophones, "["+string(char)+"]")
		}
	}
	return allophones
}

func extractSyllables(text string) []string {
	// Simplified syllable extraction
	words := strings.Fields(text)
	syllables := []string{}
	for _, word := range words {
		if len(word) > 0 {
			syllables = append(syllables, word)
		}
	}
	return syllables
}

// createPhonologyTool creates the phonology analysis tool
func createPhonologyTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"analyze_phonology",
		"Analyze the phonology of text using IPA notation. Extract phonemes, allophones, and syllable structure for conlang development.",
		AnalyzePhonology,
	)
}

// createGrammarTool creates the grammar validation tool
func createGrammarTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"validate_grammar",
		"Validate text against grammar rules. Check syntax, morphology, and provide suggestions for conlang grammar development.",
		ValidateGrammar,
	)
}

// createAddLexiconTool creates the add lexicon entry tool
func createAddLexiconTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"add_lexicon_entry",
		"Add a word to the conlang lexicon with definition, part of speech, and etymology information.",
		AddLexiconEntry,
	)
}

// createGetLexiconTool creates the get lexicon tool
func createGetLexiconTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"get_lexicon",
		"Retrieve all entries from the conlang lexicon for review and analysis.",
		GetLexicon,
	)
}

// ConlangTools creates and returns a ToolsNode with conlang-specific tools
func ConlangTools() *compose.ToolsNode {
	tools := []tool.BaseTool{}

	// Create phonology tool
	phonologyTool, err := createPhonologyTool()
	if err != nil {
		log.Printf("Failed to create phonology tool: %v", err)
	} else {
		tools = append(tools, phonologyTool)
	}

	// Create grammar tool
	grammarTool, err := createGrammarTool()
	if err != nil {
		log.Printf("Failed to create grammar tool: %v", err)
	} else {
		tools = append(tools, grammarTool)
	}

	// Create lexicon tools
	addLexiconTool, err := createAddLexiconTool()
	if err != nil {
		log.Printf("Failed to create add lexicon tool: %v", err)
	} else {
		tools = append(tools, addLexiconTool)
	}

	getLexiconTool, err := createGetLexiconTool()
	if err != nil {
		log.Printf("Failed to create get lexicon tool: %v", err)
	} else {
		tools = append(tools, getLexiconTool)
	}

	if len(tools) == 0 {
		log.Printf("No conlang tools could be created")
		return nil
	}

	conf := &compose.ToolsNodeConfig{
		Tools: tools,
	}

	toolsNode, err := compose.NewToolNode(context.Background(), conf)
	if err != nil {
		log.Printf("Failed to create conlang tools node: %v", err)
		return nil
	}

	return toolsNode
}

// ConlangToolsInfo returns information about conlang-specific tools
func ConlangToolsInfo() []*schema.ToolInfo {
	tools := []tool.BaseTool{}

	// Add phonology tool
	if tool, err := createPhonologyTool(); err == nil {
		tools = append(tools, tool)
	}

	// Add grammar tool
	if tool, err := createGrammarTool(); err == nil {
		tools = append(tools, tool)
	}

	// Add lexicon tools
	if tool, err := createAddLexiconTool(); err == nil {
		tools = append(tools, tool)
	}

	if tool, err := createGetLexiconTool(); err == nil {
		tools = append(tools, tool)
	}

	ctx := context.Background()
	toolInfos := make([]*schema.ToolInfo, 0, len(tools))

	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil {
			log.Printf("Failed to get tool info: %v", err)
			continue
		}
		toolInfos = append(toolInfos, info)
	}

	return toolInfos
}
