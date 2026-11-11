# nixy-go

Nixy is a game that teaches Unix by having you use it. You log in, help
your friend Nixy with their computers, and pick up real commands along the
way: `ls`, `cd`, `cat`, `grep`, pipes, `ssh`, `sudo`, redirection, `man`,
processes and Ctrl+C.

It's a Go rewrite of the original nixy.
It runs in a terminal or in the browser (WASM).

## How it plays

- **No mistakes.** The on-screen keyboard only lights up keys that lead to
  a command that will run. You can mash keys and you'll never type
  something the shell rejects.
- **Real Unix.** Users, permissions, paths, pipes and ssh behave like the
  real thing, just smaller. What you learn here works on a real machine.
- **Guided.** Quests introduce one idea at a time, Nixy explains why, and
  a green key on the keyboard shows the next keystroke if you get stuck.

## The quests

Each quest teaches one idea and builds on the ones before it.

1. **Connect**: `ssh nixy` from your laptop.
2. **Orientation**: `pwd`, `ls /home` and `cd` into Nixy's home.
3. **Inspection**: `cat` her readme, and `grep` her system log for errors.
4. **Modification**: delete a file with `rm`, make one with `touch`.
5. **Composition**: `ls | grep target` — send one command's output into another.
6. **Permissions**: on the server, `apt install` fails until you use `sudo`.
7. **Lockdown**: Nixy locks her home, and `sudo rm` is the way back in.
8. **Identity**: `whoami`, and `sudo whoami` (root).
9. **Notes**: `mkdir` a folder and write a to-do list with `echo`, `>` and `>>`.
10. **Manual**: install `wc` and read its manual with `man`.
11. **Counting**: `wc -l` the log, then `grep error system.log | wc -l`.
12. **Interrupt**: run `yes`, which never stops, and stop it with Ctrl+C.
13. **Processes**: find Nixy's runaway process with `ps -e`, and stop it
    with `kill`, which needs `sudo`, because it's hers.

`nx quest` shows the current quest and `nx log` all of them. If you break
something you need (it's a real system: you can), `nx panic` puts every
machine's commands back.

## Running it

```
make repl     # play in your terminal (Ctrl+C twice to quit)
make serve    # build the WASM version and serve it on http://localhost:8080
make test     # run the test suite
make fuzz     # run the quest and end-to-end fuzz tests 10x each
```

`exit` on the laptop logs you out to the `login:` prompt. Log in with
the same name to pick up where you left off.

Needs Go 1.24+ and, for `make serve`, Python 3.

## Code

The architecture, design principles and testing strategy are in
[CLAUDE.md](CLAUDE.md). In short: a simulated Unix (`pkg/simulation`,
`pkg/command`) under a quest system (`pkg/game`), Nixy's dialog
(`pkg/character`), and a renderer that draws the same screen as ANSI for
the terminal (`cmd/repl`) or HTML for the browser (`cmd/web`).
