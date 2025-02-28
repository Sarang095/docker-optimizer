package parser

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/yourusername/dockerfile-parser/internal/lexer"
)

// InstructionParser parses individual Dockerfile instructions
type InstructionParser struct {
	errorHandler *ErrorHandler
}

// NewInstructionParser creates a new instruction parser
func NewInstructionParser() *InstructionParser {
	return &InstructionParser{
		errorHandler: NewErrorHandler(),
	}
}

// ParseInstruction takes tokenized instruction data and converts it to an Instruction struct
func (p *InstructionParser) ParseInstruction(tokens *lexer.InstructionTokens, stage *Stage) (*Instruction, error) {
	if tokens == nil || tokens.Instruction == nil {
		return nil, ErrInvalidInstruction
	}

	command := tokens.GetInstructionValue()
	instruction := &Instruction{
		Command: command,
		Raw:     tokens.GetArgumentsAsString(),
		Flags:   make(map[string]string),
		Stage:   stage,
		Range: Range{
			Start: Position{
				Line:   tokens.Instruction.Line,
				Column: tokens.Instruction.Column,
			},
			// End position will be set later when we have all tokens
		},
		JSONForm: tokens.JSONForm,
	}

	// Extract comments
	if len(tokens.Comments) > 0 {
		commentStr := ""
		for _, cmt := range tokens.Comments {
			// Remove # prefix from comment
			text := strings.TrimPrefix(cmt.Value, "#")
			text = strings.TrimSpace(text)
			commentStr += text + "\n"
		}
		instruction.Comment = strings.TrimSuffix(commentStr, "\n")
	}

	// Set end position
	if len(tokens.Raw) > 0 {
		lastToken := tokens.Raw[len(tokens.Raw)-1]
		instruction.Range.End = Position{
			Line:   lastToken.Line,
			Column: lastToken.Column + len(lastToken.Value),
		}
	}

	// Parse instruction arguments based on command type
	var err error
	switch command {
	case "FROM":
		err = p.parseFromInstruction(tokens, instruction)
	case "RUN":
		err = p.parseRunInstruction(tokens, instruction)
	case "CMD":
		err = p.parseCmdInstruction(tokens, instruction)
	case "LABEL":
		err = p.parseLabelInstruction(tokens, instruction)
	case "EXPOSE":
		err = p.parseExposeInstruction(tokens, instruction)
	case "ENV":
		err = p.parseEnvInstruction(tokens, instruction)
	case "ADD":
		err = p.parseAddCopyInstruction(tokens, instruction)
	case "COPY":
		err = p.parseAddCopyInstruction(tokens, instruction)
	case "ENTRYPOINT":
		err = p.parseEntrypointInstruction(tokens, instruction)
	case "VOLUME":
		err = p.parseVolumeInstruction(tokens, instruction)
	case "USER":
		err = p.parseUserInstruction(tokens, instruction)
	case "WORKDIR":
		err = p.parseWorkdirInstruction(tokens, instruction)
	case "ARG":
		err = p.parseArgInstruction(tokens, instruction)
	case "ONBUILD":
		err = p.parseOnbuildInstruction(tokens, instruction)
	case "STOPSIGNAL":
		err = p.parseStopsignalInstruction(tokens, instruction)
	case "HEALTHCHECK":
		err = p.parseHealthcheckInstruction(tokens, instruction)
	case "SHELL":
		err = p.parseShellInstruction(tokens, instruction)
	default:
		err = fmt.Errorf("unknown instruction: %s", command)
	}

	if err != nil {
		return nil, err
	}

	return instruction, nil
}

// Parse FROM instruction
func (p *InstructionParser) parseFromInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	args := tokens.Arguments
	if len(args) == 0 {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "FROM instruction requires a base image",
			Position: instruction.Range.Start,
		}
	}

	// Check for stage name (AS name)
	for i := 0; i < len(args); i++ {
		if args[i].Type == lexer.TOKEN_AS && i+1 < len(args) {
			stageName := args[i+1].Value
			instruction.Flags["stage"] = stageName
			break
		}
	}

	// Parse platform flag
	for i := 0; i < len(args); i++ {
		if args[i].Type == lexer.TOKEN_STRING && strings.HasPrefix(args[i].Value, "--platform=") {
			platform := strings.TrimPrefix(args[i].Value, "--platform=")
			instruction.Flags["platform"] = platform
			break
		}
	}

	// The first argument that's not a flag is the base image
	for _, arg := range args {
		if arg.Type == lexer.TOKEN_STRING && !strings.HasPrefix(arg.Value, "--") {
			instruction.Args = append(instruction.Args, arg.Value)
			break
		}
	}

	return nil
}

