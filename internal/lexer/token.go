package lexer

import (
    "fmt"
    "strings"
)

// TokenType represents different types of tokens in a Dockerfile
type TokenType int

// Token represents a lexical unit in the Dockerfile
type Token struct {
    Type    TokenType // Type of the token
    Value   string    // Actual text value
    Line    int       // Line number in source
    Column  int       // Column position
    Length  int       // Length of the token
    Raw     string    // Raw token text before processing
}

// String provides a human-readable representation of a token
func (t Token) String() string {
    if t.Value != "" {
        return fmt.Sprintf("%s(%s) at line %d:%d", t.Type, t.Value, t.Line, t.Column)
    }
    return fmt.Sprintf("%s at line %d:%d", t.Type, t.Line, t.Column)
}

// Define all possible token types in a Dockerfile
const (
    TOKEN_ILLEGAL TokenType = iota // Invalid or unknown token
    TOKEN_EOF                      // End of file
    TOKEN_NEWLINE                  // New line
    TOKEN_WHITESPACE              // Space or tab
    
    // Instruction tokens
    TOKEN_INSTRUCTION_FROM
    TOKEN_INSTRUCTION_RUN
    TOKEN_INSTRUCTION_CMD
    TOKEN_INSTRUCTION_LABEL
    TOKEN_INSTRUCTION_MAINTAINER
    TOKEN_INSTRUCTION_EXPOSE
    TOKEN_INSTRUCTION_ENV
    TOKEN_INSTRUCTION_ADD
    TOKEN_INSTRUCTION_COPY
    TOKEN_INSTRUCTION_ENTRYPOINT
    TOKEN_INSTRUCTION_VOLUME
    TOKEN_INSTRUCTION_USER
    TOKEN_INSTRUCTION_WORKDIR
    TOKEN_INSTRUCTION_ARG
    TOKEN_INSTRUCTION_ONBUILD
    TOKEN_INSTRUCTION_STOPSIGNAL
    TOKEN_INSTRUCTION_HEALTHCHECK
    TOKEN_INSTRUCTION_SHELL
    
    // Special syntax tokens
    TOKEN_COMMENT           // Comments starting with #
    TOKEN_CONTINUATION     // Line continuation (\)
    TOKEN_ESCAPEDCHAR     // Escaped character
    TOKEN_HEREDOC_START   // Here-document start (<<)
    TOKEN_HEREDOC_END     // Here-document end
    
    // Argument tokens
    TOKEN_STRING          // Regular string
    TOKEN_QUOTED_STRING   // Quoted string "..." or '...'
    TOKEN_NUMBER          // Numeric value
    TOKEN_EQUALS          // = sign
    TOKEN_COLON          // : character
    TOKEN_COMMA          // , character
    TOKEN_LBRACKET       // [
    TOKEN_RBRACKET       // ]
    TOKEN_VARIABLE       // $VAR or ${VAR}
    
    // Multi-stage build tokens
    TOKEN_AS             // AS keyword in FROM
    TOKEN_STAGE_NAME     // Stage name
)

// Keywords maps instruction names to their token types
var Keywords = map[string]TokenType{
    "FROM":         TOKEN_INSTRUCTION_FROM,
    "RUN":          TOKEN_INSTRUCTION_RUN,
    "CMD":          TOKEN_INSTRUCTION_CMD,
    "LABEL":        TOKEN_INSTRUCTION_LABEL,
    "MAINTAINER":   TOKEN_INSTRUCTION_MAINTAINER,
    "EXPOSE":       TOKEN_INSTRUCTION_EXPOSE,
    "ENV":          TOKEN_INSTRUCTION_ENV,
    "ADD":          TOKEN_INSTRUCTION_ADD,
    "COPY":         TOKEN_INSTRUCTION_COPY,
    "ENTRYPOINT":   TOKEN_INSTRUCTION_ENTRYPOINT,
    "VOLUME":       TOKEN_INSTRUCTION_VOLUME,
    "USER":         TOKEN_INSTRUCTION_USER,
    "WORKDIR":      TOKEN_INSTRUCTION_WORKDIR,
    "ARG":          TOKEN_INSTRUCTION_ARG,
    "ONBUILD":      TOKEN_INSTRUCTION_ONBUILD,
    "STOPSIGNAL":   TOKEN_INSTRUCTION_STOPSIGNAL,
    "HEALTHCHECK":  TOKEN_INSTRUCTION_HEALTHCHECK,
    "SHELL":        TOKEN_INSTRUCTION_SHELL,
    "AS":           TOKEN_AS,
}

// TokenTypeStrings provides string representations of token types
var TokenTypeStrings = map[TokenType]string{
    TOKEN_ILLEGAL:     "ILLEGAL",
    TOKEN_EOF:         "EOF",
    TOKEN_NEWLINE:     "NEWLINE",
    TOKEN_WHITESPACE:  "WHITESPACE",
    // ... other token types ...
}

// Helper functions for token analysis

// IsInstruction checks if a token is an instruction
func (t Token) IsInstruction() bool {
    return t.Type >= TOKEN_INSTRUCTION_FROM && t.Type <= TOKEN_INSTRUCTION_SHELL
}

// IsArgument checks if a token can be an instruction argument
func (t Token) IsArgument() bool {
    return t.Type == TOKEN_STRING || 
           t.Type == TOKEN_QUOTED_STRING || 
           t.Type == TOKEN_NUMBER ||
           t.Type == TOKEN_VARIABLE
}

// TokenMetadata provides additional analysis data for ML processing
type TokenMetadata struct {
    IsKeyword     bool
    IsOptional    bool
    Category      string
    Impact        TokenImpact
}

// TokenImpact represents the potential impact of a token on build optimization
type TokenImpact struct {
    LayerCreating bool    // Whether this token typically creates a new layer
    CacheBreaking bool    // Whether this token typically breaks build cache
    SizeImpact    int     // Estimated impact on image size (1-10)
}

// GetTokenMetadata provides metadata for ML analysis
func (t Token) GetMetadata() TokenMetadata {
    metadata := TokenMetadata{
        IsKeyword: t.IsInstruction(),
        Category:  getCategoryForToken(t.Type),
    }
    
    // Set impact information for ML analysis
    if t.IsInstruction() {
        metadata.Impact = getInstructionImpact(t.Type)
    }
    
    return metadata
}

// Helper function to categorize tokens for ML analysis
func getCategoryForToken(typ TokenType) string {
    switch {
    case typ >= TOKEN_INSTRUCTION_FROM && typ <= TOKEN_INSTRUCTION_SHELL:
        return "instruction"
    case typ == TOKEN_VARIABLE:
        return "variable"
    case typ == TOKEN_COMMENT:
        return "metadata"
    default:
        return "syntax"
    }
}

// Analyze instruction impact for ML optimization
func getInstructionImpact(typ TokenType) TokenImpact {
    impacts := map[TokenType]TokenImpact{
        TOKEN_INSTRUCTION_FROM: {
            LayerCreating: true,
            CacheBreaking: true,
            SizeImpact:    10,
        },
        TOKEN_INSTRUCTION_RUN: {
            LayerCreating: true,
            CacheBreaking: true,
            SizeImpact:    8,
        },
        TOKEN_INSTRUCTION_COPY: {
            LayerCreating: true,
            CacheBreaking: true,
            SizeImpact:    7,
        },
        // Add other instructions...
    }
    
    if impact, exists := impacts[typ]; exists {
        return impact
    }
    
    return TokenImpact{} // Default impact
}
