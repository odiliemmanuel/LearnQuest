package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/learnquest/backend/internal/grader"
	"github.com/learnquest/backend/internal/models"
)

type MistakeFeedback struct {
	Correct           bool   `json:"correct"`
	MistakeType       string `json:"mistakeType"`
	WhatStudentDid    string `json:"whatStudentDid"`
	ConceptUnderstood string `json:"conceptUnderstood"`
	Explanation       string `json:"explanation"`
	CorrectWorking    string `json:"correctWorking"`
	RecommendedAction string `json:"recommendedAction"`
}

type WorkingStep struct {
	Label   string `json:"label"`
	Status  string `json:"status"`
	Comment string `json:"comment"`
}

type WorkingFeedback struct {
	Steps           []WorkingStep `json:"steps"`
	OverallFeedback string        `json:"overallFeedback"`
	CorrectSolution string        `json:"correctSolution"`
}

type ExplainFeedback struct {
	Title       string   `json:"title"`
	Explanation string   `json:"explanation"`
	KeyPoints   []string `json:"keyPoints"`
	Example     string   `json:"example"`
}

type PracticeOption struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

type PracticeQuestion struct {
	Prompt      string           `json:"prompt"`
	Options     []PracticeOption `json:"options"`
	CorrectKey  string           `json:"correctKey"`
	Explanation string           `json:"explanation"`
	Difficulty  int              `json:"difficulty"`
}

type PracticeFeedback struct {
	Questions []PracticeQuestion `json:"questions"`
}

const mistakeSystem = `You are LearnQuest's mistake analyst for Nigerian students (secondary school, uniform curriculum). You help a student understand exactly why their answer to a question was wrong, using only the provided curriculum context. Respond ONLY with valid JSON matching this exact schema: {"correct":boolean,"mistakeType":"...","whatStudentDid":"short description of what the student did","conceptUnderstood":"what the student actually understood","explanation":"clear class-appropriate explanation","correctWorking":"the correct steps/working","recommendedAction":"what to practice next"}. Never invent curriculum. Match the student's class level in language and depth.`

func (s *Service) promptMistake(topicID uint, question models.Question, studentText string, studentWorking string) (string, error) {
	ctxText, err := s.buildContext(topicID)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(ctxText)
	b.WriteString("\nQuestion: " + question.Prompt + "\n")
	switch question.Type {
	case "MCQ":
		b.WriteString("Options:\n")
		for _, o := range question.Options {
			b.WriteString("- " + o.Key + ". " + o.Text + "\n")
		}
		b.WriteString("Student selected: " + studentText + "\n")
		for _, o := range question.Options {
			if o.IsCorrect {
				b.WriteString("Correct option: " + o.Key + ". " + o.Text + "\n")
			}
		}
	default:
		unit := ""
		if question.CorrectUnit != "" {
			unit = " " + question.CorrectUnit
		}
		b.WriteString("Correct answer: " + question.CorrectValue + unit + "\n")
		b.WriteString("Student answer: " + studentText + "\n")
		if studentWorking != "" {
			b.WriteString("Student working:\n" + studentWorking + "\n")
		}
	}
	b.WriteString("Teacher explanation: " + question.Explanation + "\n")
	return b.String(), nil
}

func (s *Service) AnalyzeMistake(ctx context.Context, studentID, topicID, questionID uint, answerID *uint, studentText, working string) (*MistakeFeedback, *models.AIAnalysis, error) {
	var question models.Question
	if err := s.db.Preload("Options").First(&question, questionID).Error; err != nil {
		return nil, nil, err
	}
	prompt, err := s.promptMistake(topicID, question, studentText, working)
	if err != nil {
		return nil, nil, err
	}
	raw, _, err := s.chat(ctx, mistakeSystem, prompt)
	result := &MistakeFeedback{}
	status := "SUCCESS"
	var parseErr error
	if err != nil {
		status = "FAILED"
		parseErr = err
	} else if err := json.Unmarshal([]byte(raw), result); err != nil || !validMistake(result) {
		status = "FALLBACK"
		result = fallbackMistake(question, studentText, parseCorrect(question, studentText))
		parseErr = err
	}

	analysis := &models.AIAnalysis{
		StudentID:   studentID,
		QuestionID:  questionID,
		AnswerID:    answerID,
		Kind:        "MISTAKE",
		Status:      status,
		RequestJSON: prompt,
		Error:       errString(parseErr),
		CreatedAt:   time.Now(),
	}
	if raw != "" {
		analysis.ResponseJSON = raw
	}
	// Keep only the most recent analysis for this answer.
	if answerID != nil {
		s.db.Where("answer_id = ? AND kind = ?", *answerID, "MISTAKE").Delete(&models.AIAnalysis{})
	} else {
		s.db.Where("student_id = ? AND question_id = ? AND kind = ?", studentID, questionID, "MISTAKE").
			Delete(&models.AIAnalysis{})
	}
	if err := s.db.Create(analysis).Error; err != nil {
		return nil, nil, err
	}
	return result, analysis, nil
}

