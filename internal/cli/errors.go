package cli

import (
	"fmt"
)

// Exit codes for agent compatibility
const (
	CodeSuccess    = 0
	CodeGeneralErr = 1
	CodeBadArgs    = 2
	CodeNotFound   = 3
	CodeAuthFail   = 4
	CodeConflict   = 5
	CodeTimeout    = 6
	CodePrecondFail = 7
)

// AgentActionableError represents an error that can be programmatically 
// handled by an AI agent or other automated system.
type AgentActionableError struct {
	ErrorCode  string `json:"error"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	Code       int    `json:"code"`
}

func (e *AgentActionableError) Error() string {
	return fmt.Sprintf("[%s] %s (Exit Code: %d)", e.ErrorCode, e.Message, e.Code)
}

// NewError creates a new AgentActionableError
func NewError(code int, errCode string, message string, suggestion string) *AgentActionableError {
	return &AgentActionableError{
		Code:       code,
		ErrorCode:  errCode,
		Message:    message,
		Suggestion: suggestion,
	}
}
