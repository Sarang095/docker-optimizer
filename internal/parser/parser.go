package parser

import (
    "strings"
)

type Instruction struct {
    Command string
    Args    []string
}

type Dockerfile struct {
    Instructions []Instruction
}

func ParseDockerfile(content string) (*Dockerfile, error) {
    // Basic implementation - you'll want to make this more robust
    lines := strings.Split(content, "\n")
    instructions := make([]Instruction, 0)

    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }

        parts := strings.Fields(line)
        if len(parts) > 0 {
            instructions = append(instructions, Instruction{
                Command: parts[0],
                Args:    parts[1:],
            })
        }
    }

    return &Dockerfile{Instructions: instructions}, nil
}
