package journeys

import (
	"fmt"
	"strings"
)

// Journey is one motion from idea to PMF. Gates are physics; the agent only places.
type Journey struct {
	ID    string
	Name  string
	Help  string
	Gates []Gate
}

// Gate is one proof on a journey. DoneWhen is behavioural, not a feeling.
type Gate struct {
	ID       string
	Label    string
	Stage    string
	DoneWhen string
	Mission  string
	Workers  []string
	Parent   bool
}

func All() []Journey {
	return []Journey{
		{
			ID:   "consumer_subscription",
			Name: "Consumer subscription",
			Help: "B2C app. PMF is a paid plan that still comes back next month.",
			Gates: []Gate{
				{ID: "install", Label: "Install", Stage: "testing", DoneWhen: "A real person (not staff) created an account or installed the app.", Mission: "Get five non-staff installs and watch whether they finish onboarding.", Workers: []string{"/validate", "/growth"}},
				{ID: "weekly_active", Label: "Weekly active", Stage: "testing", DoneWhen: "Someone chats or opens the calendar in a real week, unprompted.", Mission: "Pick last week’s installs. Count who came back without a founder nudge.", Workers: []string{"/validate", "/growth"}},
				{ID: "paid_conversion", Label: "Paid conversion", Stage: "selling", DoneWhen: "At least one paid plan that is not a staff or family account.", Mission: "Put the paid plan in front of actives. Record who pays and who refunds.", Workers: []string{"/validate", "/growth", "/crm"}, Parent: true},
				{ID: "retained_paid", Label: "Retained paid", Stage: "selling", DoneWhen: "A paid account is still active about 30 days later.", Mission: "Follow the paying accounts. Did they chat this month?", Workers: []string{"/validate", "/crm"}, Parent: true},
				{ID: "pmf", Label: "PMF", Stage: "fundable", DoneWhen: "Repeat paid retention you can point at, not a launch spike.", Mission: "Write the three paying accounts and whether they would miss the product.", Workers: []string{"/validate", "/crm"}, Parent: true},
			},
		},
		{
			ID:   "academy",
			Name: "Academy catalog",
			Help: "Courses. PMF is repeating paid enrollments, not a free micro binge.",
			Gates: []Gate{
				{ID: "catalog", Label: "Catalog live", Stage: "testing", DoneWhen: "Micro, mini, and professional paths are publicly enumerable.", Mission: "Walk the catalog as a stranger. Note what is missing or unpriced.", Workers: []string{"/validate", "/growth"}},
				{ID: "free_enroll", Label: "Free enroll", Stage: "testing", DoneWhen: "Unpaid enrollments happen without you chasing.", Mission: "Count free enrollments this week that you did not personally invite.", Workers: []string{"/validate", "/growth"}},
				{ID: "paid_professional", Label: "Paid path", Stage: "selling", DoneWhen: "Someone pays for a mini or professional path.", Mission: "Ask last week’s free learners to pay, or find who already did. No invented receipts.", Workers: []string{"/validate", "/growth", "/crm"}, Parent: true},
				{ID: "completion", Label: "Completion", Stage: "selling", DoneWhen: "A paid learner finishes, not just enrolls.", Mission: "Pull one paid learner. Did they complete? If not, where did they stop?", Workers: []string{"/validate", "/crm"}, Parent: true},
				{ID: "repeat_pay", Label: "Repeat pay", Stage: "fundable", DoneWhen: "A second paid enrollment (same or new person) without a founder push.", Mission: "List paid enrollments. Is there a second one you did not personally close?", Workers: []string{"/validate", "/crm"}, Parent: true},
			},
		},
		{
			ID:   "invited_aum",
			Name: "Invited AUM",
			Help: "Advisor-run book. PMF is money that stays after the advisor leaves the room.",
			Gates: []Gate{
				{ID: "advisor_running", Label: "Advisor running", Stage: "testing", DoneWhen: "One advisor completed a real assessment in the product.", Mission: "Sit with the BDM. One live assessment, no demo client.", Workers: []string{"/validate"}},
				{ID: "first_funded", Label: "First funded", Stage: "testing", DoneWhen: "Real money in at least one sleeve — not a landing-page screenshot.", Mission: "Confirm a funded sleeve with a bank or broker record. No invented AUM.", Workers: []string{"/validate", "/crm"}},
				{ID: "three_books", Label: "Three books", Stage: "selling", DoneWhen: "Three funded books that have not unwound.", Mission: "Count funded books. Which ones left? Why?", Workers: []string{"/validate", "/crm"}, Parent: true},
				{ID: "pmf", Label: "PMF", Stage: "fundable", DoneWhen: "AUM that stays after the advisor is not on the call.", Mission: "Pick one book. Would it still be there if the BDM went quiet for a month?", Workers: []string{"/validate", "/crm"}, Parent: true},
			},
		},
		{
			ID:   "mortgage_enablement",
			Name: "Mortgage enablement",
			Help: "Who pays is still a hypothesis. PMF is a fee for a completed file.",
			Gates: []Gate{
				{ID: "salary_checks", Label: "Salary checks", Stage: "testing", DoneWhen: "Non-staff salary-account eligibility checks actually ran.", Mission: "Count real eligibility checks this week. Staff and founder do not count.", Workers: []string{"/validate", "/growth"}},
				{ID: "advisor_file", Label: "Advisor file", Stage: "testing", DoneWhen: "One advisor-assisted file with documents, not a calculator toy.", Mission: "Take one checker through documents with an advisor. Write what blocked them.", Workers: []string{"/validate", "/crm"}},
				{ID: "who_pays", Label: "Who pays", Stage: "testing", DoneWhen: "A written bet: buyer fee vs lender vs advisor — plus evidence for one.", Mission: "Ask one completed-file person who would pay. Do not invent a business model.", Workers: []string{"/validate"}},
				{ID: "first_fee", Label: "First fee", Stage: "selling", DoneWhen: "Someone paid TheMobileProf for a completed file.", Mission: "Collect the first fee or write why they refused. Invoice as TMP.", Workers: []string{"/validate", "/crm"}, Parent: true},
				{ID: "pmf", Label: "PMF", Stage: "fundable", DoneWhen: "Repeat paid files without you in every thread.", Mission: "Second paid file. Who sourced it?", Workers: []string{"/validate", "/crm"}, Parent: true},
			},
		},
		{
			ID:   "shop_ledger",
			Name: "Shop-floor ledger",
			Help: "B2B bay tool. PMF is a second shop that runs closeouts without you, then pays.",
			Gates: []Gate{
				{ID: "design_partner", Label: "Design partner", Stage: "testing", DoneWhen: "One shop is registered. You may still be in the bay.", Mission: "Get one workshop onto the installer. Log the first real job.", Workers: []string{"/validate"}},
				{ID: "vin_closeouts", Label: "VIN closeouts", Stage: "testing", DoneWhen: "Several closeouts on real VINs at that shop.", Mission: "Count closeouts this week. No mock ECU as evidence.", Workers: []string{"/validate"}},
				{ID: "second_shop", Label: "Second shop", Stage: "testing", DoneWhen: "A second shop runs a job without you on the floor.", Mission: "Hand the installer to a second shop. Do not stay to operate it.", Workers: []string{"/validate", "/growth"}},
				{ID: "shop_pays", Label: "Shop pays", Stage: "selling", DoneWhen: "A shop pays or commits to pay TMP.", Mission: "Invoice the design partner as TheMobileProf, or write the refusal.", Workers: []string{"/validate", "/crm"}, Parent: true},
				{ID: "pmf", Label: "PMF", Stage: "fundable", DoneWhen: "A second paying shop, or the first renews, without you on the bay.", Mission: "Name the paying shops. Would they pay again if you vanished?", Workers: []string{"/validate", "/crm"}, Parent: true},
			},
		},
	}
}

