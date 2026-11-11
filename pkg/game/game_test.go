package game

import (
	"testing"

	"github.com/josephburnett/nixy-go/pkg/file"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// --- Achievement tests ---

func TestAchievementGrantAndHas(t *testing.T) {
	a := NewAchievementSet()
	if a.Has("test") {
		t.Fatal("should not have ungranted achievement")
	}
	a.Grant("test")
	if !a.Has("test") {
		t.Fatal("should have granted achievement")
	}
}

func TestAchievementHasAll(t *testing.T) {
	a := NewAchievementSet()
	a.Grant("a")
	a.Grant("b")
	if !a.HasAll([]Achievement{"a", "b"}) {
		t.Fatal("should have all")
	}
	if a.HasAll([]Achievement{"a", "c"}) {
		t.Fatal("should not have all when missing one")
	}
}

func TestAchievementHasAllEmpty(t *testing.T) {
	a := NewAchievementSet()
	if !a.HasAll(nil) {
		t.Fatal("empty requirement should always pass")
	}
	if !a.HasAll([]Achievement{}) {
		t.Fatal("empty slice requirement should always pass")
	}
}

func TestAchievementList(t *testing.T) {
	a := NewAchievementSet()
	a.Grant("x")
	a.Grant("y")
	list := a.List()
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
}

// --- CommandTracker tests ---

func TestTrackerRecord(t *testing.T) {
	tr := NewCommandTracker()
	tr.Record(CommandRecord{Hostname: "host", Cwd: []string{"home"}, Command: "ls"})
	if !tr.HasCommandOnHost("host") {
		t.Fatal("should have command on host")
	}
	if tr.HasCommandOnHost("other") {
		t.Fatal("should not have command on other host")
	}
}

func TestTrackerLastN(t *testing.T) {
	tr := NewCommandTracker()
	for _, c := range []string{"a", "b", "c"} {
		tr.Record(CommandRecord{Command: c})
	}
	if tr.Len() != 3 {
		t.Fatalf("Len = %d, want 3", tr.Len())
	}
	got := tr.LastN(2)
	if len(got) != 2 || got[0].Command != "b" || got[1].Command != "c" {
		t.Fatalf("LastN(2) = %+v, want b, c", got)
	}
	if got := tr.LastN(10); len(got) != 3 {
		t.Fatalf("LastN(10) has %d records, want all 3", len(got))
	}
}

func TestTrackerInvocations(t *testing.T) {
	r := CommandRecord{
		Hostname: "nixy", User: "joe", Dir: []string{"home", "joe"},
		Command: "sudo cat ~/../nixy/readme.txt | grep -v x",
	}
	invs := r.Invocations()
	if len(invs) != 2 {
		t.Fatalf("got %d invocations, want 2: %+v", len(invs), invs)
	}
	if invs[0].Name != "cat" || !invs[0].Sudo {
		t.Fatalf("first invocation = %+v, want cat under sudo", invs[0])
	}
	if !invs[0].HasPath(0, []string{"home", "nixy", "readme.txt"}) {
		t.Fatalf("cat's arg should resolve to /home/nixy/readme.txt: %v", invs[0].Paths(0))
	}
	if invs[1].Name != "grep" || invs[1].Sudo {
		t.Fatalf("second invocation = %+v, want plain grep", invs[1])
	}
	if !r.IsPipeline() || !r.Invokes("grep") || r.Invokes("gre") {
		t.Fatal("Invokes must match whole program names")
	}
}

func TestTrackerOutcomes(t *testing.T) {
	tr := NewCommandTracker()
	tr.Record(CommandRecord{Hostname: "h", Command: "lsx", Status: 127})
	tr.Record(CommandRecord{Hostname: "h", Command: "grep zzz f", Status: 1})
	tr.Record(CommandRecord{Hostname: "h", Command: "ssh nixy", Running: true})
	ran := func(name string) bool {
		return tr.AnyInvocation("h", CommandRecord.OK, func(i Invocation) bool { return i.Name == name })
	}
	if ran("ls") || ran("lsx") || ran("grep") || ran("ssh") {
		t.Fatal("failed or running commands must not count as successes")
	}
	if !tr.AnyInvocation("h", CommandRecord.Failed, func(i Invocation) bool { return i.Name == "grep" }) {
		t.Fatal("the failed grep should be found as a failure")
	}
}

func TestTrackerHasVisitedDir(t *testing.T) {
	tr := NewCommandTracker()
	tr.Record(CommandRecord{Hostname: "host", Cwd: []string{"home", "user"}, Command: "ls"})
	if !tr.HasVisitedDir("host", []string{"home", "user"}) {
		t.Fatal("should find visited dir")
	}
	if tr.HasVisitedDir("host", []string{"home"}) {
		t.Fatal("should not match partial path")
	}
}

// --- MachineRegistry tests ---

func minimalFS() *file.F {
	return &file.F{
		Type: file.Folder, Owner: file.OwnerRoot,
		OwnerPermission: file.Write, CommonPermission: file.Read,
		Files: map[string]*file.F{
			"etc": {Type: file.Folder, Owner: file.OwnerRoot,
				OwnerPermission: file.Write, CommonPermission: file.Read,
				Files: map[string]*file.F{
					"hosts": {Type: file.Text, Owner: file.OwnerRoot,
						OwnerPermission: file.Write, CommonPermission: file.Read,
						Data: "local"},
				}},
		},
	}
}

func TestMachineRegistryBootInitial(t *testing.T) {
	sim := simulation.New()
	r := NewMachineRegistry([]MachineEntry{
		{Hostname: "local", Filesystem: minimalFS},
		{Hostname: "locked", Filesystem: minimalFS, UnlockedBy: "key"},
	}, "user")
	err := r.BootInitialMachines(sim)
	if err != nil {
		t.Fatal(err)
	}
	if !r.IsBooted("local") {
		t.Fatal("local should be booted")
	}
	if r.IsBooted("locked") {
		t.Fatal("locked should not be booted yet")
	}
}

func TestMachineRegistryCheckUnlocks(t *testing.T) {
	sim := simulation.New()
	r := NewMachineRegistry([]MachineEntry{
		{Hostname: "local", Filesystem: minimalFS},
		{Hostname: "locked", Filesystem: minimalFS, UnlockedBy: "key"},
	}, "user")
	r.BootInitialMachines(sim)

	ach := NewAchievementSet()
	r.CheckUnlocks(sim, ach) // no key yet
	if r.IsBooted("locked") {
		t.Fatal("should not unlock without achievement")
	}

	ach.Grant("key")
	r.CheckUnlocks(sim, ach)
	if !r.IsBooted("locked") {
		t.Fatal("should unlock after achievement")
	}
}

// --- Manager tests ---

type mockQuest struct {
	id         string
	machine    string
	required   []Achievement
	granted    []Achievement
	complete   bool
	setupRan   bool
	planTarget string
}

func (q *mockQuest) ID() string                                         { return q.id }
func (q *mockQuest) Description() string                                { return "mock quest" }
func (q *mockQuest) Machine() string                                    { return q.machine }
func (q *mockQuest) RequiredAchievements() []Achievement                { return q.required }
func (q *mockQuest) GrantedAchievements() []Achievement                 { return q.granted }
func (q *mockQuest) Setup(_ *simulation.S) error                        { q.setupRan = true; return nil }
func (q *mockQuest) IsComplete(_ *simulation.S, _ *CommandTracker) bool { return q.complete }
func (q *mockQuest) PlanNextCommand(PlanContext) string {
	return q.planTarget
}

func TestManagerActivatesFirstQuest(t *testing.T) {
	sim := simulation.New()
	r := NewMachineRegistry(nil, "user")
	q := &mockQuest{id: "q1", machine: "m"}
	m := NewManager([]Quest{q}, r)
	m.AfterCommand(sim)
	if m.ActiveQuest() == nil || m.ActiveQuest().ID() != "q1" {
		t.Fatal("first quest should be active")
	}
	if !q.setupRan {
		t.Fatal("setup should have run")
	}
}

func TestManagerCompletesAndAdvances(t *testing.T) {
	sim := simulation.New()
	r := NewMachineRegistry(nil, "user")
	q1 := &mockQuest{id: "q1", machine: "m", granted: []Achievement{"done-q1"}}
	q2 := &mockQuest{id: "q2", machine: "m", required: []Achievement{"done-q1"}}
	m := NewManager([]Quest{q1, q2}, r)

	m.AfterCommand(sim) // activates q1
	q1.complete = true
	m.AfterCommand(sim) // completes q1, activates q2

	if m.GetQuestState("q1") != QuestComplete {
		t.Fatal("q1 should be complete")
	}
	if m.ActiveQuest() == nil || m.ActiveQuest().ID() != "q2" {
		t.Fatal("q2 should be active")
	}
}

func TestManagerDoesNotActivateWithoutAchievements(t *testing.T) {
	sim := simulation.New()
	r := NewMachineRegistry(nil, "user")
	q := &mockQuest{id: "q1", machine: "m", required: []Achievement{"needed"}}
	m := NewManager([]Quest{q}, r)
	m.AfterCommand(sim)
	if m.ActiveQuest() != nil {
		t.Fatal("quest should not activate without required achievements")
	}
}

func TestManagerDialogOnActivation(t *testing.T) {
	sim := simulation.New()
	r := NewMachineRegistry(nil, "user")
	q := &mockQuest{id: "connect", machine: "m"}
	m := NewManager([]Quest{q}, r)
	m.AfterCommand(sim) // activates connect
	lines := m.Dialog.Drain()
	if len(lines) == 0 {
		t.Fatal("expected dialog on quest activation")
	}
}

func TestManagerDialogOnCompletion(t *testing.T) {
	sim := simulation.New()
	r := NewMachineRegistry(nil, "user")
	q := &mockQuest{id: "connect", machine: "m"}
	m := NewManager([]Quest{q}, r)
	m.AfterCommand(sim) // activates
	m.Dialog.Drain()    // clear activation dialog

	q.complete = true
	m.AfterCommand(sim) // completes
	lines := m.Dialog.Drain()
	if len(lines) == 0 {
		t.Fatal("expected dialog on quest completion")
	}
}
