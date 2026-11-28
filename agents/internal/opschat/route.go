package opschat

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	KindSkip   = "skip"
	KindGreet  = "greet"
	KindHelp   = "help"
	KindNote   = "note"
	KindWork   = "work"
	KindOutbox = "outbox"
	KindStamp  = "stamp"
)

// Decision is how one inbound message is handled. Department is empty when KindWork needs Classify.
type Decision struct {
	Kind       string
	Department string
	Task       string
	Action     string
	Prefixed   bool
	Reply      string
	JobID      int64
	Reason     string
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
	{regexp.MustCompile(`^(?:/today|/pm)(?:@\w+)?\b\s*`), "delivery", "report"},
	{regexp.MustCompile(`^/idea(?:@\w+)?\b\s*`), "delivery", "idea"},
	{regexp.MustCompile(`^/hr(?:@\w+)?\b\s*`), "hr", ""},
	{regexp.MustCompile(`^/file(?:@\w+)?\b\s*`), "files", ""},
}

var (
	greetRe   = regexp.MustCompile(`^(hi|hello|hey|yo|hola|good\s+(morning|afternoon|evening)|gm|morning)[\s!.]*$`)
	seoGoneRe = regexp.MustCompile(`^/seo(?:@\w+)?\b`)
)

// HelpText is /start and /help.
func HelpText(who string, prefixes []string) string {
	if who == "" {
		who = "you"
	}
	lines := []string{
		"Office bot for " + who + ".",
		"Say what happened in your own words. I log the field report onto Delivery, CRM, Dates, and the Card, and I file spend as an Accounts job. You do not number tasks.",
		"Drop a PDF and I read it (text layer or pages), summarize, and file it — or answer your caption question on the right desk.",
		"I keep the bet you named sticky, pick the desk for plain English, and when an ask spans desks I list the order and run only the first.",
		"Accept, Place ads, mark ads mature, Decide stage, People, and Defaults stay on the desk — I send the link. If nothing live covers the ask, I say so and file a gap note.",
		"When a draft needs your yes, tap Accept or Turn away under that message. /yes also Accepts. /no invented Ikeja Turns away with a reason the next draft can see.",
		"The clerk never Accepts. /approve still only sends a pending info@ draft.",
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
	if seoGoneRe.MatchString(text) {
		return Decision{Kind: KindNote, Reply: "Onsite SEO is Cursor in the bet repo. Copy .cursor/rules/onsite-seo.mdc and traffic.mdc, keep TRAFFIC.md at the repo root, and walk cite → FAQ → schema → title/meta from the designed page. Offsite listings are Delivery tasks you add. Search Console and GA4 still land on Defaults. There is no SEO worker."}
	}
	if act := outboxAction(text); act != "" {
		return Decision{Kind: KindOutbox, Action: act}
	}
	if in.CallbackData != "" {
		if yes, id, ok := ParseJobCallback(in.CallbackData); ok {
			act := "rejected"
			if yes {
				act = "accepted"
			}
			return Decision{Kind: KindStamp, Action: act, JobID: id}
		}
		return Decision{Kind: KindNote, Reply: "That button is stale. Ask the desk again."}
	}
	if d, ok := parseStampCommand(text); ok {
		return d
	}

	for _, cmd := range prefixCommands {
		if cmd.re.MatchString(text) {
			task := strings.TrimSpace(cmd.re.ReplaceAllString(text, ""))
			act := cmd.act
			if cmd.dept == "delivery" && act == "" {
				act = "report"
			}
			if task == "" {
				task = defaultTask(cmd.dept, in)
			}
			return Decision{Kind: KindWork, Department: cmd.dept, Task: task, Action: act, Prefixed: true}
		}
	}

	if morningCard(in.ReplyTo) || LooksLikeReport(text) {
		task := text
		if task == "" {
			task = "Check in on the morning card."
		}
		return Decision{Kind: KindWork, Department: "delivery", Task: task, Action: "report"}
	}

	if in.HasDocument {
		task := text
		if task == "" {
			task = "Read this attached document. Summarize what it is in a few sentences, then classify it into a project directory from the catalog. Do not invent a folder. Do not write Drive."
			return Decision{Kind: KindWork, Department: "files", Task: task}
		}
		// Caption or question with a file: conductor picks the desk; PDF text is attached in opsRequest.
		task = "Attached file: " + strings.TrimSpace(in.Filename) + "\n\n" + task
		if strings.TrimSpace(in.Filename) == "" {
			task = text
		}
		return Decision{Kind: KindWork, Task: task}
	}
	if in.HasPhoto {
		task := text
		if task == "" {
			task = "Read this receipt or invoice photo. Extract vendor or customer, amount, currency, date, and whether it is already paid. Do not invent figures."
		}
		return Decision{Kind: KindWork, Department: "accounts", Task: task, Action: "receipt"}
	}

	if greetRe.MatchString(strings.ToLower(text)) && currentDesk == "" {
		return Decision{Kind: KindGreet}
	}

	if currentDesk != "" && !(currentDesk == "delivery" && text != "" && !LooksLikeReport(text)) {
		task := text
		if task == "" {
			task = defaultTask(currentDesk, in)
		}
		act := ""
		if currentDesk == "delivery" {
			act = "report"
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

func parseStampCommand(text string) (Decision, bool) {
	text = strings.TrimSpace(text)
	if regexp.MustCompile(`^/(yes|accept)(?:@\w+)?\b`).MatchString(text) {
		return Decision{Kind: KindStamp, Action: "accepted"}, true
	}
	if re := regexp.MustCompile(`^/(no|away|reject)(?:@\w+)?\b\s*`); re.MatchString(text) {
		reason := strings.TrimSpace(re.ReplaceAllString(text, ""))
		return Decision{Kind: KindStamp, Action: "rejected", Reason: reason}, true
	}
	return Decision{}, false
}

func ParseJobCallback(data string) (yes bool, jobID int64, ok bool) {
	data = strings.TrimSpace(data)
	parts := strings.Split(data, ":")
	if len(parts) != 3 || parts[0] != "j" {
		return false, 0, false
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || id <= 0 {
		return false, 0, false
	}
	switch parts[1] {
	case "y":
		return true, id, true
	case "n":
		return false, id, true
	default:
		return false, 0, false
	}
}

func JobCallback(yes bool, jobID int64) string {
	if jobID <= 0 {
		return ""
	}
	kind := "n"
	if yes {
		kind = "y"
	}
	return "j:" + kind + ":" + strconv.FormatInt(jobID, 10)
}

func JobKeyboard(jobID int64) [][]InlineButton {
	if jobID <= 0 {
		return nil
	}
	return [][]InlineButton{{
		{Text: "Accept", Data: JobCallback(true, jobID)},
		{Text: "Turn away", Data: JobCallback(false, jobID)},
	}}
}

func StampFooter() string {
	return "This is proposed. You Accept — the clerk does not."
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
		return "What happened on the bets. Log it in your own words."
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
		"crm", "product-dev", "community", "hr",
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
