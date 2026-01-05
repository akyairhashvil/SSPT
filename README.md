# SSPT (Simple Sprint Productivity Tool)

**The Terminal-Based Cognitive Operating System.**

SSPT is a command-line interface (CLI) tool engineered to enforce the **90/30 Cycle** (90 minutes of Flow, 30 minutes of Recovery). It prioritizes strict temporal bounding, robust persistence, and contextual focus to help developers and power users maintain momentum without the cognitive overhead of complex GUI project management tools.

## Core Philosophy

*   **Temporal Bounding:** Work is divided into strict "Sprints".
*   **Contextual Integrity:** Every task, subtask, and journal entry is strictly linked to time (Day/Sprint) and scope (Workspace).
*   **Keyboard Centric:** Designed for the terminal, optimizing for speed and muscle memory.
*   **Data Sovereignty:** All data is stored locally in a robust SQLite database.

## Architecture

The project is built in **Go** and utilizes the following core technologies:
*   **[Bubble Tea](https://github.com/charmbracelet/bubbletea):** For the robust Model-View-Update (ELM architecture) TUI framework.
*   **[Lip Gloss](https://github.com/charmbracelet/lipgloss):** For style and layout definitions.
*   **SQLite:** For relational data persistence and complex querying capabilities.

## Getting Started

### Prerequisites
*   Go 1.21+
*   A terminal environment (Linux/macOS recommended).

### Installation
1.  Clone the repository:
    ```bash
    git clone https://github.com/akyairhashvil/SSPT.git
    cd SSPT
    ```
2.  Build the binary:
    ```bash
    go build -o sspt cmd/app/main.go
    ```
3.  Run the application:
    ```bash
    ./sspt
    ```

## Contributing

We welcome contributions that align with the core philosophy of "Frictionless Flow". 

### Roadmap
Please consult **`GEMINI.md`** in the root directory. This file serves as the strategic roadmap and "living specification" for the project. All major architectural decisions and phases are documented there.

### Guidelines
1.  **Conventions:** Adhere strictly to the existing project structure and Go coding conventions.
2.  **Database:** If your change requires schema modifications, ensure you update the migration logic in `internal/database/db.go`.
3.  **UI/UX:** Changes to the interface should respect the terminal boundaries and existing keybinding patterns.
4.  **Pull Requests:** Please describe *why* the change is needed, linking back to the philosophy of the tool.

## License

This project is licensed under the terms of the MIT license.
