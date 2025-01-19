package main

import (
    "flag"
    "fmt"
    "log"
    "os"

    "github.com/Sarang095/docker-optimizer/internal/optimizer"
    "github.com/Sarang095/docker-optimizer/internal/parser"
)

func main() {
    dockerfilePath := flag.String("dockerfile", "Dockerfile", "Path to the Dockerfile")
    outputPath := flag.String("output", "Dockerfile.optimized", "Path for the optimized Dockerfile")
    flag.Parse()

    if err := run(*dockerfilePath, *outputPath); err != nil {
        log.Fatal(err)
    }
}

func run(dockerfilePath, outputPath string) error {
    // Read the Dockerfile
    content, err := os.ReadFile(dockerfilePath)
    if err != nil {
        return fmt.Errorf("failed to read Dockerfile: %w", err)
    }

    // Parse the Dockerfile
    parsedDoc, err := parser.ParseDockerfile(string(content))
    if err != nil {
        return fmt.Errorf("failed to parse Dockerfile: %w", err)
    }

    // Optimize the Dockerfile
    optimizedDoc, err := optimizer.Optimize(parsedDoc)
    if err != nil {
        return fmt.Errorf("failed to optimize Dockerfile: %w", err)
    }

    // Write the optimized Dockerfile
    if err := os.WriteFile(outputPath, []byte(optimizedDoc), 0644); err != nil {
        return fmt.Errorf("failed to write optimized Dockerfile: %w", err)
    }

    fmt.Printf("Successfully optimized Dockerfile and saved to %s\n", outputPath)
    return nil
}

