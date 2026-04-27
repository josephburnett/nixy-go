# Nixy-Go

Educational Unix-learning game. Go reimplementation of ~/nixy.

The ambition: teach real Unix by *doing* it — typing commands, seeing
real output, hitting real consequences — safely enough that a beginner
can explore without getting lost.

## Core principles

Three principles, operating at three layers of the system. Specific
design decisions across the codebase are usually instances of one of
them. When a fix violates one, it's the wrong fix.

### 1. No mistakes (keystroke layer)

At every keystroke, the keyboard's valid set must contain only keys
that lead toward a correct, executable action. The player can never
type a command the shell would then reject (`missing operand`, `no
such file`). Errors that come from invalid typing are bugs in
`shell.Next()`, not runtime concerns.

This is what makes exploration safe. A beginner mashing keys learns,
not panics.

Implementation:
- `simulation.Binary.OptionalArgs` declares whether a command runs
  with zero args (`ls`, `cat`, `pwd` — true; everything else false).
- `Enter` and `|` are valid only at *segment-completion points* —
  exact-match of an optional-args command, or exact-match of a
  complete argument.
- `shell.Next()` is pipe-aware: it splits on `|` and validates only
  the segment after the last pipe, so typing `|` resets to fresh-prompt
  state for the next segment.
- Pinned by `TestShellNextInvariant_*` in `pkg/command/shell/shell_test.go`.

### 2. Real Unix, not a toy (simulation layer)

The mental model the player builds here must transfer to real Unix.
Permissions, ownership, paths, processes, pipes are honest — if
minimal — implementations of the real concepts. When the player learns
`sudo`, they're learning real `sudo`.

Concrete implications (these all flow from this single principle):
- **Identity is real.** The username chosen at login is the player's
  account on every machine. The prompt (`<user>@<host>:<path>>`) is
  ground truth — it matches `whoami`, owns the files the player
  creates, and is what `CanWrite` checks against.
- **Per-user homes.** `MachineRegistry.bootMachine` provisions
  `/home/<username>` on every booted machine. Other users (Nixy) have
  their own homes. World definitions don't ship a placeholder
  `/home/user` — each player gets a real one.
- **Permissions matter.** Don't work around `CanWrite`/`CanRead` to
  make a quest succeed. If the player can't do something, that's a
  teaching opportunity — fix the world (perms, ownership) or the
  quest. The Permissions quest specifically teaches `sudo`; preserve
  the model it depends on. (Today, `/home/nixy` is `Common: Write`
  on purpose — Nixy is loose with perms so early quests work without
  `sudo`. A future quest will tighten that and teach `sudo` as the
  way back in. Don't undo Nixy's loose perms without that quest.)
- **Errors look like Unix errors.** Stderr surfaces normally, exit
  vs. EOF behave correctly, etc.

When choosing how to fix a bug or add a feature, ask: *would this
choice make sense in real Unix?* If it papers over a real concept,
it's the wrong fix.

### 3. Guided progression (game-design layer)

A bare faithful-Unix shell would be useless to a beginner. The game
loop wraps it in scaffolding:

- **Quests** introduce concepts in order, each unlocked by achievements
  granted by the prior one. New quests should build on what's been
  taught, not require leaping forward.
- **The planner** (`game.PlanNextCommand`, `game.PlanHint`) computes
  the next concrete keystroke toward the active quest. This is what
  the green hint key on the keyboard means.
- **Nixy's dialog** (`pkg/character/`) frames each concept narratively,
  before and after. The player isn't memorizing — they're helping
  someone.

Hint vs. validation are two separate systems and often conflated:
- **Validation** (`shell.Next()` → `guide.Stdin`): *what can be typed*.
  Enforces principle #1. Hard rule.
- **Hint** (`PlanHint`): *what the planner suggests typing next*.
  Advisory. When the player goes off-plan, `PlanHint` returns `nil`
  — **not** `TermBackspace`. They may be running a different valid
  command before returning to the plan; the planner re-engages on
  the next empty prompt. Don't nag.

