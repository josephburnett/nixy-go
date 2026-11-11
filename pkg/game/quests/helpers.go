package quests

import (
	"fmt"
	"slices"
	"strings"

	"github.com/josephburnett/nixy-go/pkg/command/apt"
	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/game"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Planners build each step from the world as it is now, and before
// planning a command they make sure it can run (see need), so a plan
// the player has broken turns into the step that recovers it.

func fileExists(sim *simulation.S, hostname string, path []string) bool {
	_, ok := lookup(sim, hostname, path)
	return ok
}

// isFile reports whether path is a text file.
func isFile(sim *simulation.S, hostname string, path []string) bool {
	f, ok := lookup(sim, hostname, path)
	return ok && f.Type == file.Text
}

func lookup(sim *simulation.S, hostname string, path []string) (*file.F, bool) {
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil, false
	}
	f, err := c.Filesystem.Navigate(path)
	return f, err == nil
}

// binaryInstalled reports whether /bin/name exists as a program.
func binaryInstalled(sim *simulation.S, hostname, name string) bool {
	f, ok := lookup(sim, hostname, []string{"bin", name})
	return ok && f.Type == file.Binary
}

// runnable reports whether user can run name on hostname and get the
// real program: /bin/name is that program (not another one renamed),
// and user may run it.
func runnable(sim *simulation.S, hostname, user, name string) bool {
	f, ok := lookup(sim, hostname, []string{"bin", name})
	if !ok || f.Type != file.Binary || f.Data != name {
		return false
	}
	_, err := sim.LookupBinary(hostname, user, name)
	return err == nil
}

// need returns the command that makes name runnable for the player on
// the current host, or "" when it already is. A missing package is
// installed with `sudo apt install`; anything else (sudo or apt
// themselves gone, a program replaced by something else) is what
// `nx panic` is for: it puts every machine's /bin back as shipped,
// keeping what was properly installed since.
func need(ctx game.PlanContext, name string) string {
	if runnable(ctx.Sim, ctx.Hostname, ctx.User, name) {
		return ""
	}
	if canSudoApt(ctx) && !fileExists(ctx.Sim, ctx.Hostname, []string{"bin", name}) {
		if _, ok := apt.PackageRegistry[name]; ok {
			return "sudo apt install " + name
		}
	}
	return "nx panic"
}

// canSudoApt reports whether the player can install software on the
// current host: sudo works for them and apt works for root.
func canSudoApt(ctx game.PlanContext) bool {
	return runnable(ctx.Sim, ctx.Hostname, ctx.User, "sudo") &&
		runnable(ctx.Sim, ctx.Hostname, file.OwnerRoot, "apt")
}

// planNavigate returns the command that gets the player toward
// targetHost, or "" when they are on it. Every machine is reached by
// ssh from the laptop.
func planNavigate(ctx game.PlanContext, targetHost string) string {
	if ctx.Hostname == targetHost {
		return ""
	}
	if ctx.Hostname != "laptop" {
		return "exit"
	}
	if n := need(ctx, "ssh"); n != "" {
		return n
	}
	return "ssh " + targetHost
}

// planCd returns the cd into dir, or "" when the player is in it.
func planCd(ctx game.PlanContext, dir []string) string {
	if slices.Equal(ctx.Cwd, dir) {
		return ""
	}
	return "cd /" + strings.Join(dir, "/")
}

// firstOf returns the first non-empty step.
func firstOf(steps ...string) string {
	for _, s := range steps {
		if s != "" {
			return s
		}
	}
	return ""
}

// restore puts path back on hostname as world has it, when the player
// has deleted or moved it (or anything on the way to it), or replaced
// it with something of another type. What is there and of the right
// type is kept. With deep, a folder's missing contents are put back
// too. It returns what Nixy says about it, or nil when nothing was
// missing.
func restore(sim *simulation.S, hostname string, world *file.F, path []string, deep bool) []string {
	c, err := sim.GetComputer(hostname)
	if err != nil {
		return nil
	}
	cur, want := c.Filesystem, world
	for i, name := range path {
		w, ok := want.Files[name]
		if !ok {
			return nil // nothing to put back
		}
		have, ok := cur.Files[name]
		if !ok || have.Type != w.Type {
			if cur.Files == nil {
				cur.Files = map[string]*file.F{}
			}
			cur.Files[name] = w
			return putBackLine(path)
		}
		if i == len(path)-1 && deep && putBackMissing(have, w) {
			return putBackLine(path)
		}
		cur, want = have, w
	}
	return nil
}

