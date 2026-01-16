package input

import "testing"

func TestParseIntInRange(t *testing.T) {
	tests := []struct {
		in     string
		min    int
		max    int
		ok     bool
		expect int
	}{
		{"42", 1, 100, true, 42},
		{" 7 ", 0, 10, true, 7},
		{"-1", 0, 10, false, 0},
		{"abc", 0, 10, false, 0},
		{"11", 0, 10, false, 0},
	}

	for _, tt := range tests {
		got, ok := ParseIntInRange(tt.in, tt.min, tt.max)
		if ok != tt.ok || (ok && got != tt.expect) {
			t.Fatalf("ParseIntInRange(%q) = (%d,%v), want (%d,%v)", tt.in, got, ok, tt.expect, tt.ok)
		}
	}
}

func TestParseFloatInRange(t *testing.T) {
	tests := []struct {
		in     string
		min    float64
		max    float64
		ok     bool
		expect float64
	}{
		{"82.5", 20, 400, true, 82.5},
		{"82,5", 20, 400, true, 82.5},
		{"19.9", 20, 400, false, 0},
		{"foo", 20, 400, false, 0},
	}

	for _, tt := range tests {
		got, ok := ParseFloatInRange(tt.in, tt.min, tt.max)
		if ok != tt.ok {
			t.Fatalf("ParseFloatInRange(%q) ok=%v, want %v", tt.in, ok, tt.ok)
		}
		if ok && got != tt.expect {
			t.Fatalf("ParseFloatInRange(%q) = %f, want %f", tt.in, got, tt.expect)
		}
	}
}

func TestParseSteps(t *testing.T) {
	if _, ok := ParseSteps("8500"); !ok {
		t.Fatal("ParseSteps valid input failed")
	}
	if _, ok := ParseSteps("-1"); ok {
		t.Fatal("ParseSteps negative should fail")
	}
}

func TestParseWeight(t *testing.T) {
	if v, ok := ParseWeight("82.5"); !ok || v != 82.5 {
		t.Fatalf("ParseWeight expected 82.5, got %f ok=%v", v, ok)
	}
	if _, ok := ParseWeight("401"); ok {
		t.Fatal("ParseWeight out of range should fail")
	}
}

func TestParseMonthArg(t *testing.T) {
	month, year, ok := ParseMonthArg("0124")
	if !ok || month != 1 || year != 2024 {
		t.Fatalf("ParseMonthArg expected 1/2024, got %d/%d ok=%v", month, year, ok)
	}
	if _, _, ok := ParseMonthArg("1399"); ok {
		t.Fatal("ParseMonthArg invalid month should fail")
	}
	if _, _, ok := ParseMonthArg("0x24"); ok {
		t.Fatal("ParseMonthArg invalid digits should fail")
	}
}
