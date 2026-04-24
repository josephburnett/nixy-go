package character

// AllDialog returns all of Nixy's dialog entries.
func AllDialog() []DialogEntry {
	return []DialogEntry{
		// Connect quest
		{OnQuestActivate, "connect",
			[]string{
				"Nixy: Hey there! I'm Nixy.",
				"Nixy: I could use some help, but first you need to connect to me.",
				"Nixy: Try: `ssh nixy`",
			}},
		{OnQuestComplete, "connect",
			[]string{
				"Nixy: You made it! Welcome to my system.",
			}},

		// Orientation quest
		{OnQuestActivate, "orientation",
			[]string{
				"Nixy: Great! Now let's look around.",
				"Nixy: Try `pwd` to see where you are, `ls` to look around, and `cd` to move.",
			}},
		{OnQuestComplete, "orientation",
			[]string{
				"Nixy: You're getting the hang of this!",
			}},

		// Inspection quest
		{OnQuestActivate, "inspection",
			[]string{
				"Nixy: I have some files that need attention.",
				"Nixy: Use `cat` to read them and `grep` to search through logs.",
				"Nixy: You might need to install grep first: `apt install grep`",
			}},
		{OnQuestComplete, "inspection",
			[]string{
				"Nixy: Nice detective work!",
			}},

		// Modification quest
		{OnQuestActivate, "modification",
			[]string{
				"Nixy: Time to clean up. There's some junk in my home directory.",
				"Nixy: Delete junk.txt and create important.txt.",
				"Nixy: You'll need to install `rm` and `touch` first.",
			}},
		{OnQuestComplete, "modification",
			[]string{
				"Nixy: Much better! My home feels tidier already.",
			}},

		// Composition quest
		{OnQuestActivate, "composition",
			[]string{
				"Nixy: Here's something cool — you can chain commands with `|`",
				"Nixy: Try using `ls | grep` to find a specific file in my projects folder.",
			}},
		{OnQuestComplete, "composition",
			[]string{
				"Nixy: Pipes are powerful! You're really getting this.",
				"Nixy: By the way, I've got a friend — a server that could use your help too.",
			}},

		// Permissions quest
		{OnQuestActivate, "permissions",
			[]string{
				"Nixy: The server needs a config file in /etc, but it's locked down.",
				"Nixy: You'll need `sudo` to create files there. Try: `sudo touch /etc/config`",
			}},
		{OnQuestComplete, "permissions",
			[]string{
				"Nixy: You did it! You're a real command line pro now.",
				"Nixy: Thanks for all your help. The systems are running smoothly.",
			}},
	}
}

// FindDialog returns dialog lines for a given trigger and quest.
func FindDialog(entries []DialogEntry, trigger DialogTrigger, questID string) []string {
	for _, e := range entries {
		if e.Trigger == trigger && e.QuestID == questID {
			return e.Lines
		}
	}
	return nil
}
