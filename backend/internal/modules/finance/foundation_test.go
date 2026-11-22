package finance

import "testing"

func TestMonthlyCapacityFrequencyAndRounding(t *testing.T) {
	profile := financialProfile{
		Items: []profileItem{
			{"Weekly job", "fixed_income", "weekly", 101},
			{"Biweekly work", "variable_income", "biweekly", 101},
			{"Bonus", "irregular_income", "once", 600000},
			{"Yearly insurance", "fixed_expense", "annual", 121},
			{"Weekly travel", "essential_expense", "weekly", 101},
			{"Allowance", "discretionary", "monthly", 50},
		},
		VariableIncomePercent: 70,
		BufferPercent:         10,
		BufferMinimumPaise:    20,
	}
	if !profile.validate() {
		t.Fatal("expected valid profile")
	}
	got := profile.estimate()
	if got.FixedIncomePaise != 437 || got.VariableIncomePaise != 218 ||
		got.CountedVariablePaise != 152 || got.CommittedPaise != 449 ||
		got.BufferPaise != 45 || got.SafeSavingsPaise != 45 {
		t.Fatalf("unexpected capacity: %+v", got)
	}
	profile.BufferMinimumPaise = 1000
	if got := profile.estimate(); got.BufferPaise != 1000 || got.SafeSavingsPaise != 0 {
		t.Fatalf("expected the minimum buffer and a non-negative estimate: %+v", got)
	}
}
