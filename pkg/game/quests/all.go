package quests

import "github.com/josephburnett/nixy-go/pkg/game"

// All is the quest line, in the order it is played.
func All() []game.Quest {
	return []game.Quest{
		&Connect{},
		&Orientation{},
		&Inspection{},
		&Modification{},
		&Composition{},
		&Permissions{},
		&Lockdown{},
		&Identity{},
		&Notes{},
		&Manual{},
		&Counting{},
		&Interrupt{},
		&Processes{},
	}
}
