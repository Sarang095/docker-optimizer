package optimizer

import (
    "github.com/Sarang095/docker-optimizer/internal/parser"
    "strings"
)

type Optimization struct {
    Name        string
    Description string
    Apply       func([]parser.Instruction) []parser.Instruction
}

func Optimize(doc *parser.Dockerfile) (string, error) {
    optimizations := []Optimization{
        {
            Name:        "Combine RUN Commands",
            Description: "Combines multiple RUN commands into a single command",
            Apply:       combineRunCommands,
        },
        // Add more optimizations here
    }

    instructions := doc.Instructions
    for _, opt := range optimizations {
        instructions = opt.Apply(instructions)
    }

    // Convert back to Dockerfile format
    return formatDockerfile(instructions), nil
}

func combineRunCommands(instructions []parser.Instruction) []parser.Instruction {
    var result []parser.Instruction
    var runCommands []string

    for _, inst := range instructions {
        if inst.Command == "RUN" {
            runCommands = append(runCommands, strings.Join(inst.Args, " "))
            continue
        }

        if len(runCommands) > 0 {
            result = append(result, parser.Instruction{
                Command: "RUN",
                Args:    []string{strings.Join(runCommands, " && ")},
            })
            runCommands = nil
        }
        result = append(result, inst)
    }

    if len(runCommands) > 0 {
        result = append(result, parser.Instruction{
            Command: "RUN",
            Args:    []string{strings.Join(runCommands, " && ")},
        })
    }

    return result
}

func formatDockerfile(instructions []parser.Instruction) string {
    var builder strings.Builder
    
    for _, inst := range instructions {
        builder.WriteString(inst.Command)
        if len(inst.Args) > 0 {
            builder.WriteString(" ")
            builder.WriteString(strings.Join(inst.Args, " "))
        }
        builder.WriteString("\n")
    }

    return builder.String()
}