func validMistake(f *MistakeFeedback) bool {
	return f.Explanation != "" && f.CorrectWorking != "" && f.MistakeType != ""
}

func fallbackMistake(q models.Question, studentText string, correct bool) *MistakeFeedback {
	mistakeType := "incorrect_answer"
	if q.Type == "CALCULATION" {
		mistakeType = "calculation_error"
	}
	mf := &MistakeFeedback{
		Correct:           correct,
		MistakeType:       mistakeType,
		WhatStudentDid:    studentText,
		ConceptUnderstood: "The formula and expected method are still being practiced.",
		Explanation:       q.Explanation,
		CorrectWorking:    q.ExpectedWorking,
		RecommendedAction: "Review the lesson, then retry this question and similar practice.",
	}
	return mf
}

func parseCorrect(q models.Question, studentText string) bool {
	if q.Type == "CALCULATION" {
		got, ok := grader.Eval(studentText)
		want, ok2 := grader.Eval(q.CorrectValue)
		return ok && ok2 && want != 0 && abs(got-want) <= 0.01
	}
	return false
}

const workingSystem = `You are LearnQuest's working-step analyst. A student submitted written working for a calculation question. Inspect each line/step they wrote: did they use the right formula, substitute values correctly, calculate correctly, and give the correct unit? Respond ONLY with valid JSON: {"steps":[{"label":"step name","status":"correct|incorrect|partial","comment":"one short sentence"}],"overallFeedback":"short paragraph","correctSolution":"the fully worked correct solution"}. Keep language at the student's class level.`

func (s *Service) AnalyzeWorking(ctx context.Context, studentID, topicID, questionID uint, answerID *uint, working string) (*WorkingFeedback, *models.AIAnalysis, error) {
	var question models.Question
	if err := s.db.First(&question, questionID).Error; err != nil {
		return nil, nil, err
	}
	ctxText, err := s.buildContext(topicID)
	if err != nil {
		return nil, nil, err
	}
	prompt := fmt.Sprintf("%s\nQuestion: %s\nCorrect answer: %s %s\nCorrect working:\n%s\n\nStudent working:\n%s",
		ctxText, question.Prompt, question.CorrectValue, question.CorrectUnit, question.ExpectedWorking, working)

	raw, _, err := s.chat(ctx, workingSystem, prompt)
	result := &WorkingFeedback{}
	status := "SUCCESS"
	var parseErr error
	if err != nil {
		status = "FAILED"
		parseErr = err
		result = &WorkingFeedback{
			Steps:           []WorkingStep{{Label: "Working", Status: "partial", Comment: "See the correct working below."}},
			OverallFeedback: "AI analysis is temporarily unavailable.",
			CorrectSolution: question.ExpectedWorking,
		}
	} else if err := json.Unmarshal([]byte(raw), result); err != nil || len(result.Steps) == 0 {
		status = "FALLBACK"
		parseErr = err
		result = &WorkingFeedback{
			Steps:           []WorkingStep{{Label: "Working", Status: "partial", Comment: "Review each step against the correct working."}},
			OverallFeedback: "Step analysis unavailable; check the correct working below.",
			CorrectSolution: question.ExpectedWorking,
		}
	}
	analysis := &models.AIAnalysis{
		StudentID:    studentID,
		QuestionID:   questionID,
		AnswerID:     answerID,
		Kind:         "WORKING",
		Status:       status,
		RequestJSON:  prompt,
		ResponseJSON: raw,
		Error:        errString(parseErr),
		CreatedAt:    time.Now(),
	}
	if answerID != nil {
		s.db.Where("answer_id = ?", *answerID).Delete(&models.AIAnalysis{})
	}
	if err := s.db.Create(analysis).Error; err != nil {
		return nil, nil, err
	}
	return result, analysis, nil
}

const explainSystem = `You are LearnQuest's concept explainer for Nigerian secondary students. Explain the concept using only the provided curriculum context. Write at the student's class level, use plain language, short sentences, and one worked example. Respond ONLY with valid JSON: {"title":"...","explanation":"...","keyPoints":["..."],"example":"..."}.`

func (s *Service) Explain(ctx context.Context, studentID, topicID uint, classLevel string) (*ExplainFeedback, *models.AIAnalysis, error) {
	ctxText, err := s.buildContext(topicID)
	if err != nil {
		return nil, nil, err
	}
	prompt := fmt.Sprintf("Class level: %s\n%s\nExplain this topic clearly for the student.", classLevel, ctxText)
	raw, _, err := s.chat(ctx, explainSystem, prompt)
	result := &ExplainFeedback{}
	status := "SUCCESS"
	var parseErr error
	if err != nil {
		status = "FAILED"
		parseErr = err
		result = &ExplainFeedback{Title: "Concept", Explanation: "Explanation is temporarily unavailable.", KeyPoints: []string{}, Example: ""}
	} else if err := json.Unmarshal([]byte(raw), result); err != nil || result.Explanation == "" {
		status = "FALLBACK"
		parseErr = err
		var lesson models.Lesson
		if s.db.Where("topic_id = ?", topicID).First(&lesson).Error == nil {
			result = &ExplainFeedback{Title: lesson.Title, Explanation: lesson.Explanation, KeyPoints: lesson.KeyPoints}
		} else {
			result = &ExplainFeedback{Title: "Concept", Explanation: "Explanation is temporarily unavailable.", KeyPoints: []string{}}
		}
	}
	analysis := &models.AIAnalysis{
		StudentID:    studentID,
		QuestionID:   0,
		Kind:         "EXPLANATION",
		Status:       status,
		RequestJSON:  prompt,
		ResponseJSON: raw,
		Error:        errString(parseErr),
		CreatedAt:    time.Now(),
	}
	if err := s.db.Create(analysis).Error; err != nil {
		return nil, nil, err
	}
	return result, analysis, nil
}

