package grader

import (
	"math"
	"strconv"
	"strings"

	"github.com/learnquest/backend/internal/models"
)

const DefaultTolerance = 0.01

func GradeQuestion(q models.Question, selectedOptionID *uint, answer, working string) (bool, int) {
	var correct bool
	switch q.Type {
	case "MCQ":
		correct = selectedOptionID != nil && q.CorrectOptionID != nil && *selectedOptionID == *q.CorrectOptionID
	case "CALCULATION":
		correct = GradeNumeric(answer, q.CorrectValue, toleranceFor(q))
	default:
		correct = false
	}
	marks := 0
	if correct {
		marks = q.Marks
	}
	return correct, marks
}

func toleranceFor(q models.Question) float64 {
	if q.Tolerance > 0 {
		return q.Tolerance
	}
	return DefaultTolerance
}

func GradeNumeric(studentAnswer, expected string, tolerance float64) bool {
	got, ok := Eval(studentAnswer)
	if !ok {
		return false
	}
	want, ok := Eval(expected)
	if !ok {
		return false
	}
	diff := math.Abs(got - want)
	rel := math.Abs(want)
	allowed := tolerance
	if rel != 0 && tolerance > 0 {
		allowed = math.Max(allowed, rel*tolerance)
	}
	return diff <= allowed
}

// Eval evaluates a lenient numeric/expression string. Returns ok=false when
// the string cannot be evaluated as a number or expression. Units trailing the
// value (e.g. "6 m", "30 m/s") are tolerated and ignored.
func Eval(expr string) (float64, bool) {
	s := normalize(expr)
	if s == "" {
		return 0, false
	}
	// Try progressively shorter prefixes so trailing units/marks strip cleanly.
	for end := len(s); end > 0; end-- {
		tokens, ok := tokenize(s[:end])
		if !ok || len(tokens) == 0 {
			continue
		}
		p := &evalParser{tokens: tokens}
		val, err := p.parseExpr()
		if err != nil || p.pos != len(p.tokens) {
			continue
		}
		if math.IsNaN(val) || math.IsInf(val, 0) {
			continue
		}
		return val, true
	}
	return 0, false
}

func normalize(expr string) string {
	s := strings.ToLower(strings.TrimSpace(expr))
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "×", "*")
	s = strings.ReplaceAll(s, "÷", "/")
	s = strings.ReplaceAll(s, "−", "-")
	s = strings.ReplaceAll(s, "–", "-")
	s = strings.ReplaceAll(s, "π", "pi")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

type tokenKind int

const (
	tokNum tokenKind = iota
	tokIdent
	tokOp
	tokLParen
	tokRParen
)

type token struct {
	kind tokenKind
	val  string
	num  float64
}

func tokenize(s string) ([]token, bool) {
	var toks []token
	i := 0
	for i < len(s) {
		ch := s[i]
		switch {
		case ch >= '0' && ch <= '9' || ch == '.':
			j := i
			seenDot := false
			if ch == '.' {
				seenDot = true
			}
			j++
			for j < len(s) {
				d := s[j]
				if d >= '0' && d <= '9' {
					j++
					continue
				}
				if d == '.' && !seenDot {
					seenDot = true
					j++
					continue
				}
				if (d == 'e' || d == 'E') && j+1 < len(s) {
					k := j + 1
					if s[k] == '+' || s[k] == '-' {
						k++
					}
					if k < len(s) && s[k] >= '0' && s[k] <= '9' {
						j = k
						for j < len(s) && s[j] >= '0' && s[j] <= '9' {
							j++
						}
						continue
					}
				}
				break
			}
			numStr := s[i:j]
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return nil, false
			}
			toks = append(toks, token{kind: tokNum, num: n, val: numStr})
			i = j
		case ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '^' || ch == 'x' || ch == 'X':
			toks = append(toks, token{kind: tokOp, val: string(ch)})
			i++
		case isLetter(ch):
			j := i
			for j < len(s) && isLetter(s[j]) {
				j++
			}
			ident := s[i:j]
			if !isValidIdent(ident) {
				return nil, false
			}
			toks = append(toks, token{kind: tokIdent, val: ident})
			i = j
		case ch == '(':
			toks = append(toks, token{kind: tokLParen, val: "("})
			i++
		case ch == ')':
			toks = append(toks, token{kind: tokRParen, val: ")"})
			i++
		default:
			// unit letter or stray char: stop tokenizing; caller strips suffix later.
			return toks[:len(toks):len(toks)], true
		}
	}
	return toks, true
}

