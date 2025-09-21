# Go-Graphiti Project Instructions

## Overview

The `go-graphiti` package is a Go port of the Python [Graphiti](https://github.com/getzep/graphiti) package, which is a temporally-aware knowledge graph framework designed for building and querying dynamic knowledge graphs that evolve over time.

## Context and Background

When working on this project, you should:

1. **Read the Python Context**: Always start by reading `graphiti/CLAUDE.md` from the original Python package to understand the framework's purpose, architecture, and design principles.

2. **Understand the Port Status**: Read `go-graphiti/docs/PYTHON_TO_GO_MAPPING.md` to understand the current status of the port. This document tracks which Python methods have been implemented in Go and which are still missing or incomplete.

3. **Update Documentation**: When you make modifications to the source tree, update the `PYTHON_TO_GO_MAPPING.md` file accordingly to reflect the current implementation status.

## Implementation Guidelines

### Core Principles

1. **Replicate Structure and Functionality**: Try to replicate the structure and functionality of the original Python code as much as possible. The goal is functional parity with the Python implementation.

2. **No Placeholder Functions**: Do not use placeholder functions or stub implementations. Instead, implement the actual and exact methods found in the Python code with full functionality.

3. **Mimic Method Signatures**: Mimic the method signatures as much as reasonably possible, adapting only when it doesn't make sense due to differences between Python and Go:
   - Convert Python's duck typing to Go's explicit interfaces
   - Use Go's error handling patterns instead of Python exceptions
   - Adapt Python's dynamic typing to Go's static typing
   - Use Go's context.Context for cancellation and timeouts
   - Convert Python's kwargs to Go structs or option patterns

4. **Maintain API Compatibility**: Ensure that the Go API feels natural to Go developers while preserving the semantic meaning of the Python API.

### Technical Considerations

- **Type Safety**: Leverage Go's type system to provide compile-time safety while maintaining the flexibility of the Python API.
- **Concurrency**: Use Go's goroutines and channels effectively, especially for bulk operations and parallel processing.
- **Error Handling**: Use Go's explicit error handling rather than exceptions, providing clear error messages and proper error wrapping.
- **Resource Management**: Properly implement cleanup and resource management using defer statements and Close() methods.
- **Testing**: Maintain comprehensive tests that match the Python test coverage.

### Project Structure

Follow the existing project structure and conventions:
- Core functionality in the root package
- Supporting packages in `pkg/` subdirectories
- Maintain separation of concerns between drivers, search, LLM clients, etc.
- Use interfaces to enable dependency injection and testing

### Development Workflow

1. **Before Implementation**:
   - Read the corresponding Python code
   - Understand the method's purpose and behavior
   - Check dependencies and related methods
   - Plan the Go implementation approach

2. **During Implementation**:
   - Implement full functionality, not placeholders
   - Add appropriate error handling
   - Include proper documentation
   - Consider Go-specific optimizations

3. **After Implementation**:
   - Update `PYTHON_TO_GO_MAPPING.md`
   - Run tests to ensure functionality
   - Verify integration with existing code
   - Document any deviations from Python behavior

### Quality Standards

- Code should compile without warnings
- All public functions should have proper documentation
- Error messages should be clear and actionable
- Performance should be reasonable for the intended use cases
- Memory usage should be efficient and not leak resources

## Current Status

The port is ongoing. Check `docs/PYTHON_TO_GO_MAPPING.md` for the most current status of what has been implemented and what remains to be done.

Remember: The goal is to create a fully functional Go implementation that Python Graphiti users can migrate to with minimal changes to their application logic, while providing the benefits of Go's performance, type safety, and deployment characteristics.