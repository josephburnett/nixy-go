package character

// AllDialog returns all of Nixy's dialog entries.
func AllDialog() []DialogEntry {
	return []DialogEntry{
		// Connect quest
		{OnQuestActivate, "connect",
			[]string{
				"Hey there! I'm Nixy.",
				"I could use some help, but first you need to connect to me.",
				"Try: `ssh nixy`",
			}},
		{OnQuestComplete, "connect",
			[]string{
				"You made it! Welcome to my system.",
				"If you lose track, `nx quest` says what we're doing and `nx log` shows how far you've come.",
			}},
		// Orientation quest
		{OnQuestActivate, "orientation",
			[]string{
				"Great! Now let's look around.",
				"You start in your own home folder: the `~` in your prompt is short for it. `pwd` prints the full path.",
				"`ls /home` shows whose homes are here, and `cd` moves you. Come visit mine!",
			}},
		{OnQuestComplete, "orientation",
			[]string{
				"You're getting the hang of this!",
			}},
		// Inspection quest
		{OnQuestActivate, "inspection",
			[]string{
				"I have some files that need attention.",
				"`cat` prints a file. Start with `cat readme.txt` in my home.",
				"Then my system log, in /var/log. `grep error system.log` prints just the lines with \"error\" in them.",
			}},
		{OnQuestComplete, "inspection",
			[]string{
				"Nice detective work!",
			}},
		// Modification quest
		{OnQuestActivate, "modification",
			[]string{
				"Time to clean up. There's some junk in my home directory.",
				"Delete junk.txt with `rm` and create important.txt with `touch`.",
			}},
		{OnQuestComplete, "modification",
			[]string{
				"Much better! My home feels tidier already.",
			}},
		// Composition quest
		{OnQuestActivate, "composition",
			[]string{
				"Here's something cool: `|` sends one command's output into the next.",
				"My projects folder is a mess. In there, `ls | grep target` lists only the file I need.",
			}},
		{OnQuestComplete, "composition",
			[]string{
				"Pipes are powerful! You're really getting this.",
				"By the way, I've got a friend: a server that could use your help too.",
				"`exit` takes you back to your laptop, and from there `ssh server` gets you to it.",
			}},
		// Permissions quest
		{OnQuestActivate, "permissions",
			[]string{
				"The server needs a config file, /etc/config, but it's locked down.",
				"It doesn't even have `touch`. Try `apt install touch` and see what it says.",
				"Installing software and writing to /etc are root's jobs.",
				"`sudo` runs one command as root: `sudo apt install touch`, then `sudo touch /etc/config`.",
				"Careful: root can delete anything. If a command you need goes missing, `nx panic` puts every machine's commands back.",
			}},
		{OnQuestComplete, "permissions",
			[]string{
				"You did it! You're a real command line pro now.",
				"Thanks for all your help. The systems are running smoothly.",
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
