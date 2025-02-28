go 1.21

require (
    // Docker official packages
    github.com/docker/docker v24.0.7+incompatible
    github.com/docker/distribution v2.8.2+incompatible
    github.com/docker/go-connections v0.4.0
    
    // Error handling and utilities
    github.com/pkg/errors v0.9.1
    github.com/sirupsen/logrus v1.9.3
    
    // Testing utilities
    github.com/stretchr/testify v1.8.4

    // Context and sync utilities
    golang.org/x/sync v0.5.0
    golang.org/x/sys v0.15.0
    
    // JSON and YAML handling
    gopkg.in/yaml.v3 v3.0.1
)

// Indirect dependencies
require (
    github.com/davecgh/go-spew v1.1.1 // indirect
    github.com/pmezard/go-difflib v1.0.0 // indirect
    github.com/moby/term v0.5.0 // indirect
    github.com/morikuni/aec v1.0.0 // indirect
    golang.org/x/time v0.5.0 // indirect
    gotest.tools/v3 v3.5.1 // indirect
)

// Replace directives if needed
replace (
    github.com/docker/docker => github.com/docker/docker v24.0.7+incompatible
)