const practiceSystem = `You are LearnQuest's practice question generator for Nigerian secondary students. Create additional practice using ONLY the provided curriculum context, targeting the stated weakness. Respond ONLY with valid JSON: {"questions":[{"prompt":"...","options":[{"key":"A","text":"..."},{"key":"B","text":"..."},{"key":"C","text":"..."},{"key":"D","text":"..."}],"correctKey":"B","explanation":"short explanation","difficulty":1}]}. Generate valid, unambiguous, curriculum-aligned questions; do not invent facts.`

func (s *Service) GeneratePractice(ctx context.Context, topicID uint, weakness string, count int) (*PracticeFeedback, error) {
	if count <= 0 || count > 8 {
		count = 5
	}
	ctxText, err := s.buildContext(topicID)
	if err != nil {
		return nil, err
	}
	prompt := fmt.Sprintf("%s\nWeakness to target: %s\nGenerate %d NEW practice questions.\nImportant constraints so they are gradeable:\n  * Each question MUST have exactly 4 options (A-D).\n  * correctKey MUST equal exactly one of the option keys.\n  * For each question, give a short explanation.\n  * Prefer questions that clearly have ONE unambiguous correct option.", ctxText, weakness, count)
	raw, _, err := s.chat(ctx, practiceSystem, prompt)
	result := &PracticeFeedback{}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(raw), result); err != nil {
		return nil, err
	}
	// Validate each question is gradeable.
	cleaned := make([]PracticeQuestion, 0, len(result.Questions))
	for _, q := range result.Questions {
		if len(q.Options) != 4 || !hasExactlyOneCorrectKey(q) {
			continue
		}
		cleaned = append(cleaned, q)
	}
	result.Questions = cleaned
	return result, nil
}

func hasExactlyOneCorrectKey(q PracticeQuestion) bool {
	if q.CorrectKey == "" {
		return false
	}
	match := 0
	for _, o := range q.Options {
		if o.Key == "" || o.Text == "" {
			return false
		}
		if o.Key == q.CorrectKey {
			match++
		}
	}
	return match == 1
}

// FillPending processes PENDING mistake-analysis rows for a submitted quiz in
// the background. Never blocks the quiz request path and never touches the
// core quiz transaction.
func (s *Service) FillPending(ctx context.Context, quizID uint) {
	var pending []models.AIAnalysis
	if err := s.db.WithContext(ctx).
		Where("quiz_id = ? AND kind = ? AND status = ?", quizID, "MISTAKE", "PENDING").
		Order("id asc").Find(&pending).Error; err != nil {
		return
	}
	for _, p := range pending {
		s.fillOne(ctx, &p)
	}
}

func (s *Service) fillOne(ctx context.Context, p *models.AIAnalysis) {
	var answer models.StudentAnswer
	if p.AnswerID == nil {
		return
	}
	if err := s.db.WithContext(ctx).First(&answer, *p.AnswerID).Error; err != nil {
		return
	}
	var question models.Question
	if err := s.db.WithContext(ctx).Preload("Options").First(&question, p.QuestionID).Error; err != nil {
		return
	}
	var topic models.Topic
	if err := s.db.WithContext(ctx).First(&topic, question.TopicID).Error; err != nil {
		return
	}

	studentText := answer.StudentAnswer
	if question.Type == "MCQ" && answer.SelectedOptionID != nil {
		for _, o := range question.Options {
			if o.ID == *answer.SelectedOptionID {
				studentText = o.Key + ". " + o.Text
				break
			}
		}
	}
	prompt, err := s.promptMistake(topic.ID, question, studentText, answer.Working)
	if err != nil {
		return
	}
	raw, _, err := s.chat(ctx, mistakeSystem, prompt)
	mf := &MistakeFeedback{}
	if err != nil {
		p.Status = "FAILED"
		p.Error = err.Error()
	} else if err := json.Unmarshal([]byte(raw), mf); err != nil || !validMistake(mf) {
		p.Status = "FALLBACK"
		p.ResponseJSON = raw
		p.Error = "invalid structured response"
	} else {
		p.Status = "SUCCESS"
		p.ResponseJSON = raw
	}
	p.RequestJSON = prompt
	s.db.WithContext(ctx).Save(p)
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
