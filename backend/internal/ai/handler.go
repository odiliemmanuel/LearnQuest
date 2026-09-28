package ai

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/learnquest/backend/internal/middleware"
	"github.com/learnquest/backend/internal/models"
	"github.com/learnquest/backend/internal/response"
)

type Handler struct {
	Svc *Service
	rl  *rateLimiter
}

func NewHandler(svc *Service) *Handler {
	return &Handler{Svc: svc, rl: newRateLimiter(30, time.Minute)}
}

type analyzeMistakeInput struct {
	QuestionID    uint   `json:"questionId" binding:"required"`
	AnswerID      *uint  `json:"answerId"`
	StudentAnswer string `json:"studentAnswer"`
	Working       string `json:"working"`
}

func (h *Handler) AnalyzeMistake(c *gin.Context) {
	if !h.allow(c) {
		response.TooMany(c, "Too many AI requests. Please wait a moment.")
		return
	}
	var in analyzeMistakeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "questionId is required.")
		return
	}
	userID, _ := middleware.UserID(c)
	studentText := in.StudentAnswer
	// Prefer stored answer when answerId provided.
	if in.AnswerID != nil {
		var ans models.StudentAnswer
		if err := h.Svc.db.First(&ans, *in.AnswerID).Error; err == nil && ans.StudentID == userID {
			studentText = ans.StudentAnswer
			var q models.Question
			if h.Svc.db.Preload("Options").First(&q, ans.QuestionID).Error == nil && q.Type == "MCQ" && ans.SelectedOptionID != nil {
				for _, o := range q.Options {
					if o.ID == *ans.SelectedOptionID {
						studentText = o.Key + ". " + o.Text
						break
					}
				}
			}
			if in.Working == "" {
				in.Working = ans.Working
			}
		}
	}
	var q models.Question
	if err := h.Svc.db.First(&q, in.QuestionID).Error; err != nil {
		response.NotFound(c, "Question not found.")
		return
	}
	var topic models.Topic
	if err := h.Svc.db.First(&topic, q.TopicID).Error; err != nil {
		response.NotFound(c, "Topic not found.")
		return
	}
	ctx, cancel := timeoutCtx(c, h.Svc.cfg.AITimeout)
	defer cancel()
	result, analysis, err := h.Svc.AnalyzeMistake(ctx, userID, topic.ID, q.ID, in.AnswerID, studentText, in.Working)
	if err != nil {
		response.Internal(c, "AI analysis failed. Your result and correct working are still available.")
		return
	}
	c.JSON(http.StatusOK, envelope(result, analysis.Status, analysis.Error))
}

type analyzeWorkingInput struct {
	QuestionID uint   `json:"questionId" binding:"required"`
	AnswerID   *uint  `json:"answerId"`
	Working    string `json:"working" binding:"required,min=1"`
}

func (h *Handler) AnalyzeWorking(c *gin.Context) {
	if !h.allow(c) {
		response.TooMany(c, "Too many AI requests. Please wait a moment.")
		return
	}
	var in analyzeWorkingInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "questionId and working are required.")
		return
	}
	userID, _ := middleware.UserID(c)
	working := in.Working
	if in.AnswerID != nil {
		var ans models.StudentAnswer
		if h.Svc.db.First(&ans, *in.AnswerID).Error == nil && ans.StudentID == userID && working == "" {
			working = ans.Working
		}
	}
	var q models.Question
	if err := h.Svc.db.First(&q, in.QuestionID).Error; err != nil {
		response.NotFound(c, "Question not found.")
		return
	}
	var topic models.Topic
	if err := h.Svc.db.First(&topic, q.TopicID).Error; err != nil {
		response.NotFound(c, "Topic not found.")
		return
	}
	ctx, cancel := timeoutCtx(c, h.Svc.cfg.AITimeout)
	defer cancel()
	result, analysis, err := h.Svc.AnalyzeWorking(ctx, userID, topic.ID, q.ID, in.AnswerID, working)
	if err != nil {
		response.Internal(c, "AI analysis failed. Your correct working is still available.")
		return
	}
	c.JSON(http.StatusOK, envelope(result, analysis.Status, analysis.Error))
}

type explainInput struct {
	TopicID uint `json:"topicId" binding:"required"`
}

func (h *Handler) Explain(c *gin.Context) {
	if !h.allow(c) {
		response.TooMany(c, "Too many AI requests. Please wait a moment.")
		return
	}
	var in explainInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "topicId is required.")
		return
	}
	userID, _ := middleware.UserID(c)

	var cls string
	var profile models.StudentProfile
	if h.Svc.db.Preload("ClassLevel").Where("user_id = ?", userID).First(&profile).Error == nil && profile.ClassLevel != nil {
		cls = profile.ClassLevel.Name
	}

	ctx, cancel := timeoutCtx(c, h.Svc.cfg.AITimeout)
	defer cancel()
	result, analysis, err := h.Svc.Explain(ctx, userID, in.TopicID, cls)
	if err != nil {
		response.Internal(c, "AI explanation failed. Your lesson content is still available.")
		return
	}
	c.JSON(http.StatusOK, envelope(result, analysis.Status, analysis.Error))
}

type generatePracticeInput struct {
	TopicID  uint   `json:"topicId" binding:"required"`
	Weakness string `json:"weakness"`
	Count    int    `json:"count"`
}

func (h *Handler) GeneratePractice(c *gin.Context) {
	if !h.allow(c) {
		response.TooMany(c, "Too many AI requests. Please wait a moment.")
		return
	}
	var in generatePracticeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "topicId is required.")
		return
	}
	ctx, cancel := timeoutCtx(c, h.Svc.cfg.AITimeout)
	defer cancel()
	result, err := h.Svc.GeneratePractice(ctx, in.TopicID, in.Weakness, in.Count)
	if err != nil {
		response.OK(c, gin.H{"questions": []any{}, "status": "UNAVAILABLE"})
		return
	}
	response.OK(c, gin.H{"questions": result.Questions, "status": "SUCCESS"})
}

func envelope(data any, status, err string) gin.H {
	return gin.H{"success": true, "data": data, "aiStatus": status, "error": err}
}

func timeoutCtx(c *gin.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), d)
}

// — rate limiter —

type rateLimiter struct {
	mu   sync.Mutex
	max  int
	win  time.Duration
	hits map[string]*window
}

type window struct {
	start time.Time
	count int
}

func newRateLimiter(max int, win time.Duration) *rateLimiter {
	return &rateLimiter{max: max, win: win, hits: map[string]*window{}}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	w, ok := rl.hits[key]
	if !ok || now.Sub(w.start) > rl.win {
		rl.hits[key] = &window{start: now, count: 1}
		return true
	}
	w.count++
	return w.count <= rl.max
}

func (h *Handler) allow(c *gin.Context) bool {
	userID, _ := middleware.UserID(c)
	return h.rl.allow(strconv.FormatUint(uint64(userID), 10) + ":" + c.FullPath())
}
