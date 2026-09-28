package ai

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/learnquest/backend/internal/config"
	"github.com/learnquest/backend/internal/middleware"
	"github.com/learnquest/backend/internal/models"
	openai "github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
)

type Service struct {
	cfg    *config.Config
	db     *gorm.DB
	client *openai.Client
}

func New(cfg *config.Config, db *gorm.DB) *Service {
	s := &Service{cfg: cfg, db: db}
	if cfg.OpenAIKey != "" {
		s.client = openai.NewClient(cfg.OpenAIKey)
	}
	return s
}

func (s *Service) Enabled() bool { return s.client != nil }

type chatUsage struct{ prompt, completion int }

func (s *Service) chat(ctx context.Context, system, user string) (string, chatUsage, error) {
	if s.client == nil {
		return mockResponse(user), chatUsage{}, nil
	}
	req := openai.ChatCompletionRequest{
		Model: s.cfg.OpenAIModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: system},
			{Role: openai.ChatMessageRoleUser, Content: user},
		},
		Temperature: 0.3,
	}
	if !strings.Contains(s.cfg.OpenAIModel, "gpt-3.5") && !strings.Contains(s.cfg.OpenAIModel, "gpt-4o") && !strings.Contains(s.cfg.OpenAIModel, "o1") {
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
	}
	start := time.Now()
	resp, err := s.client.CreateChatCompletion(ctx, req)
	latency := time.Since(start)
	usage := chatUsage{prompt: resp.Usage.PromptTokens, completion: resp.Usage.CompletionTokens}
	if err != nil {
		log.Printf("ai: chat error model=%s latency=%s err=%v", s.cfg.OpenAIModel, latency, err)
		return "", usage, err
	}
	if len(resp.Choices) == 0 {
		return "", usage, errors.New("ai returned no choices")
	}
	log.Printf("ai: chat ok model=%s latency=%s prompt=%d completion=%d", s.cfg.OpenAIModel, latency.Round(time.Millisecond), usage.prompt, usage.completion)
	return resp.Choices[0].Message.Content, usage, nil
}

// buildContext retrieves the grounded curriculum context for a topic.
func (s *Service) buildContext(topicID uint) (string, error) {
	var topic models.Topic
	if err := s.db.First(&topic, topicID).Error; err != nil {
		return "", err
	}
	var subject models.Subject
	var classLevel models.ClassLevel
	var term models.Term
	s.db.First(&subject, topic.SubjectID)
	s.db.First(&classLevel, topic.ClassLevelID)
	s.db.First(&term, topic.TermID)
	var lesson models.Lesson
	s.db.Where("topic_id = ?", topicID).First(&lesson)

	var b strings.Builder
	b.WriteString("Curriculum context:\n")
	b.WriteString("Subject: " + subject.Name + "\n")
	b.WriteString("Class: " + classLevel.Name + "\n")
	b.WriteString("Term: " + term.Name + "\n")
	b.WriteString("Topic: " + topic.Name + "\n")
	if lesson.ID != 0 {
		b.WriteString("Lesson title: " + lesson.Title + "\n")
		if lesson.Introduction != "" {
			b.WriteString("Intro: " + lesson.Introduction + "\n")
		}
		if lesson.Explanation != "" {
			trunc := lesson.Explanation
			if len(trunc) > 1800 {
				trunc = trunc[:1800]
			}
			b.WriteString("Explanation: " + trunc + "\n")
		}
		for i, kp := range lesson.KeyPoints {
			if i < 8 {
				b.WriteString("- Key point: " + kp + "\n")
			}
		}
		for i, f := range lesson.Formulas {
			if i < 6 {
				b.WriteString("- Formula: " + f.Name + ": " + f.Expression + "\n")
			}
		}
	}
	return b.String(), nil
}

// traceID reads the request id from a context value set by middleware.
func traceID(ctx context.Context) string {
	if v, ok := ctx.Value(middleware.CtxRequest).(string); ok {
		return v
	}
	return ""
}

// mockResponse is returned when no OPENAI_API_KEY is configured. It is shaped
// like a mistake-analysis answer; consumers parse-and-fallback gracefully.
func mockResponse(user string) string {
	return `{"correct":false,"mistakeType":"review_needed","whatStudentDid":"Submitted the selected answer.","conceptUnderstood":"","explanation":"Detailed AI explanations are disabled in this environment. Review the lesson notes and re-practice this topic.","correctWorking":"See the teacher's correct working for this question.","recommendedAction":"Open the lesson, study the worked examples, then retake the assessment."}`
}