func isLetter(ch byte) bool {
	return ch >= 'a' && ch <= 'z'
}

func isValidIdent(id string) bool {
	switch id {
	case "pi", "e", "sqrt", "sin", "cos", "tan", "abs":
		return true
	}
	return false
}

type evalParser struct {
	tokens []token
	pos    int
}

func (p *evalParser) peek() (token, bool) {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos], true
	}
	return token{}, false
}

func (p *evalParser) parseExpr() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		t, ok := p.peek()
		if !ok {
			break
		}
		if t.kind == tokOp && (t.val == "+" || t.val == "-") {
			p.pos++
			right, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			if t.val == "+" {
				left += right
			} else {
				left -= right
			}
		} else {
			break
		}
	}
	return left, nil
}

func (p *evalParser) parseTerm() (float64, error) {
	left, err := p.parseFactor()
	if err != nil {
		return 0, err
	}
	for {
		t, ok := p.peek()
		if !ok {
			break
		}
		if t.kind == tokOp && (t.val == "*" || t.val == "/" || t.val == "x") {
			p.pos++
			right, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			if t.val == "/" {
				if right == 0 {
					return 0, errDivZero
				}
				left /= right
			} else {
				left *= right
			}
		} else {
			break
		}
	}
	return left, nil
}

func (p *evalParser) parseFactor() (float64, error) {
	t, ok := p.peek()
	if !ok {
		return 0, errUnexpected
	}
	if t.kind == tokOp && (t.val == "+" || t.val == "-") {
		p.pos++
		v, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		if t.val == "-" {
			return -v, nil
		}
		return v, nil
	}
	return p.parsePower()
}

func (p *evalParser) parsePower() (float64, error) {
	base, err := p.parsePrimary()
	if err != nil {
		return 0, err
	}
	t, ok := p.peek()
	if ok && t.kind == tokOp && t.val == "^" {
		p.pos++
		exp, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		return math.Pow(base, exp), nil
	}
	return base, nil
}

func (p *evalParser) parsePrimary() (float64, error) {
	t, ok := p.peek()
	if !ok {
		return 0, errUnexpected
	}
	switch t.kind {
	case tokNum:
		p.pos++
		return t.num, nil
	case tokIdent:
		switch t.val {
		case "pi":
			p.pos++
			return math.Pi, nil
		case "e":
			p.pos++
			// e token is base-e constant (not scientific notation marker)
			return math.E, nil
		case "sqrt", "sin", "cos", "tan", "abs":
			p.pos++
			open, ok := p.peek()
			if !ok || open.kind != tokLParen {
				return 0, errUnexpected
			}
			p.pos++
			arg, err := p.parseExpr()
			if err != nil {
				return 0, err
			}
			close, ok := p.peek()
			if !ok || close.kind != tokRParen {
				return 0, errUnexpected
			}
			p.pos++
			switch t.val {
			case "sqrt":
				return math.Sqrt(arg), nil
			case "sin":
				return math.Sin(arg), nil
			case "cos":
				return math.Cos(arg), nil
			case "tan":
				return math.Tan(arg), nil
			case "abs":
				return math.Abs(arg), nil
			}
		}
		return 0, errUnexpected
	case tokLParen:
		p.pos++
		v, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		close, ok := p.peek()
		if !ok || close.kind != tokRParen {
			return 0, errUnexpected
		}
		p.pos++
		return v, nil
	}
	return 0, errUnexpected
}

var errUnexpected = errSyntax("unexpected token")
var errDivZero = errSyntax("division by zero")

type errSyntax string

func (e errSyntax) Error() string { return string(e) }
