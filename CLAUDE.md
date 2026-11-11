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
- Every `simulation.Binary` must have a `ValidArgs` (`Register` panics
  without one). It is the single answer to "what may be typed after
  this command name": which chars come next (`Suggestion.Chars`) and
  whether the line can run as typed (`Suggestion.Complete`). There is
  no "anything goes" fallback.
  - Zero-arg commands answer Complete for `""` (`ls`, `pwd`). NOT
    standalone cat or grep — those hang without an upstream pipe
    because the game has no terminal stdin.
  - A pipe connects a command that prints to one that reads stdin.
    `Suggestion.Writes` (from `Positional.Writes`: ls, cat, grep, pwd,
    apt list) gates `|`; commands that only change files (rm, mv,
    touch, apt install) print nothing to pipe, so a pipeline never
    changes the world a later segment was validated against. Under
    `ArgContext.InPipe` only `ReadsStdin` commands lead anywhere, and
    they take exactly `PipeRequired` args (`ls | cat`, `ls | grep
    target`; not `ls | ls` or `ls | cat file`, which run but connect
    nothing).
  - `Interactive` binaries (ssh, shell) take over the terminal: they
    are kept out of pipelines and out of sudo.
- Validators are built from parts in `pkg/command/args.go`:
  - `Positional` — positional `Arg`s, how many are required standalone
    and in a pipe (`ReadsStdin`, `PipeRequired`), and whether the
    command prints (`Writes`). It offers the space
    between args and the trailing space before `|`. `NoArgs` is the
    empty one. `Variadic` repeats the last Arg (`echo WORD...`).
  - Options are `Positional.Flags`: single-character flags, each at
    most once, alone or bundled, before the first argument (`ls -a
    DIR`, `ps -e`, `kill -9 PID`). The launch side reads them with
    `command.Options(args)`, which splits them off the same way. Add
    a flag to a command this way, never by parsing `-x` ad hoc.
  - `ValidCommand` — "NAME ARGS" over a set of names, each with its
    own validator. Used for the shell's command position, sudo (any
    non-interactive /bin command, then that command's validator) and
    `Subcommands` (apt install|list, nx quest|log|panic). It offers a
    name only if its args can lead somewhere.
  - Path args (`FilePath`, `AnyPath`, `CdTarget`, `NewPath`,
    `NewName`, `EmptyDir`, `MoveDest`) offer only keys that lead to an
    accepted path: a folder is offered for descent only if something
    accepted lies below. File args never complete on folders (`cat .`,
    `rm ..`). `NewName` (mkdir) must not exist yet; `EmptyDir` (rmdir)
    is an empty folder the user can read.
  - Free-form words (new names, grep patterns, `Word` for echo) use
    portable filename characters only — nothing a real shell would
    interpret (`|`, `>`, `*`), no leading `-` — and only ones on the
    on-screen keyboard: lowercase, digits, `.` and `-` (no capitals
    or `_`, which have no key).
- **Every valid key is on the on-screen keyboard.** The web build on a
  phone has no other input, so a key that's valid but has no key on
  screen is a dead end for a tap-only player. `terminal.OnScreen(d)`
  says whether a key is drawn (digits, letters, `>`, `~`, `^C`, the
  special row); the session tests' `pressAccepted` and the random fuzz
  fail if any valid key isn't, which covers every hint and every
  planned command too.
- Commands are found one way: `sim.LookupBinary` via `/bin/<name>`,
  for running, validating and sudo alike. It looks up as a user
  (root under sudo) with the same permission checks as any file:
  running needs Read on the binary. Builtins (cd, exit, help, nx) have
  validators in `shell.builtinArgs` and can't be piped.
- `shell.Next()` is pipe-aware: it splits on `|` and validates only
  the segment after the last pipe. It adds just Enter and `|` (when
  the segment is Complete and pipeable) and the optional space after
  `|`. A pipeline finds every segment's binary before launching any.
- Output redirection (`>` replaces, `>>` appends) is the shell's, like
  `|`: `command.Parse` puts it on the segment (`Segment.Redirect`,
  partial input included) and the args end at the operator. `>` is
  offered exactly where `|` is (a complete command that `Writes`),
  only on the last segment, then a second `>`, one optional space and
  the target (`command.RedirectTarget`): a text file or a new name in
  an existing folder, validated as the shell's owner, never a folder
  or a file the line names as an operand (`cat f > f` would read an
  emptied file; GNU cat refuses it). After the target only Enter.
