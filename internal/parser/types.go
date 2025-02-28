package parser

import (
    "time"
    "github.com/docker/docker/builder/dockerfile/parser"
)

// Position represents a position in the Dockerfile
type Position struct {
    Line     int
    Column   int
    Offset   int
    FilePath string
}

// Range represents a range in the source code
type Range struct {
    Start Position
    End   Position
}

// Metadata contains information about the Dockerfile parsing process
type Metadata struct {
    ParseTime   time.Time
    Filename    string
    Size        int64
    BaseImages  []string
    StageCount  int
    Warnings    []Warning
}

// Instruction represents a Dockerfile instruction (CMD, RUN, etc.)
type Instruction struct {
    Command     string            // The instruction type (FROM, RUN, etc.)
    Args        []string          // Arguments for the instruction
    Flags       map[string]string // Instruction-specific flags
    Range       Range             // Position in the source
    Raw         string            // Raw instruction text
    Comment     string            // Associated comments
    JSONForm    bool             // Whether instruction uses JSON form
    Stage       *Stage           // Parent build stage
    Heredoc     *Heredoc         // Heredoc content if present
    Dependencies []string        // Files/resources this instruction depends on
}

// Stage represents a build stage in multi-stage builds
type Stage struct {
    Name         string
    Index        int
    BaseImage    string
    BaseStage    *Stage          // Reference to base stage if using FROM
    Instructions []Instruction
    Range        Range
    Aliases      []string        // Other names for this stage
    Variables    map[string]Variable
    Platform     string          // Target platform for this stage
}

// Heredoc represents a here-document in a Dockerfile
type Heredoc struct {
    Identifier string
    Content    string
    Range      Range
    Delimiter  string
    StripLeadingTabs bool
}

// Variable represents an ARG or ENV instruction's variable
type Variable struct {
    Name      string
    Value     string
    Default   string
    Position  Position
    Stage     *Stage
    Type      VariableType
    Scope     VariableScope
}

// VariableType represents the type of variable (ARG or ENV)
type VariableType int

const (
    ArgType VariableType = iota
    EnvType
)

// VariableScope represents the scope of a variable
type VariableScope int

const (
    GlobalScope VariableScope = iota
    StageScope
    BuildScope
)

// Warning represents a non-fatal issue found during parsing
type Warning struct {
    Level    WarnLevel
    Message  string
    Position Position
    Context  string
}

// WarnLevel indicates warning severity
type WarnLevel int

const (
    WarnLow WarnLevel = iota
    WarnMedium
    WarnHigh
)

// ParsedDockerfile represents the final parsed Dockerfile
type ParsedDockerfile struct {
    Stages       []*Stage
    GlobalArgs   map[string]Variable
    GlobalEnv    map[string]Variable
    Raw          string
    AST          *parser.Node
    Metadata     Metadata
    Errors       []error
    Warnings     []Warning
    EscapeChar   rune            // \ or ` as escape character
    ParseOptions ParseOptions
}

// ParseOptions configures the parser behavior
type ParseOptions struct {
    IncludeComments    bool
    ValidateInstructions bool
    FollowSymlinks      bool
    AllowEnvVarExpansion bool
    DefaultPlatform    string
    BuildContext      string
    TargetStage       string
}

// Parser defines the interface for Dockerfile parsing
type Parser interface {
    Parse(content string) (*ParsedDockerfile, error)
    ParseFile(filepath string) (*ParsedDockerfile, error)
    ParseWithOptions(content string, opts ParseOptions) (*ParsedDockerfile, error)
    Validate() []error
}

// InstructionVisitor interface for walking through instructions
type InstructionVisitor interface {
    Visit(instruction *Instruction) error
    VisitStage(stage *Stage) error
}

// Helpers and utility methods

func (i *Instruction) HasFlag(name string) bool {
    _, exists := i.Flags[name]
    return exists
}

func (i *Instruction) GetFlag(name string) string {
    return i.Flags[name]
}

func (i *Instruction) IsMultiline() bool {
    return len(i.Raw) > 0 && i.Raw[len(i.Raw)-1] == '\\'
}

func (s *Stage) LastInstruction() *Instruction {
    if len(s.Instructions) == 0 {
        return nil
    }
    return &s.Instructions[len(s.Instructions)-1]
}

func (s *Stage) AddInstruction(inst Instruction) {
    inst.Stage = s
    s.Instructions = append(s.Instructions, inst)
}

