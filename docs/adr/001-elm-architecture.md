# 001 - Elm-Style Architecture

## Status
Accepted

## Context
SSPT is a terminal UI application with complex, stateful interactions (timers, modals, lists, and key handling). We need a predictable way to manage state changes and side effects.

## Decision
Adopt Bubble Tea's Elm-style Model-View-Update architecture for the TUI.

## Consequences
- State changes are centralized in Update handlers, making behavior easier to reason about.
- Rendering is deterministic based on model state, improving testability.
- Side effects are modeled as commands, reducing ad-hoc goroutines and callbacks.
