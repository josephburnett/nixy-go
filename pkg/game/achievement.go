package game

type Achievement string

type AchievementSet struct {
	achieved map[Achievement]bool
}

func NewAchievementSet() *AchievementSet {
	return &AchievementSet{
		achieved: map[Achievement]bool{},
	}
}

func (a *AchievementSet) Grant(ach Achievement) {
	a.achieved[ach] = true
}

func (a *AchievementSet) Has(ach Achievement) bool {
	return a.achieved[ach]
}

func (a *AchievementSet) HasAll(achs []Achievement) bool {
	for _, ach := range achs {
		if !a.achieved[ach] {
			return false
		}
	}
	return true
}

func (a *AchievementSet) List() []Achievement {
	var out []Achievement
	for ach := range a.achieved {
		out = append(out, ach)
	}
	return out
}
