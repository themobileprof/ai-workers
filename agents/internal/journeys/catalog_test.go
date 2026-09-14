package journeys

import (
	"strings"
	"testing"
)

func TestCatalogIDsUniqueAndValid(t *testing.T) {
	seenJ := map[string]bool{}
	for _, j := range All() {
		if j.ID == "" || seenJ[j.ID] {
			t.Fatalf("journey id %q", j.ID)
		}
		seenJ[j.ID] = true
		seenG := map[string]bool{}
		if len(j.Gates) == 0 {
			t.Fatalf("%s has no gates", j.ID)
		}
		for _, g := range j.Gates {
			if g.ID == "" || seenG[g.ID] {
				t.Fatalf("%s gate %q", j.ID, g.ID)
			}
			seenG[g.ID] = true
			switch g.Stage {
			case "idea", "testing", "selling", "fundable":
			default:
				t.Fatalf("%s/%s stage %s", j.ID, g.ID, g.Stage)
			}
			if g.DoneWhen == "" || g.Mission == "" {
				t.Fatalf("%s/%s missing done-when or mission", j.ID, g.ID)
			}
		}
	}
}

func TestValidPlacement(t *testing.T) {
	if err := ValidPlacement("", ""); err != nil {
		t.Fatal(err)
	}
	if err := ValidPlacement("consumer_subscription", ""); err == nil {
		t.Fatal("half placement")
	}
	if err := ValidPlacement("consumer_subscription", "paid_conversion"); err != nil {
		t.Fatal(err)
	}
	if err := ValidPlacement("consumer_subscription", "first_funded"); err == nil {
		t.Fatal("gate from another journey")
	}
}

func TestSeedPlacement(t *testing.T) {
	j, g := SeedPlacement("mechazone")
	if err := ValidPlacement(j, g); err != nil {
		t.Fatal(err)
	}
	if j != "shop_ledger" || g != "design_partner" {
		t.Fatalf("%s %s", j, g)
	}
}

func TestPromptBlockContainsIDs(t *testing.T) {
	block := PromptBlock()
	for _, s := range []string{"consumer_subscription", "paid_conversion", "invited_aum", "shop_ledger"} {
		if !strings.Contains(block, s) {
			t.Fatalf("prompt missing %s", s)
		}
	}
}
