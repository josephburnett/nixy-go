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
				"You did it! The server's happy.",
				"Watching you work as root got me thinking about my own files...",
			}},
		// Lockdown quest
		{OnQuestActivate, "lockdown",
			[]string{
				"Anyone could change things in my home! So I've locked it: you can still look, but only I can change it.",
				"Deleting a file changes the folder it's in, so you can't delete in my home any more.",
				"I do still want old.txt gone. Try `rm old.txt` in my home and see.",
				"Then do it as root, like on the server: `sudo rm old.txt`.",
			}},
		{OnQuestComplete, "lockdown",
			[]string{
				"Locked, but not locked out: that's what `sudo` is for.",
			}},
		// Identity quest
		{OnQuestActivate, "identity",
			[]string{
				"Here's a question: who are you, as far as my computer knows?",
				"Your prompt starts with your username. `whoami` asks the system directly.",
				"Then try `sudo whoami`. Who does sudo make you?",
			}},
		{OnQuestComplete, "identity",
			[]string{
				"root! sudo runs one command as root, the user who may do anything.",
				"Your prompt still shows you: the shell is yours, only that one command was root's.",
			}},
		// Notes quest
		{OnQuestActivate, "notes",
			[]string{
				"My log says the disk is low and the network keeps timing out. Let's keep a to-do list, in your home here.",
				"`mkdir notes` makes a folder for it.",
				"In there, `echo` prints words, and `>` sends them into a file instead of the screen: `echo check the disk > todo.txt`.",
				"`>>` adds to the end of a file instead: `echo fix the network >> todo.txt`. Then `cat` it to check.",
			}},
		{OnQuestComplete, "notes",
			[]string{
				"A to-do list! Careful with `>`: it empties the file first. `>>` is the safe way to add.",
			}},
		// Manual quest
		{OnQuestActivate, "manual",
			[]string{
				"I'd like to know how many errors are in my log. `wc` counts things, but I don't have it.",
				"Install it the way you did on the server: `sudo apt install wc`.",
				"Nobody remembers every option, so commands come with a manual: `man wc` shows wc's.",
				"(The shell's own commands, like cd, aren't in /bin: `help` lists those.)",
			}},
		{OnQuestComplete, "manual",
			[]string{
				"So `wc -l` counts lines. When you're unsure what a command does, `man` it.",
			}},
		// Counting quest
		{OnQuestActivate, "counting",
			[]string{
				"Let's count. In /var/log, `wc -l system.log` counts every line in my log.",
				"To count only the errors, send grep's output into wc: `grep error system.log | wc -l`.",
			}},
		{OnQuestComplete, "counting",
			[]string{
				"Two errors. Small commands joined by pipes answer real questions: that's the Unix way.",
			}},
		// Interrupt quest
		{OnQuestActivate, "interrupt",
			[]string{
				"Some commands never stop on their own. `yes` prints y, over and over, forever.",
				"Install it (`sudo apt install yes`) and run it.",
				"When you've seen enough, press Ctrl+C: the `^C` key. It interrupts whatever is running.",
			}},
		{OnQuestComplete, "interrupt",
			[]string{
				"Stopped! When something won't end, `^C` is the first thing to try.",
			}},
		// Processes quest
		{OnQuestActivate, "processes",
			[]string{
				"Oh no. I started a `yes` hours ago to test something, and it's still running. It's slowing my whole machine down!",
				"`ps -e` lists every process here: its PID (process ID), who owns it, and its command.",
				"Find my yes, then stop it with `kill` and its PID. It's my process, though, not yours...",
			}},
		{OnQuestComplete, "processes",
			[]string{
				"My machine can breathe again! Processes belong to users, like files do, and only the owner or root may stop them.",
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