// Parse RUN instruction
func (p *InstructionParser) parseRunInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	if tokens.JSONForm {
		// Handle JSON array form
		return p.parseJSONArrayForm(tokens, instruction)
	}

	// Handle shell form (default)
	args := tokens.GetArgumentsAsString()
	if args == "" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "RUN instruction requires at least one argument",
			Position: instruction.Range.Start,
		}
	}

	// Check for heredoc
	for _, token := range tokens.Raw {
		if token.Type == lexer.TOKEN_HEREDOC_START {
			heredocContent := ""
			// Find heredoc content in subsequent tokens
			for _, t := range tokens.Raw {
				if t.Type == lexer.TOKEN_HEREDOC_CONTENT {
					heredocContent = t.Value
					break
				}
			}
			
			if heredocContent != "" {
				instruction.Heredoc = &Heredoc{
					Identifier: token.Value,
					Content:    heredocContent,
					Delimiter:  token.Value,
					Range: Range{
						Start: Position{Line: token.Line, Column: token.Column},
						// End position approximate since we don't track heredoc end position precisely
						End: Position{Line: token.Line + strings.Count(heredocContent, "\n") + 1, Column: 0},
					},
				}
			}
			break
		}
	}

	instruction.Args = []string{args}
	return nil
}

// Parse CMD instruction
func (p *InstructionParser) parseCmdInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	if tokens.JSONForm {
		return p.parseJSONArrayForm(tokens, instruction)
	}

	// Handle shell form
	args := tokens.GetArgumentsAsString()
	if args == "" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "CMD instruction requires at least one argument",
			Position: instruction.Range.Start,
		}
	}

	instruction.Args = []string{args}
	return nil
}

// Parse ENTRYPOINT instruction
func (p *InstructionParser) parseEntrypointInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	return p.parseCmdInstruction(tokens, instruction) // Same parsing logic as CMD
}

// Parse JSON array form (used for CMD, RUN, ENTRYPOINT, etc.)
func (p *InstructionParser) parseJSONArrayForm(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	// Find and parse the JSON array
	jsonStr := ""
	for _, token := range tokens.Arguments {
		if token.Type != lexer.TOKEN_WHITESPACE {
			jsonStr = token.Raw
			break
		}
	}

	if jsonStr == "" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "Missing JSON array argument",
			Position: instruction.Range.Start,
		}
	}

	// Parse JSON array
	var args []string
	err := json.Unmarshal([]byte(jsonStr), &args)
	if err != nil {
		return &DockerfileError{
			Code:     CodeSyntaxError,
			Message:  "Invalid JSON array: " + err.Error(),
			Position: instruction.Range.Start,
			Snippet:  jsonStr,
		}
	}

	instruction.Args = args
	instruction.JSONForm = true
	return nil
}

// Parse LABEL instruction
func (p *InstructionParser) parseLabelInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	// LABEL requires key-value pairs
	args := tokens.GetArgumentsAsString()
	if args == "" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "LABEL instruction requires at least one key-value pair",
			Position: instruction.Range.Start,
		}
	}

	// Parse key-value pairs
	pairs := parseKeyValuePairs(args)
	for k, v := range pairs {
		instruction.Args = append(instruction.Args, k+"="+v)
	}

	return nil
}

// Parse EXPOSE instruction
func (p *InstructionParser) parseExposeInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	// EXPOSE requires at least one port
	if len(tokens.Arguments) == 0 {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "EXPOSE instruction requires at least one port",
			Position: instruction.Range.Start,
		}
	}

	// Collect ports
	for _, token := range tokens.Arguments {
		if token.Type != lexer.TOKEN_WHITESPACE {
			// Validate port format
			port := token.Value
			if strings.Contains(port, "/") {
				parts := strings.Split(port, "/")
				port = parts[0]
				protocol := parts[1]
				if protocol != "tcp" && protocol != "udp" {
					return &DockerfileError{
						Code:     CodeInstructionError,
						Message:  "Invalid protocol: " + protocol + ". Must be tcp or udp",
						Position: Position{Line: token.Line, Column: token.Column},
					}
				}
			}

			// Validate port number
			if _, err := strconv.Atoi(port); err != nil {
				return &DockerfileError{
					Code:     CodeInstructionError,
					Message:  "Invalid port number: " + port,
					Position: Position{Line: token.Line, Column: token.Column},
				}
			}

			instruction.Args = append(instruction.Args, token.Value)
		}
	}

	return nil
}

