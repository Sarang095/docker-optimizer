package main

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

func main() {
    if len(os.Args) != 2 {
        fmt.Println("Usage: opp <path-to-dockerfile>")
        os.Exit(1)
    }

    dockerfilePath := os.Args[1]
    parser := NewDockerfileParser()
    
    instructions, err := parser.ParseFile(dockerfilePath)
    if err != nil {
        fmt.Printf("Error parsing Dockerfile: %v\n", err)
        os.Exit(1)
    }

    // Print parsed instructions for verification
    for _, inst := range instructions {
        fmt.Printf("Command: %s\nArgs: %v\nRaw: %s\n\n", 
            inst.Command, inst.Args, inst.Raw)
    }
}

type Instruction struct {
    Command string
    Args    []string
    Raw     string    // Original instruction line
    LineNum int       // Line number in Dockerfile
}

type DockerfileParser struct {
    currentLine int
    continued   bool
    buffer      strings.Builder
}

func NewDockerfileParser() *DockerfileParser {
    return &DockerfileParser{
        currentLine: 0,
        continued:   false,
    }
}

func (p *DockerfileParser) ParseFile(path string) ([]Instruction, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("failed to open Dockerfile: %w", err)
    }
    defer file.Close()

    var instructions []Instruction
    scanner := bufio.NewScanner(file)

    for scanner.Scan() {
        p.currentLine++
        line := scanner.Text()

        // Skip empty lines and comments
        if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
            continue
        }

        // Handle line continuation
        if strings.HasSuffix(line, "\\") {
            p.buffer.WriteString(strings.TrimSuffix(line, "\\"))
            p.continued = true
            continue
        }

        if p.continued {
            p.buffer.WriteString(line)
            line = p.buffer.String()
            p.buffer.Reset()
            p.continued = false
        }

        instruction, err := p.parseLine(line)
        if err != nil {
            return nil, fmt.Errorf("line %d: %w", p.currentLine, err)
        }
        
        if instruction != nil {
            instructions = append(instructions, *instruction)
        }
    }

    return instructions, nil
}

func (p *DockerfileParser) parseLine(line string) (*Instruction, error) {
    line = strings.TrimSpace(line)
    if line == "" {
        return nil, nil
    }

    parts := splitCommand(line)
    if len(parts) == 0 {
        return nil, fmt.Errorf("invalid instruction format")
    }

    command := strings.ToUpper(parts[0])
    args := parts[1:]

    // Validate command
    if !isValidCommand(command) {
        return nil, fmt.Errorf("unknown command: %s", command)
    }

    return &Instruction{
        Command: command,
        Args:    args,
        Raw:     line,
        LineNum: p.currentLine,
    }, nil
}

func splitCommand(line string) []string {
    var parts []string
    var current strings.Builder
    inQuotes := false
    escaped := false

    for _, char := range line {
        if escaped {
            current.WriteRune(char)
            escaped = false
            continue
        }

        if char == '\\' {
            escaped = true
            continue
        }

        if char == '"' {
            inQuotes = !inQuotes
            continue
        }

        if char == ' ' && !inQuotes {
            if current.Len() > 0 {
                parts = append(parts, current.String())
                current.Reset()
            }
            continue
        }

        current.WriteRune(char)
    }

    if current.Len() > 0 {
        parts = append(parts, current.String())
    }

    return parts
}

func isValidCommand(cmd string) bool {
    validCommands := map[string]bool{
        "FROM":       true,
        "RUN":        true,
        "CMD":        true,
        "LABEL":      true,
        "EXPOSE":     true,
        "ENV":        true,
        "ADD":        true,
        "COPY":       true,
        "ENTRYPOINT": true,
        "VOLUME":     true,
        "USER":       true,
        "WORKDIR":    true,
        "ARG":        true,
        "ONBUILD":    true,
        "STOPSIGNAL": true,
        "HEALTHCHECK": true,
        "SHELL":      true,
    }
    return validCommands[cmd]
}

