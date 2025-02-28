package lexer

import (
    "bufio"
    "bytes"
    "io"
    "strings"
    "unicode"
    
    "github.com/yourusername/dockerfile-parser/internal/parser"
)

// Scanner represents a lexical scanner for Dockerfile syntax
type Scanner struct {
    reader      *bufio.Reader
    position    parser.Position
    char        rune
    buffer      bytes.Buffer
    peekBuffer  bytes.Buffer
    inHeredoc   bool
    heredocWord string
    errorHandler *parser.ErrorHandler
    // New fields for enhanced scanning
    lastToken    *Token
    stageDepth   int
    variables    map[string]bool
}

func NewScanner(r io.Reader) *Scanner {
    return &Scanner{
        reader:       bufio.NewReader(r),
        position:     parser.Position{Line: 1, Column: 0},
        errorHandler: parser.NewErrorHandler(),
        variables:    make(map[string]bool),
    }
}

// Core scanning methods from previous implementation...

// Enhanced scanning methods:

func (s *Scanner) scanHeredocContent() (*Token, error) {
    s.buffer.Reset()
    startPos := s.position
    
    for {
        if err := s.scan(); err != nil {
            return nil, err
        }
        
        // Check for heredoc end
        if s.char == '\n' {
            nextLine, err := s.peekLine()
            if err != nil {
                return nil, err
            }
            if strings.TrimSpace(nextLine) == s.heredocWord {
                s.inHeredoc = false
                // Consume the heredoc word
                for i := 0; i < len(s.heredocWord)+1; i++ {
                    s.scan()
                }
                break
            }
        }
        
        s.buffer.WriteRune(s.char)
    }
    
    return &Token{
        Type:     TOKEN_HEREDOC_CONTENT,
        Value:    s.buffer.String(),
        Line:     startPos.Line,
        Column:   startPos.Column,
        Length:   s.buffer.Len(),
        Raw:      s.buffer.String(),
    }, nil
}

func (s *Scanner) scanJSONArray() (*Token, error) {
    s.buffer.Reset()
    startPos := s.position
    depth := 1
    s.buffer.WriteRune(s.char) // Write initial '['
    
    for {
        if err := s.scan(); err != nil {
            return nil, err
        }
        
        switch s.char {
        case '[':
            depth++
        case ']':
            depth--
        case '"', '\'':
            if err := s.scanQuotedContentInJSON(s.char); err != nil {
                return nil, err
            }
            continue
        }
        
        s.buffer.WriteRune(s.char)
        
        if depth == 0 {
            break
        }
    }
    
    return &Token{
        Type:     TOKEN_STRING,
        Value:    s.buffer.String(),
        Line:     startPos.Line,
        Column:   startPos.Column,
        Length:   s.buffer.Len(),
        Raw:      s.buffer.String(),
    }, nil
}

func (s *Scanner) scanQuotedContentInJSON(quote rune) error {
    s.buffer.WriteRune(quote)
    
    for {
        if err := s.scan(); err != nil {
            return err
        }
        
        if s.char == '\\' {
            s.buffer.WriteRune(s.char)
            if err := s.scan(); err != nil {
                return err
            }
            s.buffer.WriteRune(s.char)
            continue
        }
        
        s.buffer.WriteRune(s.char)
        
        if s.char == quote {
            break
        }
    }
    
    return nil
}

func (s *Scanner) scanVariable() (*Token, error) {
    s.buffer.Reset()
    startPos := s.position
    s.buffer.WriteRune(s.char) // Write '$'
    
    if err := s.scan(); err != nil {
        return nil, err
    }
    
    // Handle ${VAR} syntax
    if s.char == '{' {
        s.buffer.WriteRune(s.char)
        for {
            if err := s.scan(); err != nil {
                return nil, err
            }
            if s.char == '}' {
                s.buffer.WriteRune(s.char)
                break
            }
            if !isValidVariableChar(s.char) {
                return nil, s.errorHandler.HandleError(&parser.DockerfileError{
                    Code:     parser.CodeSyntaxError,
                    Position: s.position,
                    Message:  "Invalid character in variable name",
                    Snippet:  s.buffer.String(),
                })
            }
            s.buffer.WriteRune(s.char)
        }
    } else {
        // Handle $VAR syntax
        for isValidVariableChar(s.char) {
            s.buffer.WriteRune(s.char)
            if err := s.scan(); err != nil {
                break
            }
        }
    }
    
    varName := s.buffer.String()
    s.variables[varName] = true
    
    return &Token{
        Type:     TOKEN_VARIABLE,
        Value:    varName,
        Line:     startPos.Line,
        Column:   startPos.Column,
        Length:   len(varName),
        Raw:      varName,
    }, nil
}

func (s *Scanner) scanContinuation() (*Token, error) {
    startPos := s.position
    
    // Consume the backslash
    if err := s.scan(); err != nil {
        return nil, err
    }
    
    // Must be followed by newline
    if s.char != '\n' {
        return nil, s.errorHandler.HandleError(&parser.DockerfileError{
            Code:     parser.CodeSyntaxError,
            Position: startPos,
            Message:  "Line continuation character must be followed by newline",
            Snippet:  "\\",
        })
    }
    
    return &Token{
        Type:     TOKEN_CONTINUATION,
        Value:    "\\",
        Line:     startPos.Line,
        Column:   startPos.Column,
        Length:   1,
        Raw:      "\\",
    }, nil
}

// Helper methods

func (s *Scanner) peekLine() (string, error) {
    s.peekBuffer.Reset()
    for {
        ch, _, err := s.reader.ReadRune()
        if err != nil {
            if err == io.EOF {
                break
            }
            return "", err
        }
        
        if ch == '\n' {
            s.reader.UnreadRune()
            break
        }
        
        s.peekBuffer.WriteRune(ch)
        s.reader.UnreadRune()
    }
    return s.peekBuffer.String(), nil
}

func isValidVariableChar(ch rune) bool {
    return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}

// Enhanced metadata collection for ML
func (s *Scanner) GetTokenMetadata(token *Token) TokenMetadata {
    metadata := TokenMetadata{
        IsKeyword:  token.IsInstruction(),
        IsOptional: isOptionalInstruction(token.Type),
        Category:   getCategoryForToken(token.Type),
    }
    
    if token.IsInstruction() {
        metadata.Impact = getInstructionImpact(token.Type)
    }
    
    // Add variable tracking
    if token.Type == TOKEN_VARIABLE {
        metadata.Impact.CacheBreaking = true
    }
    
    return metadata
}

func isOptionalInstruction(tokenType TokenType) bool {
    optionalInstructions := map[TokenType]bool{
        TOKEN_INSTRUCTION_LABEL:       true,
        TOKEN_INSTRUCTION_MAINTAINER:  true,
        TOKEN_INSTRUCTION_HEALTHCHECK: true,
        TOKEN_INSTRUCTION_SHELL:       true,
    }
    return optionalInstructions[tokenType]
}
