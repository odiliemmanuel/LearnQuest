package quiz

import (
	"encoding/json"
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/learnquest/backend/internal/events"
	"github.com/learnquest/backend/internal/grader"
	"github.com/learnquest/backend/internal/middleware"
	"github.com/learnquest/backend/internal/models"
	"github.com/learnquest/backend/internal/question"
	"github.com/learnquest/backend/internal/response"
	"gorm.io/gorm"
)

const DefaultQuestionCount = 10

type Service struct {
	DB  *gorm.DB
	Bus *events.Broker
}

type StartInput struct {
	QuestionCount int  `json:"questionCount"`
	Recovery      bool `json:"recovery"`
}

type SubmitAnswerInput struct {
	QuestionID       uint   `json:"questionId" binding:"required"`
	SelectedOptionID *uint  `json:"selectedOptionId"`
	Answer           string `json:"answer"`
	Working          string `json:"working"`
}

type SubmitInput struct {
	Answers []SubmitAnswerInput `json:"answers" binding:"required"`
}

type QuizCompletedPayload struct {
	QuizID     uint    `json:"quizId"`
	StudentID  uint    `json:"studentId"`
	TopicID    uint    `json:"topicId"`
	IsRecovery bool    `json:"isRecovery"`
	Score      float64 `json:"score"`
	Correct    int     `json:"correct"`
	Total      int     `json:"total"`
}

func (s *Service) Start(c *gin.Context) {
	topicID, err := strconv.ParseUint(c.Param("topicId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid topic id.")
		return
	}
	userID, _ := middleware.UserID(c)
	var profile models.StudentProfile
	if err := s.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil || !profile.Onboarded() {
		response.BadRequest(c, "Please complete onboarding (select your class) before starting a quiz.")
		return
	}

	var topic models.Topic
	if err := s.DB.First(&topic, uint(topicID)).Error; err != nil {
		response.NotFound(c, "Topic not found.")
		return
	}

	var in StartInput
	_ = c.ShouldBindJSON(&in)

	var questions []models.Question
	if err := s.DB.Preload("Options", func(db *gorm.DB) *gorm.DB {
		return db.Order("question_options.\"order\" asc")
	}).Where("topic_id = ?", topic.ID).Order("\"questions\".\"id\" asc").Find(&questions).Error; err != nil {
		response.Internal(c, "Could not load questions.")
		return
	}
	if len(questions) == 0 {
		response.Fail(c, 400, "This topic does not have questions yet.")
		return
	}

	selected := questions
	if in.QuestionCount > 0 && in.QuestionCount < len(questions) {
		rand.Shuffle(len(questions), func(i, j int) { questions[i], questions[j] = questions[j], questions[i] })
		selected = questions[:in.QuestionCount]
	}

	totalMarks := 0
	for _, q := range selected {
		totalMarks += q.Marks
	}
	now := time.Now()
	quiz := models.Quiz{
		StudentID:  userID,
		TopicID:    topic.ID,
		Status:     "STARTED",
		IsRecovery: in.Recovery,
		TotalMarks: totalMarks,
		TotalCount: len(selected),
		StartedAt:  &now,
	}
	if err := s.DB.Create(&quiz).Error; err != nil {
		response.Internal(c, "Could not start quiz.")
		return
	}

	views := make([]question.PublicView, 0, len(selected))
	for _, q := range selected {
		views = append(views, question.ToPublic(q))
	}
	response.Created(c, gin.H{
		"quizId":     quiz.ID,
		"topic":      topic,
		"questions":  views,
		"totalMarks": totalMarks,
	})
}

func (s *Service) Submit(c *gin.Context) {
	quizID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid quiz id.")
		return
	}
	userID, _ := middleware.UserID(c)

	var quiz models.Quiz
	if err := s.DB.First(&quiz, uint(quizID)).Error; err != nil {
		response.NotFound(c, "Quiz not found.")
		return
	}
	if quiz.StudentID != userID {
		response.Forbidden(c, "This quiz does not belong to you.")
		return
	}
	if quiz.Status == "SUBMITTED" {
		response.Fail(c, 400, "This quiz has already been submitted.")
		return
	}

	var in SubmitInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Provide your answers.")
		return
	}
	if len(in.Answers) == 0 {
		response.BadRequest(c, "Provide at least one answer.")
		return
	}
	if len(in.Answers) != quiz.TotalCount {
		response.BadRequest(c, "Answer all questions before submitting.")
		return
	}

	// Load questions for this topic (source of truth for grading).
	var questions []models.Question
	if err := s.DB.Preload("Options").Where("topic_id = ?", quiz.TopicID).Find(&questions).Error; err != nil {
		response.Internal(c, "Could not load questions for grading.")
		return
	}
	qmap := map[uint]models.Question{}
	for _, q := range questions {
		qmap[q.ID] = q
	}

	tx := s.DB.Begin()
	scoreMarks := 0
	correctCount := 0
	for _, ans := range in.Answers {
		q, ok := qmap[ans.QuestionID]
		if !ok {
			tx.Rollback()
			response.BadRequest(c, "One or more answers do not belong to this quiz.")
			return
		}
		ok, marks := grader.GradeQuestion(q, ans.SelectedOptionID, ans.Answer, ans.Working)
		if ok {
			correctCount++
		}
		scoreMarks += marks
		sa := models.StudentAnswer{
			QuizID:           quiz.ID,
			StudentID:        userID,
			QuestionID:       q.ID,
			SelectedOptionID: ans.SelectedOptionID,
			StudentAnswer:    ans.Answer,
			Working:          ans.Working,
			IsCorrect:        ok,
			ReceivedMarks:    marks,
		}
		if err := tx.Create(&sa).Error; err != nil {
			tx.Rollback()
			response.Internal(c, "Could not save answers.")
			return
		}
		// If wrong, create a PENDING AI row so background workers can explain it.
		if !ok {
			tx.Create(&models.AIAnalysis{
				StudentID:  userID,
				QuestionID: q.ID,
				AnswerID:   &sa.ID,
				QuizID:     &quiz.ID,
				Kind:       "MISTAKE",
				Status:     "PENDING",
			})
		}
	}

	now := time.Now()
	score := 0.0
	if quiz.TotalMarks > 0 {
		score = float64(scoreMarks) / float64(quiz.TotalMarks) * 100
	}
	quiz.Status = "SUBMITTED"
	quiz.ScoreMarks = scoreMarks
	quiz.Score = score
	quiz.CorrectCount = correctCount
	quiz.SubmittedAt = &now
	if err := tx.Save(&quiz).Error; err != nil {
		tx.Rollback()
		response.Internal(c, "Could not finalize quiz.")
		return
	}
	if err := tx.Commit().Error; err != nil {
		response.Internal(c, "Could not save quiz result.")
		return
	}

	// Record activity + dispatch async post-quiz work (out of the request path).
	s.recordActivity(userID, quiz.TotalCount)

	s.Bus.Publish(events.QuizCompleted, QuizCompletedPayload{
		QuizID:     quiz.ID,
		StudentID:  userID,
		TopicID:    quiz.TopicID,
		IsRecovery: quiz.IsRecovery,
		Score:      score,
		Correct:    correctCount,
		Total:      quiz.TotalCount,
	}, middleware.RequestIDStr(c))

	response.OK(c, gin.H{
		"quizId":       quiz.ID,
		"score":        score,
		"scoreMarks":   scoreMarks,
		"correctCount": correctCount,
		"totalCount":   quiz.TotalCount,
		"status":       "SUBMITTED",
	})
}

