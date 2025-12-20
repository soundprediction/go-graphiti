# Go-Predicato Project Instructions

## Overview

The `go-predicato` package is a knowledge graph framework designed for building and querying dynamic knowledge graphs that evolve over time.

## Context and Background

When working on this project, you should:

1. **Update Documentation**: When you make modifications to the source tree, update the `PYTHON_TO_GO_MAPPING.md` file accordingly to reflect the current implementation status.

### Technical Considerations

- **Type Safety**: Leverage Go's type system to provide compile-time safety.
- **Concurrency**: Use Go's goroutines and channels effectively.
- **Error Handling**: Use Go's explicit error handling rather than exceptions, providing clear error messages and proper error wrapping.
- **Resource Management**: Properly implement cleanup and resource management using defer statements and Close() methods.
- **Testing**: Maintain comprehensive tests.

### Project Structure

Follow the existing project structure and conventions:
- Core functionality in the root package
- Supporting packages in `pkg/` subdirectories
- Maintain separation of concerns between drivers, search, LLM clients, etc.
- Use interfaces to enable dependency injection and testing

### Development Workflow

1. **Before Implementation**:
   - Understand the method's purpose and behavior
   - Check dependencies and related methods
   - Plan the Go implementation approach

2. **During Implementation**:
   - Implement full functionality, not placeholders
   - Add appropriate error handling
   - Include proper documentation
   - Consider Go-specific optimizations

3. **After Implementation**:
   - Run tests to ensure functionality
   - Verify integration with existing code

### Quality Standards

- Code should compile without warnings
- All public functions should have proper documentation
- Error messages should be clear and actionable
- Performance should be reasonable for the intended use cases
- Memory usage should be efficient and not leak resources

