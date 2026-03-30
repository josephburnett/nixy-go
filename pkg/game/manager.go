package game

import (
	"github.com/josephburnett/nixy-go/pkg/character"
	"github.com/josephburnett/nixy-go/pkg/simulation"
)

// Manager manages quest lifecycle, achievements, and machine unlocks.
type Manager struct {
	quests       []Quest
	states       map[string]QuestState
	active       string
	achievements *AchievementSet
	machines     *MachineRegistry
	Tracker      *CommandTracker
	Dialog       *character.DialogQueue
	dialogData   []character.DialogEntry
}

func NewManager(quests []Quest, machines *MachineRegistry) *Manager {
	states := map[string]QuestState{}
	for _, q := range quests {
		states[q.ID()] = QuestInactive
	}
	return &Manager{
		quests:       quests,
		states:       states,
		achievements: NewAchievementSet(),
		machines:     machines,
		Tracker:      NewCommandTracker(),
		Dialog:       character.NewDialogQueue(),
		dialogData:   character.AllDialog(),
	}
}

// ActiveQuest returns the currently active quest, or nil.
func (m *Manager) ActiveQuest() Quest {
	for _, q := range m.quests {
		if m.states[q.ID()] == QuestActive {
			return q
		}
	}
	return nil
}

// AfterCommand checks quest completion, grants achievements, activates next quest.
func (m *Manager) AfterCommand(sim *simulation.S) {
	// Check active quest completion
	active := m.ActiveQuest()
	if active != nil && active.IsComplete(sim, m.Tracker) {
		m.states[active.ID()] = QuestComplete
		// Trigger completion dialog
		lines := character.FindDialog(m.dialogData, character.OnQuestComplete, active.ID())
		if lines != nil {
			m.Dialog.Enqueue(lines)
		}
		for _, ach := range active.GrantedAchievements() {
			m.achievements.Grant(ach)
		}
		m.active = ""

		// Check machine unlocks
		m.machines.CheckUnlocks(sim, m.achievements)
	}

	// Activate next eligible quest
	if m.ActiveQuest() == nil {
		for _, q := range m.quests {
			if m.states[q.ID()] != QuestInactive {
				continue
			}
			if m.achievements.HasAll(q.RequiredAchievements()) {
				m.states[q.ID()] = QuestActive
				m.active = q.ID()
				q.Setup(sim)
				// Trigger activation dialog
				lines := character.FindDialog(m.dialogData, character.OnQuestActivate, q.ID())
				if lines != nil {
					m.Dialog.Enqueue(lines)
				}
				break
			}
		}
	}
}

// QuestState returns the state of a quest by ID.
func (m *Manager) GetQuestState(id string) QuestState {
	return m.states[id]
}

// Achievements returns the achievement set.
func (m *Manager) Achievements() *AchievementSet {
	return m.achievements
}

// Machines returns the machine registry.
func (m *Manager) Machines() *MachineRegistry {
	return m.machines
}

// Quests returns all quests.
func (m *Manager) Quests() []Quest {
	return m.quests
}
