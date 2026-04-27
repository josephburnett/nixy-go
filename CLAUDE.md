# Nixy-Go

Educational Unix-learning game. Go reimplementation of ~/nixy.

The whole point is to **teach real Unix**: ownership, permissions, paths,
piping, sudo. The constraints in this codebase are pedagogical — they
exist so the player learns the right mental model. When deciding how to
fix a bug or add a feature, ask: *what does this teach?* If a "fix"
papers over a Unix concept (e.g. faking ownership to make `rm` succeed),
it's the wrong fix.

## Core invariants — do not violate

These keep getting forgotten. Treat them as the spine of the project.

### 1. The keyboard cannot let the player do something wrong

At every keystroke, the valid set returned by `shell.Next()` (and gated
by `pkg/guide`) must contain only keys that lead toward a correct,
executable action. The player should never be able to type a command
that the binary then rejects with "missing operand" or "no such file."
Errors that come from typing something invalid are a bug in `Next()`,
not a runtime concern.

Implementation hooks:
- `simulation.Binary.OptionalArgs` declares whether a command can run
  with zero args (`ls`, `cat`, `pwd` — true; everything else false).
- `Enter` and `|` are shell-level "execute this segment" keys. They are
  only valid at *segment-completion points* — exact-match of an
  optional-args command, or exact-match of a complete argument.
- `shell.Next()` is pipe-aware: it splits `currentCommand` on `|` and
  validates only the segment after the last pipe.
- Pinned by `TestShellNextInvariant_*` in `pkg/command/shell/shell_test.go`.

### 2. The prompt is ground truth

`<user>@<host>:<path>>` is a claim about who, where, and where in the
filesystem. It must match reality. If the prompt says `joe`, then `joe`
is the identity for permission checks; files joe creates are owned by
joe; `whoami` (when implemented) says joe.

Login is not cosmetic. The username chosen at login is the player's
real identity on every machine they reach. `Sim.Launch` is called with
`s.Username` as the shell owner, and that flows through every
permission check.

### 3. Home differs by user

When the game starts, `MachineRegistry.bootMachine` provisions
`/home/<username>` on every booted machine (initial and late-unlock),
owned by the player with `OwnerPermission: Write, CommonPermission: Read`.
Other users have their own homes — Nixy owns `/home/nixy` on the nixy
machine. The world definitions in `pkg/game/worlds/` deliberately do
**not** ship a `/home/user` placeholder; each player gets a real home.

Nixy's `/home/nixy` currently has `CommonPermission: Write` — she's
loose with permissions on purpose, so Modification/Composition quests
work without sudo. A future quest will tighten this and teach `sudo`
as the way back in. **Do not** undo Nixy's loose perms without that
quest in flight.

### 4. Permissions are part of the lesson

Don't work around `CanWrite` / `CanRead` to make a quest succeed. If
the player can't do something, that's the teaching opportunity — fix
the world (perms, ownership) or the quest, not the permission system.
The Permissions quest specifically teaches `sudo`; preserve the model
it depends on.

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
  ├── ANSIRenderer (ansi.go)      CLI terminal output (box-drawing + ANSI colors)
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

- `pkg/process/` — `Process` interface (`Stdout`, `Stderr`, `Stdin`,
  `Next`, `Kill`) and Datum types (`Chars`, `TermCode`, `Signal`).
- `pkg/simulation/` — Binary registry, computer management, process
  launching. `Binary.OptionalArgs` gates Enter at zero-args.
- `pkg/command/shell/` — Shell with builtins (`cd`, `exit`, `nx`),
  pipeline support, and the pipe-aware `Next()` that enforces
  invariant #1.
- `pkg/guide/` — Wraps a process, validates input against `Next()`
  before forwarding. The bouncer for invariant #1.
- `pkg/game/` — Quest manager, planner (keystroke hints), command
  tracker, machine registry. `MachineRegistry.bootMachine` provisions
  per-user homes.
- `pkg/debug/` — Snapshot ring (last N keystrokes' state). Ctrl+\ dumps
  it to disk; useful when the player hits something weird in `make repl`.

### Hint vs. validation

Two separate systems, often confused:

- **Validation** (`shell.Next()` → `guide.Stdin`): *what can be typed*.
  Must respect invariant #1.
- **Hint** (`game.PlanHint` in `pkg/game/planner.go`): *what the
  planner suggests typing next* — the green key on the keyboard. Hints
  are advisory.

When the player goes off-plan (typed prefix doesn't match the planned
command), `PlanHint` returns `nil`, **not** `TermBackspace`. Don't nag
the player — they may be running a different valid command before
returning to the plan. The planner re-engages on the next empty prompt.

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
  **cannot** catch shell-level regressions (e.g. permission, key
  validation, stderr handling).
- `TestFuzzE2EHintGuided` (`pkg/session/`) — drives keystrokes through
  `Session.HandleKeystroke`, the same chokepoint used by both CLI and
  web. This is the canary for end-to-end correctness.

Liveness in the E2E fuzz is keyed on quest progression
(`activeQuestID + numCompletedQuests + hostname + cwd`). It deliberately
does **not** include tracker record count — that grows on every Enter
even when nothing real is happening, which lets stuck quests masquerade
as progress. Terminal condition demands all-Complete; "planner has a
path" is too lenient.

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
