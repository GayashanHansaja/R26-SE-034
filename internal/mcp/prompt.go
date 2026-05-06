package mcp

type Prompt struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Arguments   []PromptArgument  `json:"arguments,omitempty"`
	Template    string            `json:"template"` // The instruction to the AI
}

type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}
