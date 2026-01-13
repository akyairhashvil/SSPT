# SSPT Architecture

## Package Structure

```
cmd/
  app/              # Application entry point
    main.go         # CLI initialization, passphrase handling

internal/
  config/           # Application constants and configuration
    constants.go    # Timer durations, display settings

  database/         # SQLite persistence layer
    db.go           # Database connection, migrations, encryption
    goal.go         # Goal helpers
    goal_*.go       # Goal CRUD operations, tags, dependencies
    sprint.go       # Sprint CRUD operations
    export.go       # JSON export/import
    errors.go       # Custom error types

  models/           # Domain types
    models.go       # Workspace, Day, Sprint, Goal structs

  tui/              # Terminal UI (Bubble Tea)
    dashboard.go    # Main model and initialization
    update*.go      # Input handling
    render*.go      # View rendering
    theme.go        # Styling definitions

  util/             # Shared utilities
    logging.go      # Error logging helpers
    paths.go        # File system paths
    tags.go         # Tag parsing
```

## Data Flow

1. User input -> `tui.Update()` -> Command
2. Command -> `database.*()` -> SQL
3. SQL result -> `models.*` -> State update
4. State -> `tui.View()` -> Terminal output

## Key Design Decisions

- Elm Architecture: TUI follows Model-View-Update pattern
- Single Database: All data in one SQLite file
- Optional Encryption: SQLCipher support via build flag
- Local Storage: No network, full data sovereignty