// Parse ENV instruction
func (p *InstructionParser) parseEnvInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	// ENV requires at least one key-value pair
	args := tokens.GetArgumentsAsString()
	if args == "" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "ENV instruction requires at least one variable",
			Position: instruction.Range.Start,
		}
	}

	// Parse key-value pairs
	pairs := parseKeyValuePairs(args)
	for k, v := range pairs {
		instruction.Args = append(instruction.Args, k+"="+v)
	}

	return nil
}

// Parse ADD or COPY instruction
func (p *InstructionParser) parseAddCopyInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	args := make([]string, 0)
	hasChown := false
	hasFrom := false
	
	// Process flags
	for _, token := range tokens.Arguments {
		if token.Type == lexer.TOKEN_STRING && strings.HasPrefix(token.Value, "--") {
			if strings.HasPrefix(token.Value, "--chown=") {
				instruction.Flags["chown"] = strings.TrimPrefix(token.Value, "--chown=")
				hasChown = true
			} else if strings.HasPrefix(token.Value, "--from=") {
				instruction.Flags["from"] = strings.TrimPrefix(token.Value, "--from=")
				hasFrom = true
				
				// Track dependency on the referenced stage
				fromValue := instruction.Flags["from"]
				instruction.Dependencies = append(instruction.Dependencies, fromValue)
			} else if strings.HasPrefix(token.Value, "--chmod=") {
				instruction.Flags["chmod"] = strings.TrimPrefix(token.Value, "--chmod=")
			}
		} else if token.Type != lexer.TOKEN_WHITESPACE {
			args = append(args, token.Value)
		}
	}

	// Validate arguments
	if len(args) < 2 {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  instruction.Command + " instruction requires at least source and destination",
			Position: instruction.Range.Start,
		}
	}

	// Last argument is destination, all others are sources
	instruction.Args = args

	// COPY has --from flag, ADD cannot
	if instruction.Command == "ADD" && hasFrom {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "ADD does not support the --from flag",
			Position: instruction.Range.Start,
		}
	}

	return nil
}

// Parse VOLUME instruction
func (p *InstructionParser) parseVolumeInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	if tokens.JSONForm {
		return p.parseJSONArrayForm(tokens, instruction)
	}

	// Process arguments
	for _, token := range tokens.Arguments {
		if token.Type != lexer.TOKEN_WHITESPACE {
			instruction.Args = append(instruction.Args, token.Value)
		}
	}

	if len(instruction.Args) == 0 {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "VOLUME instruction requires at least one argument",
			Position: instruction.Range.Start,
		}
	}

	return nil
}

// Parse USER instruction
func (p *InstructionParser) parseUserInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	// USER requires a username or UID
	args := tokens.GetArgumentsAsString()
	if args == "" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "USER instruction requires a username or UID",
			Position: instruction.Range.Start,
		}
	}

	instruction.Args = []string{args}
	return nil
}

// Parse WORKDIR instruction
func (p *InstructionParser) parseWorkdirInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	// WORKDIR requires a path
	args := tokens.GetArgumentsAsString()
	if args == "" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "WORKDIR instruction requires a path",
			Position: instruction.Range.Start,
		}
	}

	instruction.Args = []string{args}
	return nil
}

// Parse ARG instruction
func (p *InstructionParser) parseArgInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	// ARG requires a name
	args := tokens.GetArgumentsAsString()
	if args == "" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "ARG instruction requires a name",
			Position: instruction.Range.Start,
		}
	}

	// Parse ARG name[=value]
	parts := strings.SplitN(args, "=", 2)
	name := strings.TrimSpace(parts[0])
	
	instruction.Args = []string{name}
	
	// Add default value if provided
	if len(parts) > 1 {
		instruction.Flags["default"] = parts[1]
	}

	return nil
}

// Parse ONBUILD instruction
func (p *InstructionParser) parseOnbuildInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	// ONBUILD requires another instruction
	args := tokens.GetArgumentsAsString()
	if args == "" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "ONBUILD instruction requires another instruction as argument",
			Position: instruction.Range.Start,
		}
	}

	// First token must be an instruction
	var triggerInstruction string
	for _, token := range tokens.Arguments {
		if token.Type != lexer.TOKEN_WHITESPACE {
			triggerInstruction = token.Value
			break
		}
	}

	// Validate that trigger instruction is not another ONBUILD
	if triggerInstruction == "ONBUILD" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "ONBUILD cannot be nested (ONBUILD ONBUILD ...)",
			Position: instruction.Range.Start,
		}
	}

	// Validate that trigger instruction is not FROM
	if triggerInstruction == "FROM" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "ONBUILD cannot trigger FROM instruction",
			Position: instruction.Range.Start,
		}
	}

	instruction.Args = []string{args}
	return nil
}

