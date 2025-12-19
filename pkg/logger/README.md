# Logger Package

This package provides colored logging support for go-predicato using Go's standard `log/slog` library.

## Features

- **Red colored error messages** - Errors are displayed in red for easy identification
- **Yellow colored warning messages** - Warnings are displayed in yellow to distinguish from errors
- **Green colored persist messages** - Info messages containing "persist" are displayed in green to highlight database operations
- **Standard output for info/debug** - Other info and debug messages use standard formatting
- **Drop-in replacement** - Compatible with standard `slog.Logger`

## Usage

### Quick Start

```go
import (
    "log/slog"
    "github.com/soundprediction/go-predicato/pkg/logger"
)

// Create a logger with default settings (stderr, Info level)
log := logger.NewDefaultLogger(slog.LevelInfo)

// Use like any slog.Logger
log.Info("Application started")              // Standard output
log.Info("Persisting nodes to database")     // Green colored output
log.Warn("Low disk space")                   // Yellow colored output
log.Error("Failed to connect to DB")         // Red colored output
```

### Custom Configuration

```go
import (
    "os"
    "log/slog"
    "github.com/soundprediction/go-predicato/pkg/logger"
)

// Create a logger with custom writer and options
log := slog.New(logger.NewColorHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
    AddSource: true,
}))

log.Debug("Debugging info")
log.Info("Normal operation")
log.Warn("Warning message")
log.Error("Error occurred", "error", err, "context", "database")
```

### Integration with Predicato

The colored logger is automatically used in go-predicato commands and examples:

```go
import (
    "log/slog"
    "github.com/soundprediction/go-predicato"
    predicatoLogger "github.com/soundprediction/go-predicato/pkg/logger"
)

logger := predicatoLogger.NewDefaultLogger(slog.LevelInfo)
client := predicato.NewClient(driver, llmClient, embedderClient, config, logger)
```

## API Reference

### `NewDefaultLogger(level slog.Level) *slog.Logger`
Creates a logger with color support writing to stderr with the specified log level.

### `NewLogger(w io.Writer, level slog.Level) *slog.Logger`
Creates a logger with color support writing to the specified writer with the specified log level.

### `NewColorHandler(w io.Writer, opts *slog.HandlerOptions) *ColorHandler`
Creates a color handler that wraps slog's text handler with color support.

## Color Codes

- **Errors**: Red (`\033[31m`)
- **Warnings**: Yellow (`\033[33m`)
- **Persist operations**: Green (`\033[32m`) - Applied to info messages containing "persist"
- **Other Info/Debug**: No color (standard terminal output)

## Terminal Compatibility

ANSI color codes work on:
- macOS Terminal
- Linux terminals (bash, zsh, etc.)
- Windows Terminal (Windows 10+)
- Most modern terminal emulators

If your terminal doesn't support ANSI codes, the escape sequences will be visible but won't affect functionality.

## Testing

Run the tests with:
```bash
go test ./pkg/logger/...
```

## Implementation Details

The color handler wraps Go's standard `slog.TextHandler` and modifies log messages by prepending ANSI color codes based on the log level. It properly implements the `slog.Handler` interface, including support for attributes and groups.
