package workers

import (
	"context"
	"log"

	"github.com/learnquest/backend/internal/ai"
	"github.com/learnquest/backend/internal/events"
	"github.com/learnquest/backend/internal/gamification"
	"github.com/learnquest/backend/internal/knowledge"
	"github.com/learnquest/backend/internal/models"
	"github.com/learnquest/backend/internal/quiz"
	"github.com/learnquest/backend/internal/recommendation"
	"gorm.io/gorm"
)

type QuizProcessor struct {
	DB  *gorm.DB
	AI  *ai.Service
	Bus *events.Broker
}

func (p *QuizProcessor) HandleQuizCompleted(ctx context.Context, e events.Event) error {
	payload, ok := e.Data.(quiz.QuizCompletedPayload)
	if !ok {
		req, ok := e.Data.(*quiz.QuizCompletedPayload)
		if !ok {
			return nil
		}
		payload = *req
	}

	studentID := payload.StudentID
	topicID := payload.TopicID

	// 1. Knowledge state must be fresh before recommendations/badges read it.
	state, err := knowledge.UpdateFromQuiz(ctx, p.DB, studentID, topicID, payload.Score, payload.Correct, payload.Total)
	if err != nil {
		log.Printf("workers: knowledge update failed student=%d topic=%d err=%v", studentID, topicID, err)
	}

	var topic models.Topic
	subjectID := uint(0)
	if p.DB.WithContext(ctx).First(&topic, topicID).Error == nil {
		subjectID = topic.SubjectID
	}

	passed := payload.Score >= 60
	perfect := payload.Score >= 99.9

	// 2. Recommendations + close completed recovery missions.
	if err := recommendation.GenerateFromQuiz(ctx, p.DB, studentID, subjectID); err != nil {
		log.Printf("workers: recommendations failed student=%d err=%v", studentID, err)
	}
	if payload.IsRecovery {
		if err := recommendation.CompleteForQuiz(ctx, p.DB, studentID, topicID, passed); err != nil {
			log.Printf("workers: complete recovery failed student=%d err=%v", studentID, err)
		}
	}

	// 3. XP, challenges, badges (never block, never share the quiz transaction).
	if err := gamification.AwardQuizXP(ctx, p.DB, studentID, payload.QuizID, payload.Correct, perfect, payload.IsRecovery, passed); err != nil {
		log.Printf("workers: xp failed student=%d err=%v", studentID, err)
	}
	if err := gamification.UpdateChallenges(ctx, p.DB, studentID); err != nil {
		log.Printf("workers: challenges failed student=%d err=%v", studentID, err)
	}
	if _, err := gamification.CheckBadges(ctx, p.DB, studentID, subjectID, calcCorrect(p.DB, payload.QuizID)); err != nil {
		log.Printf("workers: badges failed student=%d err=%v", studentID, err)
	}

	// 4. AI explanation prefetch (slow, fully outside the request path).
	if p.AI != nil {
		p.AI.FillPending(ctx, payload.QuizID)
	}
	_ = state
	return nil
}

func calcCorrect(db *gorm.DB, quizID uint) bool {
	var n int64
	db.Model(&models.StudentAnswer{}).
		Joins("JOIN questions ON questions.id = student_answers.question_id").
		Where("student_answers.quiz_id = ? AND student_answers.is_correct = ? AND questions.type = ?", quizID, true, "CALCULATION").
		Count(&n)
	return n > 0
}
