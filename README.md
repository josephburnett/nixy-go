# nixy-go

Nixy is a game that teaches Unix by having you use it. You log in, help
your friend Nixy with their computers, and pick up real commands along the
way: `ls`, `cd`, `cat`, `grep`, pipes, `ssh`, `sudo`.

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

## Running it

```
make repl     # play in your terminal (Ctrl+C twice to quit)
make serve    # build the WASM version and serve it on http://localhost:8080
make test     # run the test suite
make fuzz     # run the quest and end-to-end fuzz tests 10x each
```

Needs Go 1.24+ and, for `make serve`, Python 3.

## Code

The architecture, design principles and testing strategy are in
[CLAUDE.md](CLAUDE.md). In short: a simulated Unix (`pkg/simulation`,
`pkg/command`) under a quest system (`pkg/game`), Nixy's dialog
(`pkg/character`), and a renderer that draws the same screen as ANSI for
the terminal (`cmd/repl`) or HTML for the browser (`cmd/web`).
