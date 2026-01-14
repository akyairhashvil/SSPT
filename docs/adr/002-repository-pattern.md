# 002 - Repository Pattern for Database Access

## Status
Accepted

## Context
The application uses SQLite (optionally SQLCipher) and needs consistent access patterns, error handling, and transaction management across many features.

## Decision
Use a repository-style database layer in `internal/database` to isolate SQL and provide higher-level methods (CRUD and queries) for domain models.

## Consequences
- Callers interact with stable, testable methods instead of raw SQL.
- Database behavior can be refactored without changing UI code.
- Centralized error wrapping improves debugging and observability.
