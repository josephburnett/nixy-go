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
	// PlanNextCommand computes the target command from current state.
	// Returns empty string if the quest should be complete (or no action needed).
	PlanNextCommand(sim *simulation.S, tracker *CommandTracker, hostname string, cwd []string) string
}
