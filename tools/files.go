package tools

import (
	"context"
	"l2/storage"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// File represents a file operation request
type File struct {
	Path    string `json:"path" jsonschema:"required,description=The path of the file to operate on"`
	Content string `json:"content" jsonschema:"description=The content to write to the file (required for write operations)"`
}

// Result represents the result of a file operation
type Result struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Content string `json:"content,omitempty"`
}

// AddFile creates or overwrites a file with the specified content
func AddFile(ctx context.Context, file *File) (*Result, error) {
	if file.Path == "" {
		return &Result{
			Success: false,
			Message: "File path is required",
		}, nil
	}

	if file.Content == "" {
		return &Result{
			Success: false,
			Message: "File content is required for write operations",
		}, nil
	}

	err := storage.WriteDataFile(file.Path, []byte(file.Content))
	if err != nil {
		return &Result{
			Success: false,
			Message: "Failed to write file: " + err.Error(),
		}, nil
	}

	return &Result{
		Success: true,
		Message: "File written successfully",
	}, nil
}

// ReadFile reads the content of a file
func ReadFile(ctx context.Context, file *File) (*Result, error) {
	if file.Path == "" {
		return &Result{
			Success: false,
			Message: "File path is required",
		}, nil
	}

	data, err := storage.ReadDataFile(file.Path)
	if err != nil {
		return &Result{
			Success: false,
			Message: "Failed to read file: " + err.Error(),
		}, nil
	}

	return &Result{
		Success: true,
		Message: "File read successfully",
		Content: string(data),
	}, nil
}

// createAddFileTool creates the add file tool
func createAddFileTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"add_file",
		"Create or overwrite a file with specified content. Use this tool to store conlang documentation, grammar rules, vocabulary lists, and other language resources.",
		AddFile,
	)
}

// createReadFileTool creates the read file tool
func createReadFileTool() (tool.InvokableTool, error) {
	return utils.InferTool(
		"read_file",
		"Read the content of a file. Use this tool to retrieve stored conlang documentation, grammar rules, vocabulary lists, and other language resources.",
		ReadFile,
	)
}

// Tools creates and returns a ToolsNode with all available tools
func Tools() *compose.ToolsNode {
	// Create file management tools
	addFileTool, err := createAddFileTool()
	if err != nil {
		log.Printf("Failed to create add file tool: %v", err)
	}

	readFileTool, err := createReadFileTool()
	if err != nil {
		log.Printf("Failed to create read file tool: %v", err)
	}

	// Create conlang-specific tools
	phonologyTool, err := createPhonologyTool()
	if err != nil {
		log.Printf("Failed to create phonology tool: %v", err)
	}

	grammarTool, err := createGrammarTool()
	if err != nil {
		log.Printf("Failed to create grammar tool: %v", err)
	}

	addLexiconTool, err := createAddLexiconTool()
	if err != nil {
		log.Printf("Failed to create add lexicon tool: %v", err)
	}

	getLexiconTool, err := createGetLexiconTool()
	if err != nil {
		log.Printf("Failed to create get lexicon tool: %v", err)
	}

	// Collect all tools
	tools := []tool.BaseTool{}
	if addFileTool != nil {
		tools = append(tools, addFileTool)
	}
	if readFileTool != nil {
		tools = append(tools, readFileTool)
	}
	if phonologyTool != nil {
		tools = append(tools, phonologyTool)
	}
	if grammarTool != nil {
		tools = append(tools, grammarTool)
	}
	if addLexiconTool != nil {
		tools = append(tools, addLexiconTool)
	}
	if getLexiconTool != nil {
		tools = append(tools, getLexiconTool)
	}

	if len(tools) == 0 {
		log.Printf("No tools could be created")
		return nil
	}

	conf := &compose.ToolsNodeConfig{
		Tools: tools,
	}

	toolsNode, err := compose.NewToolNode(context.Background(), conf)
	if err != nil {
		log.Printf("Failed to create tools node: %v", err)
		return nil
	}

	return toolsNode
}

// ToolsInfo returns information about all available tools
func ToolsInfo() []*schema.ToolInfo {
	// Get file management tool info
	addFileTool, err := createAddFileTool()
	if err != nil {
		log.Printf("Failed to create add file tool for info: %v", err)
	}

	readFileTool, err := createReadFileTool()
	if err != nil {
		log.Printf("Failed to create read file tool for info: %v", err)
	}

	// Get conlang tool info
	phonologyTool, err := createPhonologyTool()
	if err != nil {
		log.Printf("Failed to create phonology tool for info: %v", err)
	}

	grammarTool, err := createGrammarTool()
	if err != nil {
		log.Printf("Failed to create grammar tool for info: %v", err)
	}

	addLexiconTool, err := createAddLexiconTool()
	if err != nil {
		log.Printf("Failed to create add lexicon tool for info: %v", err)
	}

	getLexiconTool, err := createGetLexiconTool()
	if err != nil {
		log.Printf("Failed to create get lexicon tool for info: %v", err)
	}

	// Collect all tools
	tools := []tool.BaseTool{}
	if addFileTool != nil {
		tools = append(tools, addFileTool)
	}
	if readFileTool != nil {
		tools = append(tools, readFileTool)
	}
	if phonologyTool != nil {
		tools = append(tools, phonologyTool)
	}
	if grammarTool != nil {
		tools = append(tools, grammarTool)
	}
	if addLexiconTool != nil {
		tools = append(tools, addLexiconTool)
	}
	if getLexiconTool != nil {
		tools = append(tools, getLexiconTool)
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