func (s *Service) Result(c *gin.Context) {
	quizID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid quiz id.")
		return
	}
	userID, _ := middleware.UserID(c)

	var quiz models.Quiz
	if err := s.DB.Preload("Answers").First(&quiz, uint(quizID)).Error; err != nil {
		response.NotFound(c, "Quiz not found.")
		return
	}
	if quiz.StudentID != userID {
		response.Forbidden(c, "This quiz does not belong to you.")
		return
	}

	var topic models.Topic
	var subject models.Subject
	var classLevel models.ClassLevel
	var term models.Term
	s.DB.First(&topic, quiz.TopicID)
	s.DB.First(&subject, topic.SubjectID)
	s.DB.First(&classLevel, topic.ClassLevelID)
	s.DB.First(&term, topic.TermID)

	var questions []models.Question
	s.DB.Preload("Options", func(db *gorm.DB) *gorm.DB {
		return db.Order("question_options.\"order\" asc")
	}).Where("topic_id = ?", quiz.TopicID).Find(&questions)
	qmap := map[uint]models.Question{}
	for _, q := range questions {
		qmap[q.ID] = q
	}

	// Load AI analysis rows for this quiz's answers.
	var analyses []models.AIAnalysis
	s.DB.Where("quiz_id = ? AND kind = ?", quiz.ID, "MISTAKE").Find(&analyses)
	aiByAnswer := map[uint]models.AIAnalysis{}
	for _, a := range analyses {
		if a.AnswerID != nil {
			aiByAnswer[*a.AnswerID] = a
		}
	}

	feedback := make([]map[string]any, 0, len(quiz.Answers))
	for _, ans := range quiz.Answers {
		q, ok := qmap[ans.QuestionID]
		if !ok {
			continue
		}
		item := buildFeedback(q, ans)
		if ai, exists := aiByAnswer[ans.ID]; exists {
			var aiData any = nil
			if ai.Status == "SUCCESS" && ai.ResponseJSON != "" {
				var parsed any
				_ = json.Unmarshal([]byte(ai.ResponseJSON), &parsed)
				aiData = parsed
			}
			item["ai"] = map[string]any{
				"status":   ai.Status,
				"feedback": aiData,
				"kind":     ai.Kind,
			}
		} else {
			item["ai"] = map[string]any{"status": "", "feedback": nil}
		}
		feedback = append(feedback, item)
	}

	response.OK(c, gin.H{
		"quiz":      quiz,
		"topic":     topic,
		"subject":   subject,
		"className": classLevel.Name,
		"term":      term,
		"feedback":  feedback,
	})
}

func buildFeedback(q models.Question, ans models.StudentAnswer) map[string]any {
	item := map[string]any{
		"question":         q,
		"isCorrect":        ans.IsCorrect,
		"receivedMarks":    ans.ReceivedMarks,
		"studentAnswer":    ans.StudentAnswer,
		"working":          ans.Working,
		"selectedOptionId": ans.SelectedOptionID,
		"explanation":      q.Explanation,
		"expectedWorking":  q.ExpectedWorking,
	}
	switch q.Type {
	case "MCQ":
		var correctText string
		for _, o := range q.Options {
			if o.IsCorrect {
				correctText = o.Key + ". " + o.Text
				break
			}
		}
		item["correctAnswer"] = correctText
	case "CALCULATION":
		item["correctAnswer"] = q.CorrectValue + (func() string {
			if q.CorrectUnit != "" {
				return " " + q.CorrectUnit
			}
			return ""
		})()
	}
	return item
}

func (s *Service) recordActivity(studentID uint, count int) {
	if count == 0 {
		return
	}
	today := time.Now().Truncate(24 * time.Hour)
	var act models.DailyActivity
	err := s.DB.Where("student_id = ? AND date = ?", studentID, today).First(&act).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.DB.Create(&models.DailyActivity{StudentID: studentID, Date: today, QuestionCount: count})
		return
	}
	act.QuestionCount += count
	s.DB.Save(&act)
}