// Parse STOPSIGNAL instruction
func (p *InstructionParser) parseStopsignalInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	// STOPSIGNAL requires a signal
	args := tokens.GetArgumentsAsString()
	if args == "" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "STOPSIGNAL instruction requires a signal",
			Position: instruction.Range.Start,
		}
	}

	// Validate signal format (number or signal name)
	signal := strings.TrimSpace(args)
	if _, err := strconv.Atoi(signal); err != nil {
		// Not a number, must be a signal name
		if !strings.HasPrefix(signal, "SIG") {
			return &DockerfileError{
				Code:     CodeInstructionError,
				Message:  "Invalid signal format: " + signal,
				Position: instruction.Range.Start,
			}
		}
	}

	instruction.Args = []string{signal}
	return nil
}

// Parse HEALTHCHECK instruction
func (p *InstructionParser) parseHealthcheckInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	args := make([]string, 0)
	
	// Check for NONE option
	firstArg := ""
	for _, token := range tokens.Arguments {
		if token.Type != lexer.TOKEN_WHITESPACE {
			firstArg = token.Value
			break
		}
	}
	
	if firstArg == "NONE" {
		instruction.Args = []string{"NONE"}
		return nil
	}
	
	// Process flags
	cmdFound := false
	for i := 0; i < len(tokens.Arguments); i++ {
		token := tokens.Arguments[i]
		if token.Type == lexer.TOKEN_WHITESPACE {
			continue
		}
		
		if strings.HasPrefix(token.Value, "--") {
			if i+1 < len(tokens.Arguments) && tokens.Arguments[i+1].Type != lexer.TOKEN_WHITESPACE {
				flagName := strings.TrimPrefix(token.Value, "--")
				flagValue := tokens.Arguments[i+1].Value
				instruction.Flags[flagName] = flagValue
				i++ // Skip the next token (flag value)
			}
		} else if token.Value == "CMD" {
			cmdFound = true
			// Collect the rest as the CMD
			cmdArgs := ""
			for j := i+1; j < len(tokens.Arguments); j++ {
				if tokens.Arguments[j].Type != lexer.TOKEN_WHITESPACE {
					cmdArgs += tokens.Arguments[j].Value + " "
				}
			}
			args = append(args, "CMD", strings.TrimSpace(cmdArgs))
			break
		} else {
			args = append(args, token.Value)
		}
	}
	
	// Validate that CMD is present
	if !cmdFound && firstArg != "NONE" {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "HEALTHCHECK requires CMD or NONE",
			Position: instruction.Range.Start,
		}
	}
	
	instruction.Args = args
	return nil
}

// Parse SHELL instruction
func (p *InstructionParser) parseShellInstruction(tokens *lexer.InstructionTokens, instruction *Instruction) error {
	if !tokens.JSONForm {
		return &DockerfileError{
			Code:     CodeInstructionError,
			Message:  "SHELL instruction requires JSON array format",
			Position: instruction.Range.Start,
		}
	}
	
	return p.parseJSONArrayForm(tokens, instruction)
}

// Helper function to parse key-value pairs from string
func parseKeyValuePairs(input string) map[string]string {
	result := make(map[string]string)
	
	// State variables for parsing
	var key, value string
	inQuote := false
	quoteChar := ' '
	isKey := true
	escaped := false
	
	for _, ch := range input {
		if escaped {
			// Previous character was backslash, add this character as-is
			if isKey {
				key += string(ch)
			} else {
				value += string(ch)
			}
			escaped = false
			continue
		}
		
		if ch == '\\' {
			escaped = true
			continue
		}
		
		switch ch {
		case '"', '\'':
			if inQuote && ch == quoteChar {
				// End of quoted section
				inQuote = false
			} else if !inQuote {
				// Start of quoted section
				inQuote = true
				quoteChar = ch
			} else {
				// Different quote character inside quotes, treat as literal
				if isKey {
					key += string(ch)
				} else {
					value += string(ch)
				}
			}
		case '=':
			if !inQuote && isKey {
				isKey = false
			} else {
				if isKey {
					key += string(ch)
				} else {
					value += string(ch)
				}
			}
		case ' ', '\t':
			if inQuote {
				// Space inside quotes, keep it
				if isKey {
					key += string(ch)
				} else {
					value += string(ch)
				}
			} else if !isKey && value != "" {
				// Space after value, end of this key-value pair
				result[key] = value
				key = ""
				value = ""
				isKey = true
			}
		default:
			if isKey {
				key += string(ch)
			} else {
				value += string(ch)
			}
		}
	}
	
	// Add the last key-value pair if any
	if key != "" {
		result[key] = value
	}
	
	return result
}