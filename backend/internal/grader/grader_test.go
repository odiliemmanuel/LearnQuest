package grader

import (
	"testing"

	"github.com/learnquest/backend/internal/models"
)

func TestEval(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"6", 6, true},
		{"6 m", 6, true},
		{"30 m/s", 30, true},
		{"1e8", 1e8, true},
		{"1e8 Hz", 1e8, true},
		{"100000000", 1e8, true},
		{"3e8", 3e8, true},
		{"2.5", 2.5, true},
		{"3 x 10^8", 3e8, true},
		{"3×10^8", 3e8, true},
		{"sqrt(16)", 4, true},
		{"2^10", 1024, true},
		{"1/2", 0.5, true},
		{"(1+2)*3", 9, true},
		{"5e10", 5e10, true},
		{"pi", 3.141592653589793, true},
		{"", 0, false},
		{"abc", 0, false},
	}
	for _, c := range cases {
		got, ok := Eval(c.in)
		if ok != c.ok {
			t.Errorf("Eval(%q) ok=%v want %v", c.in, ok, c.ok)
			continue
		}
		if ok && !closeTo(got, c.want) {
			t.Errorf("Eval(%q)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestGradeNumericTolerance(t *testing.T) {
	cases := []struct {
		got, want string
		tol       float64
		ok        bool
	}{
		{"6", "6", 0.05, true},
		{"6.02", "6", 0.05, true},
		{"6 m", "6", 0.05, true},
		{"100000000", "1e8", 0.001, true},
		{"90000000", "1e8", 0.001, false}, // 10% off, tol is 0.001 (relative)
		{"510", "510", 0.5, true},
		{"514", "510", 0.5, true}, // relative tolerance: 510*0.5 = 255
		{"", "510", 0.5, false},
	}
	for _, c := range cases {
		if got := GradeNumeric(c.got, c.want, c.tol); got != c.ok {
			t.Errorf("GradeNumeric(%q,%q,%v)=%v want %v", c.got, c.want, c.tol, got, c.ok)
		}
	}
}

func TestGradeQuestionMCQ(t *testing.T) {
	optA := models.QuestionOption{ID: 1, IsCorrect: true}
	optB := models.QuestionOption{ID: 2}
	q := models.Question{Type: "MCQ", CorrectOptionID: &optA.ID, Marks: 3}
	correct, marks := GradeQuestion(q, &optA.ID, "", "")
	if !correct || marks != 3 {
		t.Errorf("correct option should be marked correct with full marks")
	}
	correct, marks = GradeQuestion(q, &optB.ID, "", "")
	if correct || marks != 0 {
		t.Errorf("wrong option should be marked incorrect with 0 marks")
	}
	correct, _ = GradeQuestion(q, nil, "", "")
	if correct {
		t.Errorf("nil option should be incorrect")
	}
}

func closeTo(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= 1e-9
}