## Architecture

Four layers: Simulation (pure Unix) → Game State (quests/achievements) →
Character (Nixy dialog) → Interface (TUI/Web).

### Rendering pipeline

```
State (pkg/terminal/state.go)     Content state: lines, input, dialog, notice, keyboard
  ↓
Reflow (pkg/terminal/reflow.go)   Wraps lines to current viewport width
  ↓
Layout (pkg/terminal/layout.go)   Composes screen into styled segments
  ↓
Renderer (interface)              Platform-specific output from a Frame
  ├── ANSIRenderer (ansi.go)      CLI output (box-drawing + ANSI colors)
  └── HTMLRenderer (html.go)      Web output (<pre> + CSS classes)
```

**Both renderers must produce the same logical layout.** Layout changes
land in `Layout()`; renderers are token-formatters. Update the test
suite for both.

### Entry points

- `cmd/repl/` — CLI via Bubbletea. Handles `tea.KeyMsg` → `process.Datum`
  and `tea.WindowSizeMsg` for resize. Owns Ctrl+C double-press exit and
  Ctrl+\ snapshot dump.
- `cmd/web/` — WASM via `syscall/js`. Exports `nixyInit`,
  `nixyKeystroke`, `nixyResize`, `nixyDumpState` to JavaScript.
- Both use `pkg/session/` for shared game/guide/shell initialization.

### Key packages

- `pkg/process/` — `Process` interface and Datum types.
- `pkg/simulation/` — Binary registry, computer management, process
  launching. `Binary.OptionalArgs` gates Enter at zero-args.
- `pkg/command/shell/` — Shell with builtins, pipeline support, and
  the pipe-aware `Next()` that enforces principle #1.
- `pkg/guide/` — Wraps a process, validates input against `Next()`
  before forwarding. The bouncer for principle #1.
- `pkg/game/` — Quest manager, planner (keystroke hints), command
  tracker, machine registry. `MachineRegistry.bootMachine` provisions
  per-user homes.
- `pkg/character/` — Nixy's dialog data and queue.
- `pkg/debug/` — Snapshot ring (last N keystrokes' state). Ctrl+\
  dumps it to disk; useful when something weird happens in `make repl`.

## Testing

- `make test` — full suite, requires `-p 1` due to the global binary
  registry.
- `make fuzz` — runs both fuzz tests 10× each.
- `make check-wasm` — confirms the web build still compiles (no
  Bubbletea or other non-WASM imports leaked into `pkg/`).

### Fuzz strategy

Two fuzz tests with different goals:

- `TestFuzzQuests` (`pkg/game/quests/`) — bypasses the real shell,
  drives quests via `shellState`. Catches quest-logic bugs but
  **cannot** catch shell-level regressions.
- `TestFuzzE2EHintGuided` (`pkg/session/`) — drives keystrokes through
  `Session.HandleKeystroke`, the same chokepoint used by both CLI and
  web. The canary for end-to-end correctness.

Liveness in the E2E fuzz is keyed on quest progression
(`activeQuestID + numCompletedQuests + hostname + cwd`). Tracker
record count is deliberately **not** included — it grows on every
Enter even when nothing real is happening, which lets stuck quests
masquerade as progress. Terminal demands all-Complete; "planner has
a path" is too lenient.

## Conventions

- All dev gestures are Makefile targets: `test`, `fuzz`, `build`,
  `repl`, `web`, `serve`, `check-wasm`. Don't invent ad-hoc invocations.
- Bubbletea is only imported in `cmd/repl/` — never in `pkg/` (would
  break WASM compilation).
- `web/nixy.wasm` and `web/wasm_exec.js` are build artifacts
  (`.gitignored`).
- New commands registered via `simulation.Register` should set
  `OptionalArgs` appropriately. Default false is correct for anything
  that requires arguments.
