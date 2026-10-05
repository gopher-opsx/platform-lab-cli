package cli

import "testing"

func TestSelectChallengeScenarioOverride(t *testing.T) {
	got, err := selectChallengeScenario("postgres-down")
	if err != nil {
		t.Fatalf("selectChallengeScenario returned error: %v", err)
	}
	if got != "postgres-down" {
		t.Fatalf("got %q, want postgres-down", got)
	}
}

func TestSelectChallengeScenarioRejectsIneligibleScenario(t *testing.T) {
	if _, err := selectChallengeScenario("lost-persistence"); err == nil {
		t.Fatal("expected ineligible challenge scenario to be rejected")
	}
}

func TestSelectChallengeScenarioRandomChoiceIsEligible(t *testing.T) {
	got, err := selectChallengeScenario("")
	if err != nil {
		t.Fatalf("selectChallengeScenario returned error: %v", err)
	}

	for _, candidate := range challengeEligibleScenarios {
		if got == candidate {
			return
		}
	}

	t.Fatalf("random selection %q is not challenge eligible", got)
}
