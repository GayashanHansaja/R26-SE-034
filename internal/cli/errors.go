package cli

import (
	"fmt"
)

// Exit codes for agent compatibility.
const (
	// CodeSuccess indicates the command completed successfully.
	CodeSuccess = 0
	// CodeGeneralErr indicates an unspecified error occurred.
	CodeGeneralErr = 1
	// CodeBadArgs indicates invalid arguments were provided.
	CodeBadArgs = 2
	// CodeNotFound indicates a requested resource was not found.
	CodeNotFound = 3
	// CodeAuthFail indicates an authentication failure.
	CodeAuthFail = 4
	// CodeConflict indicates a resource conflict.
	CodeConflict = 5
	// CodeTimeout indicates the operation timed out.
	CodeTimeout = 6
	// CodePrecondFail indicates a precondition for the command was not met.
	CodePrecondFail = 7
)

// AgentActionableError represents an error that can be programmatically
// handled by an AI agent or other automated system.
type AgentActionableError struct {
	// ErrorCode is a machine-readable string identifying the error type.
	ErrorCode string `json:"error"`
	// Message is a human-readable description of the error.
	Message string `json:"message"`
	// Suggestion provides a possible fix or next step.
	Suggestion string `json:"suggestion,omitempty"`
	// Code is the numeric exit code associated with this error.
	Code int `json:"code"`
}

// Error implements the error interface.
func (e *AgentActionableError) Error() string {
	return fmt.Sprintf("[%s] %s (Exit Code: %d)", e.ErrorCode, e.Message, e.Code)
}

// NewError creates a new AgentActionableError with the provided details.
func NewError(code int, errCode string, message string, suggestion string) *AgentActionableError {
	return &AgentActionableError{
		Code:       code,
		ErrorCode:  errCode,
		Message:    message,
		Suggestion: suggestion,
	}
}
