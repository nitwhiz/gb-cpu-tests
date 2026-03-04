# gb-cpu-tests

A collection of Game Boy CPU instruction tests provided as YAML specifications and a Go library for loading them.

## Overview

The project provides comprehensive tests for each Game Boy CPU instruction. Each test includes the initial state, the expected final state, and the cycle-accurate bus operations.

- **Specs**: Located in `spec/*.yaml`, named by opcode (e.g., `0b.yaml`).
- **Data**: Includes registers (A, B, C, D, E, H, L, F, PC, SP), RAM values, and bus cycles (address, data, and mode: read/write).
- **Format**: YAML for easy parsing and human readability.

## Go Library

The `spec` package provides a Go API for loading these tests, with the YAML files embedded in the binary.

### Usage

```go
import "github.com/nitwhiz/gb-cpu-tests/spec"

// Load tests for a specific opcode
suite, err := spec.Load(0x0B)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Suite: %s, Tests: %d\n", suite.CanonicalName, len(suite.Tests))

// Iterate over all available test suites
for suite, err := range spec.LoadAll() {
    if err != nil {
        log.Fatal(err)
    }
    // Process suite
}
```

## Data Source

The data in this repository is sourced from [adtennant/GameboyCPUTests](https://github.com/adtennant/GameboyCPUTests.git).

## Structure

- `spec/`: YAML test specifications for each opcode.
- `spec.go`: Go data structures and loading logic.
- `cmd/gen/`: Tools for generating or updating test specifications.
- `data/`: Raw test data sources.