func Lookup(id string) (Journey, bool) {
	id = strings.TrimSpace(id)
	for _, j := range All() {
		if j.ID == id {
			return j, true
		}
	}
	return Journey{}, false
}

func LookupGate(journeyID, gateID string) (Gate, bool) {
	j, ok := Lookup(journeyID)
	if !ok {
		return Gate{}, false
	}
	gateID = strings.TrimSpace(gateID)
	for _, g := range j.Gates {
		if g.ID == gateID {
			return g, true
		}
	}
	return Gate{}, false
}

func ValidPlacement(journeyID, gateID string) error {
	if strings.TrimSpace(journeyID) == "" && strings.TrimSpace(gateID) == "" {
		return nil
	}
	if strings.TrimSpace(journeyID) == "" || strings.TrimSpace(gateID) == "" {
		return fmt.Errorf("journey and gate must be set together")
	}
	if _, ok := Lookup(journeyID); !ok {
		return fmt.Errorf("unknown journey %q", journeyID)
	}
	if _, ok := LookupGate(journeyID, gateID); !ok {
		return fmt.Errorf("unknown gate %q on %s", gateID, journeyID)
	}
	return nil
}

// SeedPlacement is the committed starting stamp for this office's known bets.
func SeedPlacement(slug string) (journey, gate string) {
	switch strings.TrimSpace(slug) {
	case "momlaunchpad":
		return "consumer_subscription", "paid_conversion"
	case "academy":
		return "academy", "paid_professional"
	case "finchest":
		return "invited_aum", "first_funded"
	case "homegauge":
		return "mortgage_enablement", "salary_checks"
	case "mechazone":
		return "shop_ledger", "design_partner"
	default:
		return "", ""
	}
}

// PromptBlock is the catalog the placement worker may choose from. IDs only.
func PromptBlock() string {
	var b strings.Builder
	b.WriteString("You may only choose journey and gate IDs from this catalog. Do not invent IDs or new journeys.\n")
	for _, j := range All() {
		b.WriteString("\nJourney ")
		b.WriteString(j.ID)
		b.WriteString(" — ")
		b.WriteString(j.Name)
		b.WriteString(": ")
		b.WriteString(j.Help)
		b.WriteByte('\n')
		for _, g := range j.Gates {
			fmt.Fprintf(&b, "  gate %s [%s] %s. Done when: %s Default mission: %s\n", g.ID, g.Stage, g.Label, g.DoneWhen, g.Mission)
		}
	}
	return b.String()
}
