// Package acadchat is the Academy Telegram student assistant (not the office ops bot).
package acadchat

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/opschat"
)

const (
	KindSkip    = "skip"
	KindHelp    = "help"
	KindRefuse  = "refuse"
	KindOnboard = "onboard"
	KindLink    = "link"
	KindStart   = "start"
	KindInvite  = "invite"
	KindWork    = "work"
)

const DefaultWebhookSecret = "acadTelegram00001_acadTg000000trig01"
const DefaultWebhookURL = "https://workers.themobileprof.com/webhook/telegram-academy/webhook"

// Inbound is one Academy Telegram update.
type Inbound struct {
	ChatID    string
	FromID    string
	Username  string
	Name      string
	Text      string
	Group     bool
	FromBot   bool
	Join      bool
	JoinNames []string
	HasPhoto  bool
	MigrateTo string
}

type Decision struct {
	Kind    string
	Task    string
	Reply   string
	Email   string
	RefCode string
}

type tgUpdate struct {
	Message      *tgMessage    `json:"message"`
	MyChatMember *tgChatMember `json:"my_chat_member"`
}

type tgChatMember struct {
	Chat          tgChat `json:"chat"`
	NewChatMember *struct {
		Status string `json:"status"`
		User   tgUser `json:"user"`
	} `json:"new_chat_member"`
}

type tgMessage struct {
	From              tgUser   `json:"from"`
	Chat              tgChat   `json:"chat"`
	Text              string   `json:"text"`
	Caption           string   `json:"caption"`
	Photo             []any    `json:"photo"`
	NewChatMembers    []tgUser `json:"new_chat_members"`
	MigrateToChatID   int64    `json:"migrate_to_chat_id"`
	MigrateFromChatID int64    `json:"migrate_from_chat_id"`
}

type tgUser struct {
	ID       int64  `json:"id"`
	IsBot    bool   `json:"is_bot"`
	Name     string `json:"first_name"`
	Username string `json:"username"`
}

type tgChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

func ParseUpdate(raw []byte) (Inbound, bool) {
	var u tgUpdate
	if json.Unmarshal(raw, &u) != nil {
		return Inbound{}, false
	}
	if u.MyChatMember != nil && u.MyChatMember.NewChatMember != nil {
		st := strings.ToLower(u.MyChatMember.NewChatMember.Status)
		if st == "member" || st == "administrator" {
			ch := u.MyChatMember.Chat
			in := Inbound{
				ChatID:  strconv.FormatInt(ch.ID, 10),
				FromID:  strconv.FormatInt(u.MyChatMember.NewChatMember.User.ID, 10),
				Group:   ch.Type == "group" || ch.Type == "supergroup",
				Join:    true,
				FromBot: u.MyChatMember.NewChatMember.User.IsBot,
				Name:    u.MyChatMember.NewChatMember.User.Name,
			}
			return in, in.ChatID != ""
		}
		return Inbound{}, false
	}
	if u.Message == nil {
		return Inbound{}, false
	}
	m := u.Message
	in := Inbound{
		ChatID:   strconv.FormatInt(m.Chat.ID, 10),
		FromID:   strconv.FormatInt(m.From.ID, 10),
		Username: m.From.Username,
		Name:     m.From.Name,
		FromBot:  m.From.IsBot,
		Group:    m.Chat.Type == "group" || m.Chat.Type == "supergroup",
		Text:     strings.TrimSpace(m.Text),
		HasPhoto: len(m.Photo) > 0,
	}
	if in.Text == "" {
		in.Text = strings.TrimSpace(m.Caption)
	}
	for _, p := range m.NewChatMembers {
		if p.IsBot {
			continue
		}
		in.Join = true
		n := strings.TrimSpace(p.Name)
		if n == "" {
			n = strings.TrimSpace(p.Username)
		}
		if n != "" {
			in.JoinNames = append(in.JoinNames, n)
		}
		if in.FromID == strconv.FormatInt(m.From.ID, 10) && p.ID != 0 {
			in.FromID = strconv.FormatInt(p.ID, 10)
			in.Name = p.Name
			in.Username = p.Username
		}
	}
	if m.MigrateToChatID != 0 {
		in.MigrateTo = strconv.FormatInt(m.MigrateToChatID, 10)
	}
	if m.MigrateFromChatID != 0 && in.MigrateTo == "" {
		in.MigrateTo = in.ChatID
		in.ChatID = strconv.FormatInt(m.MigrateFromChatID, 10)
	}
	if in.FromBot && !in.Join {
		return in, in.MigrateTo != ""
	}
	if in.ChatID == "" {
		return in, false
	}
	if in.Text == "" && !in.Join && !in.HasPhoto && in.MigrateTo == "" {
		return in, false
	}
	return in, true
}