// restoreText is restore for a text file whose contents matter: one
// that no longer passes keep (the player emptied or replaced it) is
// put back as world has it too.
func restoreText(sim *simulation.S, hostname string, world *file.F, path []string, keep func(data string) bool) []string {
	if lines := restore(sim, hostname, world, path, false); lines != nil {
		return lines
	}
	f, ok := lookup(sim, hostname, path)
	w, wok := world.Navigate(path)
	if !ok || wok != nil || f.Type != file.Text || keep(f.Data) {
		return nil
	}
	*f = *w
	return []string{fmt.Sprintf("Huh, /%s isn't what it was. I put it back so we can keep going.", strings.Join(path, "/"))}
}

// putBackMissing adds want's entries that are missing from have (or of
// the wrong type), recursively, and reports whether it changed have.
func putBackMissing(have, want *file.F) bool {
	changed := false
	for name, w := range want.Files {
		h, ok := have.Files[name]
		switch {
		case !ok || h.Type != w.Type:
			if have.Files == nil {
				have.Files = map[string]*file.F{}
			}
			have.Files[name] = w
			changed = true
		case w.Type == file.Folder:
			if putBackMissing(h, w) {
				changed = true
			}
		}
	}
	return changed
}

func putBackLine(path []string) []string {
	return []string{fmt.Sprintf("Huh, /%s went missing. I put it back so we can keep going.", strings.Join(path, "/"))}
}

// readOK reports whether the player successfully read path on hostname
// with cat, or searched it with grep.
func readOK(t *game.CommandTracker, hostname string, path []string) bool {
	return t.AnyInvocation(hostname, game.CommandRecord.OK, func(inv game.Invocation) bool {
		return reads(inv, path)
	})
}

// reads reports whether the invocation reads path: a cat of it, or a
// grep through it.
func reads(inv game.Invocation, path []string) bool {
	switch inv.Name {
	case "cat":
		return inv.HasPath(0, path)
	case "grep":
		return inv.HasPath(1, path)
	}
	return false
}

// ranOK reports whether name ran and succeeded on hostname.
func ranOK(t *game.CommandTracker, hostname, name string) bool {
	return t.AnyInvocation(hostname, game.CommandRecord.OK, func(inv game.Invocation) bool {
		return inv.Name == name
	})
}

// ranOKAs reports whether name ran and succeeded on hostname, under
// sudo or not.
func ranOKAs(t *game.CommandTracker, hostname, name string, sudo bool) bool {
	return t.AnyInvocation(hostname, game.CommandRecord.OK, func(inv game.Invocation) bool {
		return inv.Name == name && inv.Sudo == sudo
	})
}

// planCdHome is planCd for a folder in the player's home, written with
// `~` as a person would: `cd ~`, `cd ~/notes`.
func planCdHome(ctx game.PlanContext, rel ...string) string {
	dir := append(file.Home(ctx.User), rel...)
	if slices.Equal(ctx.Cwd, dir) {
		return ""
	}
	return "cd " + strings.Join(append([]string{"~"}, rel...), "/")
}

// planRemove returns the next step in deleting whatever is at path on
// the current host, "" when nothing is there: an rm of a file, or an
// rmdir of an empty folder, working from the deepest entry up. It uses
// sudo when the player can't do it themselves (a folder root made).
func planRemove(ctx game.PlanContext, path []string) string {
	f, ok := lookup(ctx.Sim, ctx.Hostname, path)
	if !ok {
		return ""
	}
	path = slices.Clone(path)
	for f.Type == file.Folder && len(f.Files) > 0 {
		names := make([]string, 0, len(f.Files))
		for n := range f.Files {
			names = append(names, n)
		}
		slices.Sort(names)
		path = append(path, names[0])
		f = f.Files[names[0]]
	}
	cmd := "rm"
	if f.Type == file.Folder {
		cmd = "rmdir"
	}
	line := cmd + " /" + strings.Join(path, "/")
	c, err := ctx.Sim.GetComputer(ctx.Hostname)
	if err != nil {
		return ""
	}
	parent, err := c.Filesystem.Lookup(path[:len(path)-1], ctx.User)
	if err == nil && parent.CanWrite(ctx.User) && (f.Type != file.Folder || f.CanRead(ctx.User)) {
		return firstOf(need(ctx, cmd), line)
	}
	return firstOf(need(ctx, "sudo"), need(ctx, cmd), "sudo "+line)
}

// restoreHome puts the player's home back on hostname if it's gone (or
// isn't a folder), as the machine provisioned it.
func restoreHome(sim *simulation.S, hostname, user string) []string {
	if user == "" {
		return nil
	}
	home := file.Home(user)
	world := &file.F{Type: file.Folder, Files: map[string]*file.F{
		"home": {Type: file.Folder, Files: map[string]*file.F{
			user: {Type: file.Folder, Owner: user,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{}},
		}},
	}}
	if f, ok := lookup(sim, hostname, home[:1]); !ok || f.Type != file.Folder {
		return nil // no /home to put it in: that's the world's to repair
	}
	return restore(sim, hostname, world, home, false)
}
