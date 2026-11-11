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
such file`). Errors that come from invalid typing are bugs in a
command's `ValidArgs` or in `shell.Next()`, not runtime concerns.

This is what makes exploration safe. A beginner mashing keys learns,
not panics.

Implementation:
- Every `simulation.Binary` has a `ValidArgs` (`Register` panics
  without one). It is the single answer to "what may be typed after
  this command name": a `Suggestion` with the chars that come next,
  whether the line can run as typed (`Complete`), and whether it then
  prints to stdout (`Writes`). There is no "anything goes" fallback.
  - Zero-arg commands answer Complete for `""` (`ls`, `pwd`). NOT
    standalone cat or grep: those would hang, since the game has no
    terminal stdin.
  - A pipe connects a command that prints to one that reads stdin.
    `Writes` gates `|`; commands that only change files (rm, mv,
    touch, apt install) print nothing to pipe, so a pipeline never
    changes the world a later segment was validated against. Under
    `ArgContext.InPipe` only `ReadsStdin` commands lead anywhere, with
    exactly `PipeRequired` args (`ls | cat`, `ls | grep target`; not
    `ls | ls`, which runs but connects nothing).
  - `Interactive` binaries (ssh, shell) take over the terminal: they
    are kept out of pipelines and out of sudo.
- Validators are built from parts in `pkg/command/args.go`:
  - `Positional`: positional `Arg`s, how many are required standalone
    and in a pipe, whether it `Writes`, `Variadic` (repeat the last
    Arg: `echo WORD...`), and `Flags`: single-letter options, each at
    most once, alone or bundled, before the first argument (`ls -a`,
    `kill -9 PID`). Commands read them back with `command.Options`;
    never parse `-x` ad hoc. `NoArgs` is the empty Positional.
  - `ValidCommand`: "NAME ARGS" over a set of names, each with its own
    validator, offering a name only if its args can lead somewhere.
    It serves the shell's command position, sudo, and `Subcommands`
    (apt install|list, nx quest|log|panic).
  - Path args (`FilePath`, `AnyPath`, `CdTarget`, `NewPath`,
    `NewName`, `EmptyDir`, `MoveDest`, `RedirectTarget`) share
    `pathArg`: they offer only keys that lead to an accepted path, and
    a folder only if something accepted lies below. File args never
    complete on folders (`cat .`).
  - Free-form words (`Word`, `Pattern`, new names) use portable
    filename characters a real shell won't interpret (no `|`, `>`,
    `*`), no leading `-`, and only keys on the on-screen keyboard.
- **Every valid key is on the on-screen keyboard.** The web build on a
  phone has no other input. `terminal.OnScreen(d)` says whether a key
  is drawn (digits, letters, `>`, `~`, `^C`, the special row); the
  session tests' `pressAccepted` and the random fuzz fail on a valid
  key that isn't, which covers every hint and planned command too.
- Commands are found one way: `sim.LookupBinary` via `/bin/<name>`,
  for running, validating and sudo alike, as a user (root under sudo)
  with the same permission checks as any file. Builtins (cd, exit,
  help, nx) have validators in `shell.builtinArgs` and can't be piped.
- `shell.Next()` is pipe-aware: `command.Parse` splits the line on
  `|`, and only the last segment is validated. The shell adds Enter
  and `|` (when the segment is Complete and pipeable), the optional
  space after `|`, and `>`. A pipeline finds every segment's binary
  before launching any.
- Redirection (`>` replaces, `>>` appends) is offered exactly where
  `|` is, only on the last segment; then a second `>`, one optional
  space and the target (`command.RedirectTarget`: a text file or a new
  name in an existing folder, as the shell's owner, never a file the
  line names as an operand: `cat f > f` would read an emptied file).
- A key the keyboard doesn't offer does nothing at all: the session
  returns before recording, running or redrawing anything.
- Permission denied is not a principle-1 violation: it is a real
  consequence (principle 2), as is kill's "Operation not permitted" on
  someone else's process. Usage errors, missing paths and wrong file
  types are bugs.
- Guards:
  - `TestKeyboardWalk_NoMistakes` (`pkg/session/`) walks the keyboard
    (`|` and `>` included) over the real worlds in several states
    (`keyboardScenarios`) and checks every reachable state: never a
    dead end, every Enter terminates (or, for a command that never
    ends, Ctrl+C is offered and ends it) with no stderr but permission
    errors, and a pipeline leaves every file as it was.
  - `TestShellNextInvariant_EveryEnterStateTerminates` does the same on
    a small fixture with every command (`testBinaries`);
    `TestShellNextInvariant_Lines` pins whole lines per bug class.
- Known gaps: `ssh` to a host in /etc/hosts that isn't booted says
  "connection refused" (no world does that today).

### 2. Real Unix, not a toy (simulation layer)

The mental model the player builds here must transfer to real Unix.
Permissions, ownership, paths, processes, pipes are honest — if
minimal — implementations of the real concepts. When the player learns
`sudo`, they're learning real `sudo`.

Concrete implications (these all flow from this single principle):
- **Identity is real.** The username chosen at login is the player's
  account on every machine. The prompt (`<user>@<host>:<path>>`) is
  ground truth: it matches `whoami`, owns the files the player
  creates, and is what `CanWrite` checks. Existing accounts (root,
  nixy: `game.Accounts`, every owner in the world definitions) can't
  be taken at login, as there are no passwords to prove them.
- **Homes and $HOME are real.** `MachineRegistry.bootMachine`
  provisions `/home/<username>` on every machine as it boots; worlds
  ship no placeholder home. A shell's home is its owner's
  (`file.Home`; `/root` for root). Login shells (`sim.Login`: the
  laptop login and ssh; `-sh` in ps) start there, or in `/` if it
  can't be entered; bare `cd` goes there. The prompt shows it as `~`.
  The shell expands `~` and `~/...` in every arg before the command
  runs (`file.ExpandHome`; under sudo it is still the player's home),
  and path validators offer `~` through the same function.
- **Permissions matter.** Don't work around `CanWrite`/`CanRead` to
  make a quest succeed: fix the world or the quest. `/home/nixy` ships
  `Common: Write` on purpose (Nixy is loose with perms, so early
  quests need no sudo); the Lockdown quest tightens it to Read, and
  `sudo rm old.txt` becomes the way back in.
- **Permissions are checked one way.** Three levels (None/Read/Write);
  on a folder, Read means listing and search (Unix's r and x), Write
  adds creating and deleting entries. `file.Lookup(path, user)` walks
  a path as the kernel does: every folder passed through must be
  readable. Commands use it (or `LookupDir`, `ReadFile`, `CreateFile`,
  `DeleteFile`); `Navigate` ignores permissions and is for the game
  layer only. Root passes every check. Completion respects the same
  rules: an unreadable folder can be named (`ls /root` → permission
  denied, a real lesson) but nothing inside it is offered.
- **Installing software takes root.** `apt install` writes `/bin` as
  whoever runs it, so a normal user gets real apt's lock error ("are
  you root?"). Worlds (`pkg/game/worlds`) ship the everyday basics;
  the rest are a `sudo apt install` away, which is where quests teach
  them. Files created under sudo are owned by root.
- **Errors and output look like Unix.** File errors are `file.Err*`
  sentinels carrying strerror text; `command.Fail` prints them as
  coreutils does (`cat: foo: No such file or directory`). Exit
  statuses are real: 0 success, 1 failure (grep: 1 no match, 2
  trouble), 126 not allowed to run, 127 not found, 128+n killed by
  signal n. Commands print data as it is (`cat` adds no newline);
  output that doesn't end its line is ended before the next prompt,
  as zsh does. World text files end in a newline
  (`TestTextFilesEndInNewline`), so `wc -l` agrees with `cat`.
- **Processes are real** (`pkg/process`). A process's output is one
  ordered sequence of stdout and stderr writes (`Read`). Every process
  exits with a status (`Exit`). End of input (`CloseStdin`) is not
  `Kill`: a pipe closes the next stage's stdin when the stage before
  it exits. A pipeline's status is its last stage's; every stage's
  stderr reaches the terminal. Commands embed `command.Proc` (output
  queue, exit status, owner, no input); a pipe reader adds Stdin and
  CloseStdin.
- **Every process is in a process table.** Everything starts through
  `sim.Start` (or `sim.Launch` by name, or `sim.Login`), which puts it
  in its computer's table with a PID and command line until it exits.
  `computer.C.Signal(pid, user, sig)` is kill(2): owner or root only.
  `ps` / `ps -e` read the table when first read, so they list
  themselves; `kill [-9] PID` offers only live PIDs. (ssh is not a
  process of its own: its table entry on the client is the remote
  shell, so `kill -9` of it reports "Killed", not a closed
  connection.)
- **Signals are real.** The default action (`process.Written.Signal`)
  is to die with 128+n. Interactive shells ignore SIGINT and SIGTERM,
  like bash. A pipeline signals every stage, and a writer whose reader
  has exited gets SIGPIPE, so `yes | head` ends. The shell reports a
  job killed by SIGKILL/SIGTERM/SIGHUP as `Killed`/`Terminated`/`Hangup`.
- **Ctrl+C is SIGINT to the foreground job.** While a command runs,
  the shell offers `process.SigInt` as a key, echoes `^C` and signals
  the job (`$?` 130). It isn't offered at an idle prompt, nor by a
  shell whose job is interactive (ssh): the remote shell decides. So a
  command that never ends always has a way out.
- **Redirection is the shell's, as the player.** Before any command
  runs the shell opens the target (`shell.openRedirect`), creating it
  or emptying it, so `ls > files` lists `files` and a redirect that
  fails runs nothing. Under `sudo` the file is still opened as the
  player, the classic lesson. The job's stdout goes to the file it
  opened, like a file descriptor.
- **Nothing hangs.** Processes act only when called. A pipeline moves
  data one step per `Read`, and the session reads a bounded number of
  times per keystroke, so a source that never ends can't freeze the
  game; within that bound it keeps reading while a command runs, so
  `yes | head | wc -l` finishes in the keystroke that started it.

When choosing how to fix a bug or add a feature, ask: *would this
choice make sense in real Unix?* If it papers over a real concept,
it's the wrong fix.

### 3. Guided progression (game-design layer)

A bare faithful-Unix shell would be useless to a beginner. The game
loop wraps it in scaffolding:

- **Quests** introduce concepts in order, each unlocked by achievements
  granted by the prior one, each building on what's been taught. The
  line (`quests.All`): Connect (ssh) → Orientation (pwd, ls, cd) →
  Inspection (cat, grep) → Modification (rm, touch) → Composition
  (`|`) → Permissions (server: apt, sudo) → Lockdown (sudo in Nixy's
  locked home) → Identity (whoami, sudo whoami) → Notes (mkdir, echo,
  `>`, `>>`) → Manual (install wc, man) → Counting (wc -l, grep | wc
  -l) → Interrupt (yes, ^C) → Processes (ps -e, kill refused, sudo
  kill).
- **The planner** (`Quest.PlanNextCommand`, `game.PlanHint`) computes
  the next keystroke toward the active quest: the green hint key.
  `game.PlanThought` says why, in the thought line; every plannable
  command has a thought. While the player's last line still runs (and
  isn't ssh), `Game.GetPlannedCommand` plans `game.Interrupt` (the ^C
  key), whatever the quest.
- **Quests check outcomes, not spelling.** A quest whose goal is an
  effect checks the world (`/etc/config` exists). Otherwise it asks
  the tracker what ran and how it ended: `CommandRecord.Invocations()`
  parses the line into programs (sudo seen through, `~` expanded, each
  named for the program that really ran, resolved through /bin before
  the line runs: `game.Resolve`), with args resolved as paths from the
  folder the command ran in (`Invocation.HasPath`), and `OK()` /
  `Failed()` read its status. So `cat readme.txt` and `cat
  ~/../nixy/readme.txt | grep x` both read Nixy's readme, while a grep
  that matched nothing or a rejected key don't. A deliberate failure
  is an outcome too: Permissions plans `apt install touch` until a
  failed one is recorded. Never match command strings.
- **Recovery: the player can break anything, never the quest line.**
  Planners compute each step from the world as it is now, and:
  - before planning a command they call `need(ctx, name)`: a missing
    `/bin/<name>` gets `sudo apt install <name>` (if sudo and apt
    work), anything else (sudo or apt gone, a program replaced by
    another) gets `nx panic`, which puts every machine's `/bin` back
    as shipped, keeping properly installed programs.
  - a quest implementing `game.Repairer` puts its fixtures back after
    every command while it's active, from the world definition
    (`restore`; `restoreText` when the contents matter, like a system
    log with errors to find). Nixy says so in the dialog.
  A hint the keyboard rejects is a planner bug, and the fuzz fails on
  it; hints are never filtered to hide one.
- **Nixy's dialog** (`pkg/character/`) frames each concept
  narratively, before and after. The player isn't memorizing — they're
  helping someone.

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

- `cmd/repl/` — CLI via Bubbletea: `tea.KeyMsg` → `process.Datum`,
  resize, Ctrl+\ snapshot dump. Ctrl+C is SIGINT whenever the keyboard
  offers `^C`; only at a prompt does it count toward the double-press
  quit.
- `cmd/web/` — WASM via `syscall/js`, exporting `nixyInit`,
  `nixyKeystroke`, `nixyResize`, `nixyDumpState`. The on-screen
  keyboard is tappable: `Layout()` sets `Segment.Key` on keys valid
  now, HTMLRenderer emits it as `data-key`, and `web/index.html` sends
  it like a typed key. `terminal.KeyName` / `KeyDatum` are the one
  mapping, Ctrl+C (`^C`) and Ctrl+L (`^L`) included.
- Both use `pkg/session/`. The session never ends itself: `exit` on
  the laptop logs out to `login:`, and each username keeps its own
  game, so logging back in resumes it. `nx panic` restores `/bin` and
  replaces the whole shell stack with a fresh laptop shell. Only the
  CLI's double Ctrl+C quits. The keyboard must never be empty — the
  web build has no other input.

### Key packages

- `pkg/process/` — `P` interface (ordered output, end of input, exit
  status, signals), `Written`, process tables (`Space`), Datum types.
- `pkg/simulation/` — Binary registry, computers, process launching.
- `pkg/command/` — validator parts (`args.go`), `Parse`, `Proc`, and
  one package per command; `shell/` holds the shell, its builtins,
  pipelines, redirection and the pipe-aware `Next()`.
- `pkg/guide/` — wraps a process and forwards only keys its `Next()`
  offers. The bouncer for principle #1.
- `pkg/game/` — quest manager, planner, command tracker, machine
  registry; `quests/` and `worlds/` are the content.
- `pkg/character/` — Nixy's dialog data and queue.
- `pkg/debug/` — snapshot ring of the last keystrokes' state; Ctrl+\
  dumps it to disk.

## Testing

- `make test` — full suite. Packages and sessions share no mutable
  state: the binary registry is filled from `init` and read-only after
  the first lookup, and nx reaches the shell through
  `simulation.S.Nx`, not a global.
- `make fuzz` — runs the fuzz tests 10× each.
- `make check-wasm` — confirms the web build still compiles.

Fuzz and liveness tests, with different goals:

- `TestFuzzQuests` (`pkg/game/quests/`) — types whole lines into a
  real login shell (`shellState`), bypassing the keyboard. Catches
  quest-logic bugs, not keystroke-level ones.
- `TestFuzzE2EHintGuided` (`pkg/session/`) — keystrokes through
  `Session.HandleKeystroke`, the chokepoint both front ends use, 80%
  following the planner. Every key goes through `pressAccepted`,
  which fails if the keyboard doesn't offer it or the hint isn't
  valid. Liveness is keyed on quest progression (`worldKey`), not
  tracker size, which grows on every Enter and would let a stuck
  quest look like progress.
- `TestFuzzE2ERandomKeys` — random valid keys (half the time the hint)
  on through logouts, logins and nx panics, now and then typing lines
  that break things (`scriptedLines`). Fails on panics, an empty
  keyboard, a hint that isn't valid, stderr other than a real
  consequence, or an input line that isn't what the shell holds. Then
  following only the hints (`followHints`) must finish every quest.
- `TestFuzzE2ERejectedKeys` — random play that also presses keys the
  keyboard doesn't offer, failing if any changes anything observable,
  and checks the process tables hold exactly the live shells.
- `TestBrokenPrerequisitesRecover` — for each quest, breaks what it
  depends on and demands the hints still finish the line.
- `TestPlannedCommandsAreTaught` (`pkg/game/quests/`) — every program
  a plan runs appears in backticks in Nixy's dialog before it's
  planned.
- `TestPlaythrough` (`pkg/session/`) — presses only hint keys from
  login to the end and keeps a transcript: `NIXY_TRANSCRIPT=FILE go
  test ./pkg/session -run TestPlaythrough` writes it out. Read it
  after changing quests or dialog.

## Conventions

- All dev gestures are Makefile targets: `test`, `fuzz`, `build`,
  `repl`, `web`, `serve`, `check-wasm`, `lint`. Don't invent ad-hoc
  invocations.
- Bubbletea is only imported in `cmd/repl/` — never in `pkg/` (would
  break WASM compilation).
- `web/nixy.wasm` and `web/wasm_exec.js` are build artifacts
  (`.gitignored`).
- A new command: register it with `simulation.Register` and a
  `ValidArgs` built from `pkg/command/args.go` (usually `Positional`),
  `Interactive` if it takes over the terminal. Add it to
  `testBinaries` in `pkg/command/shell/shell_test.go`, import it in
  `pkg/command/all`, give it an apt package in `apt.PackageRegistry`
  (`TestEveryPackageIsRegistered`), and write its manual page in
  `pkg/command/man/pages.go` (`TestEveryCommandHasAPage`): short, and
  true of the real command.
