# Nixy-Go

Educational Unix-learning game. Go reimplementation of ~/nixy.

## Architecture

Four layers: Simulation (pure Unix) → Game State (quests/achievements) → Character (Nixy dialog) → Interface (TUI/Web).

### Rendering pipeline

```
State (pkg/terminal/state.go)     Content state machine: lines, input, dialog, hint, keyboard
  ↓
Reflow (pkg/terminal/reflow.go)   Wraps lines to current viewport width
  ↓
Frame (pkg/terminal/render.go)    Snapshot of everything needed to render one screen
  ↓
Renderer (interface)              Platform-specific output from a Frame
  ├── ANSIRenderer (ansi.go)      CLI terminal output (box-drawing + ANSI colors)
  └── HTMLRenderer (html.go)      Web output (<pre> + CSS classes)
```

**Both renderers must produce the same logical layout.** Any change to element order, spacing, or structure in one renderer must be mirrored in the other. Update tests for both.

### Entry points

- `cmd/repl/` — CLI via Bubbletea. Handles tea.KeyMsg → process.Datum conversion and tea.WindowSizeMsg for resize.
- `cmd/web/` — WASM via syscall/js. Exports nixyInit/nixyKeystroke/nixyResize to JavaScript.
- Both use `pkg/session/` for shared game/guide/shell initialization.

### Key packages

- `pkg/process/` — Process interface (Stdout/Stderr/Stdin/Next/Kill) and Datum types (Chars, TermCode, Signal)
- `pkg/simulation/` — Binary registry, computer management, process launching
- `pkg/command/shell/` — Shell with builtins (cd, exit, nx), pipeline support, Next() for valid input
- `pkg/guide/` — Wraps process, validates input against Next() before forwarding
- `pkg/game/` — Quest manager, planner (keystroke hints), command tracker, machine registry

## Conventions

- Tests require `-p 1` due to global binary registry
- All dev gestures are Makefile targets: `test`, `fuzz`, `build`, `repl`, `web`, `serve`, `check-wasm`, etc.
- Bubbletea is only imported in `cmd/repl/` — never in `pkg/` (would break WASM compilation)
- `web/nixy.wasm` and `web/wasm_exec.js` are build artifacts (.gitignored)
