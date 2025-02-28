package lexer

import (
	"io"
	"strings"

	"github.com/yourusername/dockerfile-parser/internal/parser"
)

// Lexer represents a lexical analyzer for Dockerfile syntax
type Lexer struct {
	scanner      *Scanner
	currentToken *Token
	peekToken    *Token
	tokens       []*Token
	errors       []error
	position     int
	inHeredoc    bool
	heredocID    string
	lineTokens   []*Token // Tokens in current logical line
}

// NewLexer creates a new lexer for tokenizing Dockerfile content
func NewLexer(r io.Reader) *Lexer {
	scanner := NewScanner(r)
	l := &Lexer{
		scanner:    scanner,
		tokens:     make([]*Token, 0),
		errors:     make([]error, 0),
		lineTokens: make([]*Token, 0),
	}
	// Initialize by reading first two tokens
	l.nextToken()
	l.nextToken()
	return l
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() *Token {
	token := l.currentToken
	l.nextToken()
	return token
}

// PeekToken returns the next token without advancing the lexer
func (l *Lexer) PeekToken() *Token {
	return l.peekToken
}

// nextToken advances the lexer to the next token
func (l *Lexer) nextToken() {
	l.currentToken = l.peekToken
	token, err := l.scanner.Scan()
	
	if err != nil {
		if err != io.EOF {
			l.errors = append(l.errors, err)
		}
		l.peekToken = &Token{Type: TOKEN_EOF}
		return
	}
	
	// Skip whitespace tokens unless in heredoc
	if !l.inHeredoc && token.Type == TOKEN_WHITESPACE {
		token, err = l.scanner.Scan()
		if err != nil {
			if err != io.EOF {
				l.errors = append(l.errors, err)
			}
			l.peekToken = &Token{Type: TOKEN_EOF}
			return
		}
	}
	
	// Handle heredoc start and end
	if token.Type == TOKEN_HEREDOC_START {
		l.inHeredoc = true
		l.heredocID = token.Value
	} else if l.inHeredoc && token.Type == TOKEN_HEREDOC_END && token.Value == l.heredocID {
		l.inHeredoc = false
	}
	
	l.tokens = append(l.tokens, token)
	l.peekToken = token
}

// GetTokens returns all tokens processed so far
func (l *Lexer) GetTokens() []*Token {
	return l.tokens
}

// GetErrors returns all errors encountered during lexing
func (l *Lexer) GetErrors() []error {
	return l.errors
}

// TokenizeAll processes all tokens from the input
func (l *Lexer) TokenizeAll() ([]*Token, []error) {
	for l.currentToken.Type != TOKEN_EOF {
		l.NextToken()
	}
	return l.tokens, l.errors
}

// TokenizeLine tokenizes a single logical line (handling continuations)
func (l *Lexer) TokenizeLine() ([]*Token, error) {
	l.lineTokens = make([]*Token, 0)
	continuationMode := false
	
	for {
		token := l.NextToken()
		
		// End of file
		if token.Type == TOKEN_EOF {
			break
		}
		
		// Add token to current line
		l.lineTokens = append(l.lineTokens, token)
		
		// Handle line continuation
		if token.Type == TOKEN_CONTINUATION {
			continuationMode = true
			continue
		}
		
		// End of line
		if token.Type == TOKEN_NEWLINE {
			if !continuationMode {
				break
			}
			continuationMode = false
		}
	}
	
	return l.lineTokens, nil
}

// ProcessInstructionLine processes a line containing a Dockerfile instruction
func (l *Lexer) ProcessInstructionLine() (*InstructionTokens, error) {
	tokens, err := l.TokenizeLine()
	if err != nil {
		return nil, err
	}
	
	// Empty line or comment-only line
	if len(tokens) == 0 || tokens[0].Type == TOKEN_COMMENT {
		return nil, nil
	}
	
	// Check if first token is an instruction
	if !tokens[0].IsInstruction() {
		return nil, &parser.DockerfileError{
			Code:    parser.CodeSyntaxError,
			Message: "Line must start with an instruction",
			Position: parser.Position{
				Line:   tokens[0].Line,
				Column: tokens[0].Column,
			},
			Snippet: tokens[0].Raw,
		}
	}
	
	// Extract instruction and its arguments
	instruction := tokens[0]
	args := make([]*Token, 0)
	comments := make([]*Token, 0)
	
	for i := 1; i < len(tokens); i++ {
		if tokens[i].Type == TOKEN_COMMENT {
			comments = append(comments, tokens[i])
		} else if tokens[i].Type != TOKEN_NEWLINE && tokens[i].Type != TOKEN_CONTINUATION {
			args = append(args, tokens[i])
		}
	}
	
	return &InstructionTokens{
		Instruction: instruction,
		Arguments:   args,
		Comments:    comments,
		Raw:         tokens,
	}, nil
}

// InstructionTokens represents a parsed Dockerfile instruction and its tokens
type InstructionTokens struct {
	Instruction *Token    // The instruction token
	Arguments   []*Token  // Argument tokens
	Comments    []*Token  // Comment tokens
	Raw         []*Token  // All tokens in the instruction line
	JSONForm    bool      // Whether the instruction uses JSON form
}

// IsJSONForm checks if the instruction uses JSON array form
func (l *Lexer) IsJSONForm(tokens []*Token) bool {
	// Start looking after instruction token
	for i := 1; i < len(tokens); i++ {
		if tokens[i].Type == TOKEN_WHITESPACE {
			continue
		}
		
		// If first non-whitespace token after instruction is '['
		if tokens[i].Value == "[" {
			return true
		}
		
		return false
	}
	
	return false
}

// GetInstructionValue retrieves the string value of the instruction
func (it *InstructionTokens) GetInstructionValue() string {
	if it.Instruction == nil {
		return ""
	}
	return it.Instruction.Value
}

// GetArgumentsAsString converts argument tokens to a single string
func (it *InstructionTokens) GetArgumentsAsString() string {
	args := make([]string, 0, len(it.Arguments))
	
	for _, arg := range it.Arguments {
		// Skip whitespace
		if arg.Type == TOKEN_WHITESPACE {
			continue
		}
		args = append(args, arg.Value)
	}
	
	return strings.Join(args, " ")
}

// ProcessAllInstructions tokenizes all instructions in the Dockerfile
func (l *Lexer) ProcessAllInstructions() ([]*InstructionTokens, []error) {
	instructions := make([]*InstructionTokens, 0)
	
	for l.currentToken.Type != TOKEN_EOF {
		inst, err := l.ProcessInstructionLine()
		
		if err != nil {
			l.errors = append(l.errors, err)
			// Skip to next line
			for l.currentToken.Type != TOKEN_EOF && l.currentToken.Type != TOKEN_NEWLINE {
				l.NextToken()
			}
			continue
		}
		
		// Skip empty lines or comment-only lines
		if inst != nil {
			instructions = append(instructions, inst)
		}
	}
	
	return instructions, l.errors
}

// DetectStages analyzes tokens to identify build stages
func (l *Lexer) DetectStages() ([]StageInfo, error) {
	stages := make([]StageInfo, 0)
	currentStage := StageInfo{
		Index: 0,
	}
	
	instructions, errors := l.ProcessAllInstructions()
	if len(errors) > 0 {
		// Return first error for simplicity
		return nil, errors[0]
	}
	
	stageIndex := 0
	
	for _, inst := range instructions {
		// Every FROM instruction begins a new stage
		if inst.GetInstructionValue() == "FROM" {
			// Save previous stage if not first FROM
			if currentStage.StartLine > 0 {
				currentStage.EndLine = inst.Instruction.Line - 1
				stages = append(stages, currentStage)
			}
			
			// Start a new stage
			stageName := ""
			baseImage := ""
			
			// Extract stage name if present (FROM base AS name)
			for i := 0; i < len(inst.Arguments); i++ {
				if inst.Arguments[i].Type == TOKEN_AS && i+1 < len(inst.Arguments) {
					stageName = inst.Arguments[i+1].Value
					break
				}
			}
			
			// Extract base image
			if len(inst.Arguments) > 0 {
				baseImage = inst.Arguments[0].Value
			}
			
			currentStage = StageInfo{
				Index:     stageIndex,
				Name:      stageName,
				BaseImage: baseImage,
				StartLine: inst.Instruction.Line,
			}
			
			stageIndex++
		}
	}
	
	// Add the last stage
	if currentStage.StartLine > 0 {
		// EOF is the end of last stage
		if len(instructions) > 0 {
			lastInst := instructions[len(instructions)-1]
			currentStage.EndLine = lastInst.Instruction.Line
		}
		stages = append(stages, currentStage)
	}
	
	return stages, nil
}

// StageInfo contains information about a build stage
type StageInfo struct {
	Index     int    // Stage index (0-based)
	Name      string // Stage name (might be empty)
	BaseImage string // Base image or stage name
	StartLine int    // Line where stage begins
	EndLine   int    // Line where stage ends
}

// Variable tracking helper
type VariableInfo struct {
	Name     string
	Type     string // "ARG" or "ENV"
	Value    string
	Line     int
	Column   int
	StageIdx int
}

// DetectVariables analyzes tokens to identify variable declarations
func (l *Lexer) DetectVariables() []VariableInfo {
	variables := make([]VariableInfo, 0)
	stages, err := l.DetectStages()
	
	if err != nil {
		return variables
	}
	
	instructions, _ := l.ProcessAllInstructions()
	
	for _, inst := range instructions {
		// Only process ARG and ENV instructions
		instType := inst.GetInstructionValue()
		if instType != "ARG" && instType != "ENV" {
			continue
		}
		
		// Find stage index for this instruction
		stageIdx := -1
		for i, stage := range stages {
			if inst.Instruction.Line >= stage.StartLine && 
				(stage.EndLine == 0 || inst.Instruction.Line <= stage.EndLine) {
				stageIdx = i
				break
			}
		}
		
		// Parse variable declarations from args
		args := inst.GetArgumentsAsString()
		declarations := parseVariableDeclarations(args)
		
		for name, value := range declarations {
			variables = append(variables, VariableInfo{
				Name:     name,
				Type:     instType,
				Value:    value,
				Line:     inst.Instruction.Line,
				Column:   inst.Instruction.Column,
				StageIdx: stageIdx,
			})
		}
	}
	
	return variables
}

// Helper function to parse variable declarations from ARG/ENV arguments
func parseVariableDeclarations(args string) map[string]string {
	result := make(map[string]string)
	
	// Handle multiple declarations (space or equals separated)
	parts := strings.Split(args, " ")
	
	for _, part := range parts {
		if part == "" {
			continue
		}
		
		// Check for KEY=VALUE format
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			result[kv[0]] = kv[1]
		} else {
			// ARG without value
			result[part] = ""
		}
	}
	
	return result
}