- A key the keyboard doesn't offer does nothing at all: the session
  returns before recording, running or redrawing anything
  (`TestFuzzE2ERejectedKeys` presses such keys during random play and
  checks that nothing observable changes).
- Permission denied is not a principle-1 violation: it is a real
  consequence (principle 2) and the guards allow it, as they allow
  kill's "Operation not permitted" on someone else's process (the
  same lesson for processes). Usage errors, missing paths and wrong
  file types are bugs.
- Guards:
  - `TestKeyboardWalk_NoMistakes` (`pkg/session/`) walks the keyboard
    (`|` included) over the real worlds at several world states and
    checks every reachable state: never a dead end (some key besides
    Backspace), every Enter terminates (by itself, or a command that
    never ends, like `yes`, by Ctrl+C, which must then be offered)
    with no stderr but permission denied, and a pipeline leaves every
    file as it was (except a redirect's target).
    Reaching each scenario types the planner's commands key by key.
  - `TestShellNextInvariant_EveryEnterStateTerminates` does the same on
    a small fixture with every command; `TestShellNextInvariant_Lines`
    pins whole lines per bug class.
  - The hint-guided fuzz can't catch these alone: it follows the
    planner's exact output, never exploring partial-input states.
- Known gaps: `ssh` to a host listed in /etc/hosts but not booted
  says "connection refused" (a real consequence; no world does that
  today, and the walk would flag it).

### 2. Real Unix, not a toy (simulation layer)

The mental model the player builds here must transfer to real Unix.
Permissions, ownership, paths, processes, pipes are honest — if
minimal — implementations of the real concepts. When the player learns
`sudo`, they're learning real `sudo`.

Concrete implications (these all flow from this single principle):
- **Identity is real.** The username chosen at login is the player's
  account on every machine. The prompt (`<user>@<host>:<path>>`) is
  ground truth — it matches `whoami`, owns the files the player
  creates, and is what `CanWrite` checks against. Existing accounts
  (root, nixy: `game.Accounts`, every owner in the world definitions)
  can't be taken at login: real login would ask for their password,
  and the game has none, so the login keyboard never offers Enter on
  one.
- **Per-user homes.** `MachineRegistry.bootMachine` provisions
  `/home/<username>` on every booted machine. Other users (Nixy) have
  their own homes. World definitions don't ship a placeholder
  `/home/user` — each player gets a real one.
- **$HOME is real.** A shell's home comes from its owner
  (`file.Home`: `/home/<user>`, `/root` for root). Login shells
  (`shell -l`: the laptop login and ssh) start there, falling back to
  `/` if it can't be entered; bare `cd` goes there, and like `cd DIR`
  is offered only when it would move. The prompt shows home as `~`,
  like bash's `\w`. The shell expands `~` and `~/...` in every arg
  before the command runs (`file.ExpandHome`; under sudo it is still
  the player's home), and path validators offer `~` at the start of a
  path through the same function, so typing and running agree. The
  `~` key sits at the end of the keyboard's bottom letter row.
- **Permissions matter.** Don't work around `CanWrite`/`CanRead` to
  make a quest succeed. If the player can't do something, that's a
  teaching opportunity — fix the world (perms, ownership) or the
  quest. The Permissions quest specifically teaches `sudo`; preserve
  the model it depends on. `/home/nixy` ships `Common: Write` on
  purpose — Nixy is loose with perms so early quests work without
  `sudo` — and the Lockdown quest (after Permissions) tightens it to
  Read: the player's `rm old.txt` there is denied (deleting writes
  the folder) and `sudo rm old.txt` is the way back in.
- **Installing software takes root.** `apt install` writes `/bin` as
  whoever runs it, so a normal user gets real apt's lock error ("are
  you root?") and `sudo apt install` works; installed files are owned
  by the installer. That is why Nixy ships grep, rm and touch (and
  sudo, for Lockdown): the early quests need no apt, and apt arrives
  with sudo in the Permissions quest (the server lacks touch; the planner has the
  player try `apt install touch`, read the error, then use sudo).
  Files created under sudo are owned by root.
- **Permissions are checked one way.** The model has three levels
  (None/Read/Write); on a folder, Read means both listing and search
  (real Unix's r and x), Write adds creating and deleting entries.
  `file.Lookup(path, user)` walks a path as the kernel does: every
  folder passed through must be readable. Commands go through it (or
  `LookupDir`, `ReadFile`, `CreateFile`, `DeleteFile`, which use it);
  `Navigate` ignores permissions and is for the game layer only (quest
  checks, world setup). Root passes every check.
- **Completion respects permissions.** Validators see `ArgContext.User`
  (root under sudo). A folder the user can't read can still be named
  (`ls /root`, `cd /root` → permission denied, a real lesson), but
  nothing inside it is offered, so the keyboard never steers into an
  unreadable tree or leaks its names.
- **Errors look like Unix errors.** File errors are `file.Err*`
  sentinels carrying strerror text; `command.Fail` prints them the way
  coreutils does (`cat: foo: No such file or directory`, `rm: cannot
  remove 'x': Permission denied`). Stderr surfaces normally, exit vs.
  EOF behave correctly, etc.
- **Processes are real** (`pkg/process`). A process's output is one
  ordered sequence of stdout and stderr writes (`Read`), shown on the
  terminal in the order written. Every process exits with a status
  (`Exit`; 0 is success, 1 for a failed command, 127 command not
  found, 126 not allowed to run, 137 killed). End of input
  (`CloseStdin`) is not `Kill`: a pipe closes the next stage's stdin
  when the stage before it exits, and a stage that doesn't read stdin
  ignores that and still prints. Every stage's stderr reaches the
  terminal; a pipeline's status is its last stage's. The shell keeps
  `$?` (`shell.Info.LastStatus`), and the tracker records it on every
  command line (`CommandRecord.Status`, `Running` for one still going,
  like ssh), so quests can check what a command did. Embed
  `process.Written` to get the output queue and exit status for free.
- **Every process is in a process table.** Everything starts through
  `sim.Start` (or `sim.Launch` by binary name), which puts it in its
  computer's table with a PID and its command line; it leaves the
  table when it exits. `computer.C.Processes()` lists PID, owner and
  command line; `computer.C.Signal(pid, user, sig)` (and `Kill`) lets
  only the owner or root signal (`process.ErrNotPermitted`,
  `ErrNoProcess`). Each pipeline stage is its own process; ssh shows
  as `ssh HOST` on the client and its remote `shell -l` on the
  server. `ps` (yours) and `ps -e` (everyone's) print `PID USER CMD`
  and read the table when first read, so they list themselves and
  their pipeline; `kill [-9] PID` offers only PIDs in the table.
  (ssh is not yet a process of its own: its table entry is the remote
  shell, so `kill -9` of it reports "Killed", not a closed
  connection.)
- **Redirection is done by the shell, as the player.** Before any
  command runs the shell opens the target (`shell.openRedirect`):
  creating it owned by the player, or emptying it for `>`, so `ls >
  files` lists `files`, and a redirect that fails (`echo hi >
  /etc/motd` → `/etc/motd: Permission denied`) runs nothing. Under
  `sudo` the command is root but the file is still opened as the
  player, the classic real-Unix lesson. The job's stdout goes to the
  file it opened (like a file descriptor, it follows the file, not the
  name); stderr still reaches the terminal.
- **Nothing hangs.** Processes act only when called. A pipeline moves
  data one step per `Read`, and the session reads a bounded number of
  times per keystroke, so a source that never ends can't freeze the
  game. Within that bound it keeps reading while a command runs, even
  through quiet reads, so a pipeline that works before it prints
  (`yes | head | wc -l`) finishes in the keystroke that started it.
- **Signals are real.** `P.Signal(sig)` delivers one; the default
  (`Written.Signal`) is to die of it with status 128+n (`SigInt` 130,
  `SigPipe` 141, `SigTerm` 143; `SigKill` is `Kill`, 137).
  `computer.C.Signal(pid, user, sig)` is kill(2): owner or root only.
  Interactive shells ignore SIGINT and SIGTERM like bash (SIGHUP and
  SIGKILL end them). A pipeline signals every stage, and a stage whose
  reader has exited gets SIGPIPE, so `yes | head` ends. When a
  foreground job dies of SIGKILL/SIGTERM/SIGHUP the shell prints
  `Killed`/`Terminated`/`Hangup` on stderr, as bash does.
- **Ctrl+C is SIGINT to the foreground job.** While a command runs,
  the shell offers `process.SigInt` as a key (on top of whatever the
  job offers), echoes `^C` and signals the job; `$?` is 130. It is not
  offered at an idle prompt, nor by a shell whose job is interactive
  (ssh): there the remote shell decides, by the same rule. So a
  command that never ends (`yes`) always leaves a way out, and the
  keyboard is never empty. The key is `^C` on the on-screen keyboard
  (`terminal.KeyName`); the CLI sends it on Ctrl+C only when it is
  offered, and the web build maps Ctrl+C to it unless text is
  selected.

When choosing how to fix a bug or add a feature, ask: *would this
choice make sense in real Unix?* If it papers over a real concept,
it's the wrong fix.

### 3. Guided progression (game-design layer)

A bare faithful-Unix shell would be useless to a beginner. The game
loop wraps it in scaffolding:

- **Quests** introduce concepts in order, each unlocked by achievements
  granted by the prior one. New quests should build on what's been
  taught, not require leaping forward.
- **The planner** (`Quest.PlanNextCommand`, `game.PlanHint`) computes
  the next concrete keystroke toward the active quest. This is what
  the green hint key on the keyboard means. `game.PlanThought` says
  why, in the thought line; every plannable command has a thought.
- **Quests check outcomes, not spelling.** A quest whose goal is an
  effect checks the world (`/etc/config` exists, junk.txt is gone).
  Otherwise it asks the tracker what ran and how it ended:
  `CommandRecord.Invocations()` parses the line into programs (sudo
  seen through, `~` expanded) whose args resolve as paths from the
  folder the command ran in (`Invocation.HasPath`), and `OK()` /
  `Failed()` read its status. So `cat readme.txt`, `cat
  /home/nixy/readme.txt` and `cat ~/../nixy/readme.txt | grep x` all
  count as reading Nixy's readme, while a grep that matched nothing,
  `lsx`, or a rejected command don't. A deliberate failure is an
  outcome too: Permissions plans `apt install touch` until a failed
  one is recorded, Lockdown plans `rm old.txt` until a denied one
  is. Never match command strings.
- **Recovery: the player can break anything, never the quest line.**
  Planners compute each step from the world as it is now, and:
  - before planning a command they call `need(ctx, name)`: when
    `/bin/<name>` is missing they plan `sudo apt install <name>` (if
    sudo and apt work), and otherwise, or when it's been replaced by
    another program, `nx panic`. `nx panic` puts every machine's
    `/bin` back as shipped, keeping properly installed programs and
    dropping anything else, so after it `sudo apt install` always
    works. It is introduced in the Permissions dialog, where sudo
    makes breaking things possible.
  - a quest implementing `game.Repairer` puts back its own fixtures
    after every command while it's active (readme.txt until it's
    read, the system log until it's searched, the projects folder and
    its files, Nixy's home, the server's /etc), from the world
    definition, keeping whatever is there and of the right type. Nixy
    says so in the dialog ("/home/nixy/readme.txt went missing. I put
    it back"): it's her file, and she's a real user on her machine.
  Hints are never filtered to hide a bad plan: a hint the keyboard
  rejects is a planner bug, and the fuzz fails on it.
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
  Ctrl+\ snapshot dump. Ctrl+C is SIGINT instead whenever the keyboard
  offers `^C` (a command is running); only at a prompt does it count
  toward the double-press quit.
- `cmd/web/` — WASM via `syscall/js`. Exports `nixyInit`,
  `nixyKeystroke`, `nixyResize`, `nixyDumpState` to JavaScript.
  The on-screen keyboard is tappable: `Layout()` sets `Segment.Key` on
  keys valid right now, HTMLRenderer emits it as `data-key`, and
  `web/index.html` sends it to `nixyKeystroke` like a typed key
  (`terminal.KeyName` / `KeyDatum` are the one mapping).
- Both use `pkg/session/` for shared game/guide/shell initialization.
- The session never ends itself. `exit` on the laptop logs out to the
  `login:` prompt; each username keeps its own game, so logging back in
  resumes it. `nx panic` puts back every machine's shipped `/bin`
  (from its world definition; programs installed since stay, anything
  else there goes) and replaces the whole shell stack with a fresh
  laptop shell (never a logout). Only the CLI's double Ctrl+C quits.
  The keyboard must never be empty — the web build has no other input.

### Key packages

- `pkg/process/` — `Process` interface (ordered output, end of input,
  exit status), `Written` (embed it), process tables (`Space`), Datum
  types.
- `pkg/simulation/` — Binary registry, computer management, process
  launching. `Binary.ValidArgs` (required) drives the keyboard.
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

- `make test` — full suite. Packages and sessions share no mutable
  state: the binary registry is filled from `init` and read-only from
  the first lookup (`Register` panics after that), and nx reaches the
  shell through the game's simulation (`simulation.S.Nx`), not a
  global.
- `make fuzz` — runs the fuzz tests 10× each.
- `make check-wasm` — confirms the web build still compiles (no
  Bubbletea or other non-WASM imports leaked into `pkg/`).

### Fuzz strategy

Fuzz and liveness tests with different goals:

- `TestFuzzQuests` (`pkg/game/quests/`) — bypasses the real shell,
  drives quests via `shellState`. Catches quest-logic bugs but
  **cannot** catch shell-level regressions.
- `TestFuzzE2EHintGuided` (`pkg/session/`) — drives keystrokes through
  `Session.HandleKeystroke`, the same chokepoint used by both CLI and
  web. The canary for end-to-end correctness. Every key is typed
  through `pressAccepted`, which fails if the keyboard doesn't offer
  it or if the hint key isn't itself valid. (The session drops a hint
  the keyboard doesn't offer: the planner plans command lines, so
  while one runs, with only ^C offered, it shows no hint.)
- `TestFuzzE2ERandomKeys` (`pkg/session/`) — random valid keys (half
  the time the hint) through the same chokepoint, wandering into rm,
  mv, sudo and long pipelines, and on through logouts, logins and nx
  panics; each seed starts random play at a different quest and now
  and then types lines that break things (`sudo rm /bin/sudo`, delete
  a quest's file). Fails on panics, an empty keyboard, stderr other
  than permission denied, or a hint that isn't a valid key. Then the
  liveness check: from wherever random play left the world, following
  only the hints (`followHints`) must finish every quest within
  `hintFinishKeys` keys, never meeting an invalid hint, a plan without
  a thought, or an empty prompt with no plan.
- `TestBrokenPrerequisitesRecover` (`pkg/session/`, deterministic) —
  for each quest, breaks what it depends on (its files, the commands
  it plans, sudo, apt) and demands the hints still finish the line.
- `TestPlannedCommandsAreTaught` (`pkg/game/quests/`) — every program
  a plan runs appears in backticks in Nixy's dialog before it's
  planned.
- `TestFuzzE2ERejectedKeys` (`pkg/session/`) — random play that also
  presses keys the keyboard doesn't offer, failing if any changes
  anything observable, and checks after every key that the process
  tables hold exactly the live shells.

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
- New commands registered via `simulation.Register` need a `ValidArgs`
  built from the parts in `pkg/command/args.go` (usually
  `Positional`), and `Interactive` if they take over the terminal.
  Add the command to `testBinaries` in `pkg/command/shell/shell_test.go`
  so the structural walk covers it, import it in `pkg/command/all`
  (what the front ends and session tests import), and give it an apt
  package in `apt.PackageRegistry` (`TestEveryPackageIsRegistered`
  checks every package runs). Worlds ship the basics; the rest are a
  `sudo apt install` away, which is where quests teach them. Write
  its manual page in `pkg/command/man/pages.go`
  (`TestEveryCommandHasAPage`): short, and true of the real command.
  `man CMD` offers only commands in /bin with a page; `help [BUILTIN]`
  is the shell's, for its builtins (`shell.builtinHelp`).
