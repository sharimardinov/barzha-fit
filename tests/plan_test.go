package tests

import (
	"strings"
	"testing"

	"barzhafit/backend/service"
)

func TestParseTrainingPlanWeekPlan(t *testing.T) {
	plan := `{"week_plan":[` +
		`{"day":1,"name":"Day 1","focus":"A","type":"train","items":["A1","A2"]},` +
		`{"day":2,"name":"Day 2","focus":"B","type":"train","items":["B1"]},` +
		`{"day":3,"name":"Day 3","focus":"C","type":"train","items":["C1"]},` +
		`{"day":4,"name":"Day 4","focus":"D","type":"train","items":["D1"]},` +
		`{"day":5,"name":"Day 5","focus":"E","type":"train","items":["E1"]},` +
		`{"day":6,"name":"Day 6","focus":"F","type":"train","items":["F1"]},` +
		`{"day":7,"name":"Day 7","focus":"G","type":"rest","items":["Rest"]}` +
		`],"comment":""}`

	tp, ok := service.ParseTrainingPlan(plan)
	if !ok {
		t.Fatal("ParseTrainingPlan failed for week_plan JSON")
	}
	if len(tp.Days) != 7 {
		t.Fatalf("expected 7 days, got %d", len(tp.Days))
	}
	if len(tp.Types) != 7 {
		t.Fatalf("expected 7 types, got %d", len(tp.Types))
	}
	if tp.Types[0] != "train" || tp.Types[6] != "rest" {
		t.Fatalf("unexpected types: %v", tp.Types)
	}
}

func TestSplitPlanByDaysFromJSON(t *testing.T) {
	plan := `{"week_plan":[` +
		`{"day":1,"name":"Day 1","focus":"A","type":"train","items":["A1"]},` +
		`{"day":2,"name":"Day 2","focus":"B","type":"train","items":["B1"]},` +
		`{"day":3,"name":"Day 3","focus":"C","type":"train","items":["C1"]},` +
		`{"day":4,"name":"Day 4","focus":"D","type":"train","items":["D1"]},` +
		`{"day":5,"name":"Day 5","focus":"E","type":"train","items":["E1"]},` +
		`{"day":6,"name":"Day 6","focus":"F","type":"train","items":["F1"]},` +
		`{"day":7,"name":"Day 7","focus":"G","type":"rest","items":["Rest"]}` +
		`],"comment":""}`

	days := service.SplitPlanByDays(plan)
	if len(days) != 7 {
		t.Fatalf("expected 7 days, got %d", len(days))
	}
	if days[1] == "" || days[7] == "" {
		t.Fatal("expected non-empty day blocks")
	}
}

func TestFormatPlanForDisplay(t *testing.T) {
	plan := `{"days":["Day A","Day B","Day C","Day D","Day E","Day F","Day G"],"comment":""}`
	text, ok := service.FormatPlanForDisplay(plan)
	if !ok {
		t.Fatal("FormatPlanForDisplay failed")
	}
	if !strings.Contains(text, "ДЕНЬ 1") || !strings.Contains(text, "ДЕНЬ 7") {
		t.Fatalf("unexpected display output: %q", text)
	}
}
