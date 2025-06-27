package config

import (
	"context"
	"l2/tools"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

// NewLLMClient creates and configures a new LLM client with tools
func NewLLMClient() compose.Runnable[[]*schema.Message, []*schema.Message] {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Failed to load .env file: %v", err)
	}

	// Create chat model
	client, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		// Model:   "deepseek/deepseek-r1-0528-qwen3-8b:free",
		Model:   "google/gemini-2.5-flash",
		BaseURL: "https://openrouter.ai/api/v1",
		APIKey:  os.Getenv("OPENROUTER"),
	})
	if err != nil {
		log.Fatalf("Failed to create chat model: %v", err)
	}

	// Get tool information and bind to client
	toolInfos := tools.ToolsInfo()
	if toolInfos == nil {
		log.Fatal("Failed to get tool information")
	}

	// Log tool information for debugging
	log.Printf("Available tools: %d", len(toolInfos))
	for _, tool := range toolInfos {
		log.Printf("Tool: %s", tool.Name)
	}

	if err := client.BindTools(toolInfos); err != nil {
		log.Fatalf("Failed to bind tools to client: %v", err)
	}

	// Build the processing chain
	chain := compose.NewChain[[]*schema.Message, []*schema.Message]()

	// Add a system message to instruct the model about tool usage
	toolInstructions := `

**Tool Usage Guidelines:**
Use tools for actual data operations, but be creative for examples and suggestions.

**Use tools when:**
- Users ask to retrieve stored lexicon data → Use get_lexicon tool
- Users ask to save new words to the lexicon → Use add_lexicon_entry tool  
- Users ask to read existing files → Use read_file tool
- Users ask to save new files → Use add_file tool
- Users ask to analyze phonology of specific text → Use analyze_phonology tool
- Users ask to validate grammar of specific text → Use validate_grammar tool
- **CRITICAL: When you just defined a word and the user says "Yes" to adding it → Use add_lexicon_entry tool immediately**
- **CRITICAL: When you propose a word definition and user agrees → Use add_lexicon_entry tool**

**Do NOT use tools when:**
- Users ask for example words, translations, or creative suggestions → Provide these directly
- Users ask for made-up vocabulary or example sentences → Create these yourself
- Users ask for hypothetical language features → Describe and demonstrate them directly

**Available Tools:**
- **get_lexicon**: Retrieve all entries from the conlang lexicon
- **add_lexicon_entry**: Add words to the conlang lexicon with definition, part of speech, and etymology
- **analyze_phonology**: Analyze text phonology using IPA notation, extract phonemes, allophones, and syllable structure
- **validate_grammar**: Validate text against grammar rules and provide suggestions
- **read_file**: Read stored conlang documentation, grammar rules, vocabulary lists, and other language resources
- **add_file**: Create or overwrite files for storing conlang documentation, grammar rules, vocabulary lists, and other language resources

**IMPORTANT: When you propose a word definition and the user agrees (says "Yes", "Add it", etc.), immediately use the add_lexicon_entry tool with the word you just defined.**
**Be flexible and creative when users ask for examples or suggestions.**`

	toolsNode := tools.Tools()
	chain.
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) ([]*schema.Message, error) {
			// Read system prompt from system.md
			systemContent, err := os.ReadFile("system.md")
			if err != nil {
				log.Printf("Warning: Failed to read system.md: %v", err)
				// Fallback to basic system prompt
				systemMsg := schema.SystemMessage("You are ConlangGPT, a comprehensive expert assistant for designing and exploring constructed languages (conlangs)." + toolInstructions)
				return append([]*schema.Message{systemMsg}, input...), nil
			}
			// Combine system prompt with tool instructions
			fullSystemPrompt := string(systemContent) + toolInstructions
			systemMsg := schema.SystemMessage(fullSystemPrompt)
			return append([]*schema.Message{systemMsg}, input...), nil
		})).
		AppendChatModel(client, compose.WithNodeName("chat_model")).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, input *schema.Message) ([]*schema.Message, error) {
			if len(input.ToolCalls) > 0 {
				return toolsNode.Invoke(ctx, input)
			}
			return []*schema.Message{input}, nil
		}))

	// Compile the chain
	agent, err := chain.Compile(context.Background())
	if err != nil {
		log.Fatalf("Failed to compile agent chain: %v", err)
	}

	return agent
}