func HelpText() string {
	return strings.Join([]string{
		"Hi — I'm the Academy host in this room.",
		"Talk about your course. No prefix needed. /cm if you want to be sure I pick it up.",
		"Stuck on a workshop? Just say so — I will ask you what I need, one thing at a time, then tell you exactly what to type.",
		"I stay quiet when you are just chatting.",
		"Payment or your own billing: DM me the email on your LMS account (https://lms.themobileprof.com). I will not quote receipts in the group.",
		"Bring a friend: type /invite and I send you a link to share. Each morning I check in — a greeting, where you are in the course, or an open ask for lesson/workshop/quiz questions — then ₦500 airtime for whoever showed up most, even if nobody earned it yesterday.",
	}, "\n")
}

func GroupEmailText() string {
	return "Do not put your LMS email in the group. Open a private chat with this bot and send that address there. I will not quote receipts here."
}

func RefuseText() string {
	return "This room is for your course. Ask about a lesson, or DM me the email on your LMS account for payment. I cannot help with that command here."
}

var (
	helpRe   = regexp.MustCompile(`^/(help)(?:@\w+)?\b`)
	startRe  = regexp.MustCompile(`(?i)^/start(?:@\w+)?(?:\s+(\S+))?`)
	inviteRe = regexp.MustCompile(`(?i)^/invite(?:@\w+)?\b`)
	opsRe    = regexp.MustCompile(`^/(approve|kill|stop|accounts|ops|legal|hr|seo|growth|crm|validate|today|pm|idea|file)(?:@\w+)?\b`)
	cmRe     = regexp.MustCompile(`^(?:/community|/cm)(?:@\w+)?\b\s*`)
	emailRe  = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	payAskRe = regexp.MustCompile(`(?i)\b(pay|paid|payment|billing|invoice|receipt|checkout|enrol+?|enroll)\b`)
)

func Decide(in Inbound) Decision {
	if in.FromBot && !in.Join {
		return Decision{Kind: KindSkip}
	}
	if in.Join {
		who := strings.Join(in.JoinNames, ", ")
		if who == "" {
			who = strings.TrimSpace(in.Name)
		}
		if who == "" {
			who = "there"
		}
		return Decision{Kind: KindOnboard, Task: "Welcome " + who + " to the Academy Telegram room."}
	}
	text := strings.TrimSpace(in.Text)
	if m := startRe.FindStringSubmatch(text); m != nil {
		code := referralCodeOf(m[1])
		if in.Group {
			return Decision{Kind: KindHelp, Reply: HelpText()}
		}
		return Decision{Kind: KindStart, RefCode: code, Task: text}
	}
	if inviteRe.MatchString(text) {
		return Decision{Kind: KindInvite, Task: text}
	}
	if helpRe.MatchString(text) {
		return Decision{Kind: KindHelp, Reply: HelpText()}
	}
	if opsRe.MatchString(text) {
		return Decision{Kind: KindRefuse, Reply: RefuseText()}
	}
	task := strings.TrimSpace(cmRe.ReplaceAllString(text, ""))
	if in.Group {
		if firstEmail(task) != "" {
			return Decision{Kind: KindHelp, Reply: GroupEmailText()}
		}
	} else if em := firstEmail(task); em != "" {
		return Decision{Kind: KindLink, Email: em, Task: task}
	}
	if task == "" && in.HasPhoto {
		if in.Group {
			task = "Someone posted a photo in the group. If it looks like a receipt or payment, do not quote status — tell them to DM this bot the email on their LMS account. If it is a study screenshot, ask which course title and lesson."
		} else {
			task = "Student sent a photo. If it is a receipt, ask for the email on their LMS account. Do not invent paid/unpaid. If it is a study screenshot, ask which course title and lesson."
		}
	}
	if task == "" {
		task = "Welcome this person and ask which course and lesson they are on."
	}
	if LooksLikePayment(task) && in.Group {
		task += "\n(Payment asked in a group: do not quote billing status. Tell them to DM this bot the LMS email.)"
	}
	return Decision{Kind: KindWork, Task: task}
}

func referralCodeOf(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "r_")
	s = strings.TrimPrefix(s, "ref_")
	s = strings.TrimPrefix(s, "ref-")
	return strings.Trim(s, "/")
}

func firstEmail(s string) string {
	m := emailRe.FindString(s)
	return strings.ToLower(strings.TrimSpace(m))
}

func LooksLikePayment(text string) bool {
	return payAskRe.MatchString(text)
}

func ClipText(s string) string {
	return opschat.ClipText(s)
}
