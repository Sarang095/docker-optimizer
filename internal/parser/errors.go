package parser

import (
    "fmt"
    "strings"
    "github.com/pkg/errors"
)

// Common error variables
var (
    ErrEmptyDockerfile    = errors.New("dockerfile is empty")
    ErrInvalidSyntax      = errors.New("invalid dockerfile syntax")
    ErrInvalidInstruction = errors.New("invalid instruction")
    ErrCircularDependency = errors.New("circular dependency detected in build stages")
    ErrMissingStage       = errors.New("referenced stage not found")
    ErrDuplicateStage     = errors.New("duplicate stage name")
    ErrInvalidBase        = errors.New("invalid base image specification")
)

// ErrorCode represents specific error types for better error handling
type ErrorCode int

const (
    CodeSyntaxError ErrorCode = iota + 1
    CodeValidationError
    CodeReferenceError
    CodeInstructionError
    CodeStageError
    CodeVariableError
    CodeIOError
    CodeInternalError
)

// DockerfileError provides detailed error information
type DockerfileError struct {
    Code     ErrorCode
    Stage    string    // Build stage name if applicable
    Position Position  // Error location
    Message  string    // User-friendly message
    Details  string    // Technical details
    Snippet  string    // Problematic code snippet
    Hints    []string  // Suggested fixes
    Cause    error    // Underlying error
}

func (e *DockerfileError) Error() string {
    var sb strings.Builder
    
    // Location information
    if e.Stage != "" {
        sb.WriteString(fmt.Sprintf("Stage '%s': ", e.Stage))
    }
    
    // Basic error message
    sb.WriteString(fmt.Sprintf("Line %d:%d - %s\n", 
        e.Position.Line, 
        e.Position.Column, 
        e.Message))
    
    // Code snippet if available
    if e.Snippet != "" {
        sb.WriteString("\nProblematic code:\n")
        sb.WriteString(e.Snippet + "\n")
        // Mark the specific position
        if e.Position.Column > 0 {
            sb.WriteString(strings.Repeat(" ", e.Position.Column-1) + "^\n")
        }
    }
    
    // Add technical details if available
    if e.Details != "" {
        sb.WriteString("\nDetails: " + e.Details + "\n")
    }
    
    // Add hints if available
    if len(e.Hints) > 0 {
        sb.WriteString("\nSuggestions:\n")
        for _, hint := range e.Hints {
            sb.WriteString("- " + hint + "\n")
        }
    }
    
    return sb.String()
}

func (e *DockerfileError) Unwrap() error {
    return e.Cause
}

// ErrorCollector collects multiple errors during parsing
type ErrorCollector struct {
    errors []error
}

func NewErrorCollector() *ErrorCollector {
    return &ErrorCollector{
        errors: make([]error, 0),
    }
}

func (c *ErrorCollector) Add(err error) {
    if err != nil {
        c.errors = append(c.errors, err)
    }
}

func (c *ErrorCollector) HasErrors() bool {
    return len(c.errors) > 0
}

func (c *ErrorCollector) Errors() []error {
    return c.errors
}

// Error constructors
func NewSyntaxError(pos Position, message string, snippet string) *DockerfileError {
    return &DockerfileError{
        Code:     CodeSyntaxError,
        Position: pos,
        Message:  message,
        Snippet:  snippet,
        Hints:    getSyntaxErrorHints(message),
    }
}

func NewStageError(stageName string, pos Position, message string) *DockerfileError {
    return &DockerfileError{
        Code:     CodeStageError,
        Stage:    stageName,
        Position: pos,
        Message:  message,
    }
}

func NewInstructionError(pos Position, instruction string, message string) *DockerfileError {
    return &DockerfileError{
        Code:     CodeInstructionError,
        Position: pos,
        Message:  fmt.Sprintf("Invalid %s instruction: %s", instruction, message),
    }
}

// Helper functions for error handling
func getSyntaxErrorHints(errMsg string) []string {
    hints := make([]string, 0)
    
    commonErrors := map[string]string{
        "unknown instruction": "Instruction ka naam uppercase mein likhein (jaise FROM, RUN, COPY)",
        "missing separator": "Instruction aur uske arguments ke beech space daalna zaroori hai",
        "unexpected EOF": "Kisi instruction ya quote ko close karna bhool gaye hain",
        "empty continuation": "Line continuation (\\) ke baad kuch content hona chahiye",
        "invalid reference format": "Base image ya stage ka reference format sahi nahi hai",
        "invalid port": "Port number valid range (1-65535) mein hona chahiye",
        "invalid syntax": "Dockerfile syntax check karein aur documentation follow karein",
    }
    
    for pattern, hint := range commonErrors {
        if strings.Contains(strings.ToLower(errMsg), pattern) {
            hints = append(hints, hint)
        }
    }
    
    // Add general hint if no specific hints found
    if len(hints) == 0 {
        hints = append(hints, "Docker documentation check karein: https://docs.docker.com/engine/reference/builder/")
    }
    
    return hints
}

// Context provides additional error context
type ErrorContext struct {
    Filename    string
    BuildStage  string
    Instruction string
    Line        string
}

// ErrorHandler manages error handling during parsing
type ErrorHandler struct {
    collector  *ErrorCollector
    context    ErrorContext
}

func NewErrorHandler() *ErrorHandler {
    return &ErrorHandler{
        collector: NewErrorCollector(),
    }
}

func (h *ErrorHandler) WithContext(ctx ErrorContext) *ErrorHandler {
    h.context = ctx
    return h
}

func (h *ErrorHandler) HandleError(err error) {
    if err == nil {
        return
    }

    // Convert to DockerfileError if needed
    var dockerfileErr *DockerfileError
    if !errors.As(err, &dockerfileErr) {
        dockerfileErr = &DockerfileError{
            Code:    CodeInternalError,
            Message: err.Error(),
            Cause:   err,
        }
    }

    // Add context information
    if h.context.BuildStage != "" {
        dockerfileErr.Stage = h.context.BuildStage
    }

    h.collector.Add(dockerfileErr)
}