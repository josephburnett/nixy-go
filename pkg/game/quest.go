package game

import "github.com/josephburnett/nixy-go/pkg/simulation"

type QuestState int

const (
	QuestInactive QuestState = iota
	QuestActive
	QuestComplete
)

// Quest defines a game quest.
type Quest interface {
	// ID returns a unique quest identifier.
	ID() string
	// Description returns a human-readable description of the quest objective.
	Description() string
	// Machine returns which machine this quest is associated with.
	Machine() string
	// RequiredAchievements returns achievements needed to activate this quest.
	RequiredAchievements() []Achievement
	// GrantedAchievements returns achievements granted on completion.
	GrantedAchievements() []Achievement
	// Setup mutates simulation state when quest activates (called once).
	Setup(sim *simulation.S) error
	// IsComplete checks if quest completion condition is met.
	IsComplete(sim *simulation.S, tracker *CommandTracker) bool
	// PlanNextCommand computes the next command line toward the goal
	// from the world as it is now, so a plan the player has broken is
	// replaced by one that recovers. The plan must be typeable at an
	// empty prompt in the player's shell. Returns "" if no action is
	// needed (the quest is complete).
	PlanNextCommand(ctx PlanContext) string
}

// PlanContext is what a planner sees: the world, the player's history,
// and where the player's shell is.
type PlanContext struct {
	Sim      *simulation.S
	Tracker  *CommandTracker
	Hostname string
	Cwd      []string
	User     string
}

// Repairer is a quest that can put back what it needs from the world
// when the player has broken it (deleted or moved a file the quest is
// about). The manager calls Repair after every command while the quest
// is active. It returns what Nixy says about it, or nil when nothing
// needed repair.
type Repairer interface {
	Repair(sim *simulation.S, tracker *CommandTracker) []string
}
