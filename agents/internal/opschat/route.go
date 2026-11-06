package opschat

import (
	"regexp"
	"strings"
)

const (
	KindSkip   = "skip"
	KindGreet  = "greet"
	KindHelp   = "help"
	KindNote   = "note"
	KindWork   = "work"
	KindOutbox = "outbox"
)

// Decision is how one inbound message is handled. Department is empty when KindWork needs Classify.
type Decision struct {
	Kind       string
	Department string
	Task       string
	Action     string
	Prefixed   bool
	Reply      string
}

var prefixCommands = []struct {
	re   *regexp.Regexp
	dept string
	act  string
}{
	{regexp.MustCompile(`^(?:/community|/cm)(?:@\w+)?\b\s*`), "community", ""},
	{regexp.MustCompile(`^/accounts(?:@\w+)?\b\s*`), "accounts", ""},
	{regexp.MustCompile(`^/ops(?:@\w+)?\b\s*`), "internal-ops", ""},
	{regexp.MustCompile(`^/validate(?:@\w+)?\b\s*`), "product-dev", ""},
	{regexp.MustCompile(`^/growth(?:@\w+)?\b\s*`), "growth", ""},
	{regexp.MustCompile(`^/crm(?:@\w+)?\b\s*`), "crm", ""},
	{regexp.MustCompile(`^/legal(?:@\w+)?\b\s*`), "legal", ""},
	{regexp.MustCompile(`^/seo(?:@\w+)?\b\s*`), "seo", ""},
	{regexp.MustCompile(`^(?:/today|/pm)(?:@\w+)?\b\s*`), "delivery", "today"},
	{regexp.MustCompile(`^/idea(?:@\w+)?\b\s*`), "delivery", "idea"},
	{regexp.MustCompile(`^/hr(?:@\w+)?\b\s*`), "hr", ""},
	{regexp.MustCompile(`^/file(?:@\w+)?\b\s*`), "files", ""},
}

var greetRe = regexp.MustCompile(`^(hi|hello|hey|yo|hola|good\s+(morning|afternoon|evening)|gm|morning)[\s!.]*$`)

// HelpText is /start and /help.
func HelpText(who string, prefixes []string) string {
	if who == "" {
		who = "you"
	}
	lines := []string{
		"Office bot for " + who + ".",
		"Say the work. I pick one desk and do it in this chat. Chat never Accepts — that stays on the desk.",
	}
	if len(prefixes) > 0 {
		lines = append(lines, "Shortcuts if you want: "+strings.Join(prefixes, ", ")+".")
	}
	return strings.Join(lines, "\n")
}

// Decide is deterministic routing. KindWork with empty Department means run Classify next.
func Decide(in Inbound, currentDesk string) Decision {
	if in.FromBot {
		return Decision{Kind: KindSkip}
	}
	if in.Group {
		return Decision{Kind: KindNote, Reply: "This office bot is 1:1. Student rooms use the Academy bot."}
	}
	text := strings.TrimSpace(in.Text)
	if helpCmd(text) {
		return Decision{Kind: KindHelp}
	}
	if act := outboxAction(text); act != "" {
		return Decision{Kind: KindOutbox, Action: act}
	}

	for _, cmd := range prefixCommands {
		if cmd.re.MatchString(text) {
			task := strings.TrimSpace(cmd.re.ReplaceAllString(text, ""))
			act := cmd.act
			if cmd.dept == "delivery" && act == "" {
				act = "today"
			}
			if task == "" {
				task = defaultTask(cmd.dept, in)
			}
			return Decision{Kind: KindWork, Department: cmd.dept, Task: task, Action: act, Prefixed: true}
		}
	}

	if in.HasDocument {
		task := text
		if task == "" {
			task = "Classify this file into a project directory from the catalog. Do not invent a folder. Do not write Drive."
		}
		return Decision{Kind: KindWork, Department: "files", Task: task}
	}
	if in.HasPhoto {
		task := text
		if task == "" {
			task = "Read this receipt or invoice photo. Extract vendor or customer, amount, currency, date, and whether it is already paid. Do not invent figures."
		}
		return Decision{Kind: KindWork, Department: "accounts", Task: task, Action: "receipt"}
	}

	if morningCard(in.ReplyTo) {
		task := text
		if task == "" {
			task = "Check in on the morning card."
		}
		return Decision{Kind: KindWork, Department: "delivery", Task: task, Action: "today"}
	}

	if greetRe.MatchString(strings.ToLower(text)) && currentDesk == "" {
		return Decision{Kind: KindGreet}
	}

	if currentDesk != "" {
		task := text
		if task == "" {
			task = defaultTask(currentDesk, in)
		}
		act := ""
		if currentDesk == "delivery" {
			act = "today"
		}
		return Decision{Kind: KindWork, Department: currentDesk, Task: task, Action: act}
	}

	task := text
	if task == "" {
		task = "Greet the person and ask how you can help."
	}
	return Decision{Kind: KindWork, Task: task}
}

func helpCmd(text string) bool {
	return regexp.MustCompile(`^/(start|help)(?:@\w+)?\b`).MatchString(text)
}

func outboxAction(text string) string {
	switch {
	case regexp.MustCompile(`^/approve(?:@\w+)?\b`).MatchString(text):
		return "send"
	case regexp.MustCompile(`^/(kill|stop)(?:@\w+)?\b`).MatchString(text):
		return "kill"
	default:
		return ""
	}
}

func morningCard(replyTo string) bool {
	s := strings.TrimSpace(replyTo)
	return strings.HasPrefix(s, "Today on the desk")
}

func defaultTask(dept string, in Inbound) string {
	switch dept {
	case "files":
		return "Classify this file into a project directory from the catalog. Do not invent a folder. Do not write Drive."
	case "accounts":
		if in.HasPhoto {
			return "Read this receipt or invoice photo. Extract vendor or customer, amount, currency, and date. Do not invent figures."
		}
		return "Read this as books work. Extract fields. Do not invent figures."
	case "delivery":
		return "Today's tasks and check-ins."
	case "community":
		return "Welcome this person and ask how you can help in the community."
	default:
		return "Greet the person and ask how you can help."
	}
}

// AllowedDepartments is what Classify may return.
func AllowedDepartments() []string {
	return []string{
		"delivery", "growth", "legal", "files", "accounts", "internal-ops",
		"crm", "product-dev", "seo", "community", "hr",
	}
}

func KnownDepartment(dept string) bool {
	dept = strings.TrimSpace(strings.ToLower(dept))
	for _, d := range AllowedDepartments() {
		if d == dept {
			return true
		}
	}
	return false
}
