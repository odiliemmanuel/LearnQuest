package knowledge

import (
	"context"
	"time"

	"github.com/learnquest/backend/internal/models"
	"gorm.io/gorm"
)

const (
	StateNew       = "NEW"
	StateWeak      = "WEAK"
	StateImproving = "IMPROVING"
	StateStrong    = "STRONG"
)

func StateFor(mastery float64) string {
	switch {
	case mastery < 0.5:
		return StateWeak
	case mastery < 0.75:
		return StateImproving
	default:
		return StateStrong
	}
}

// UpdateFromQuiz recomputes the student's KnowledgeState for the topic and the
// subject-level progress, then persists both. Returns the updated state.
func UpdateFromQuiz(ctx context.Context, db *gorm.DB, studentID, topicID uint, quizScore float64, quizCorrect, quizTotal int) (*models.KnowledgeState, error) {
	var topic models.Topic
	if err := db.WithContext(ctx).First(&topic, topicID).Error; err != nil {
		return nil, err
	}

	var ks models.KnowledgeState
	db.WithContext(ctx).Where("student_id = ? AND topic_id = ?", studentID, topicID).First(&ks)
	isNew := ks.ID == 0

	ks.StudentID = studentID
	ks.TopicID = topicID
	ks.Attempts += quizTotal
	ks.CorrectCount += quizCorrect

	allTime := 0.0
	if ks.Attempts > 0 {
		allTime = float64(ks.CorrectCount) / float64(ks.Attempts)
	}
	lastScore := quizScore / 100.0
	mastery := lastScore*0.6 + allTime*0.4
	if isNew {
		mastery = lastScore
	}

	now := time.Now()
	ks.Mastery = round2(mastery)
	ks.Confidence = round2(min(0.95, 0.25+float64(ks.Attempts)*0.02))
	ks.State = StateFor(ks.Mastery)
	ks.LastAssessedAt = &now

	if isNew {
		if err := db.WithContext(ctx).Create(&ks).Error; err != nil {
			return nil, err
		}
	} else {
		if err := db.WithContext(ctx).Save(&ks).Error; err != nil {
			return nil, err
		}
	}

	if err := RecomputeSubject(ctx, db, studentID, topic.SubjectID); err != nil {
		return nil, err
	}
	return &ks, nil
}

// RecomputeSubject rebuilds the subject-level progress row for a student.
func RecomputeSubject(ctx context.Context, db *gorm.DB, studentID, subjectID uint) error {
	var topicIDs []uint
	db.WithContext(ctx).Model(&models.Topic{}).Where("subject_id = ?", subjectID).Pluck("id", &topicIDs)

	var states []models.KnowledgeState
	db.WithContext(ctx).Where("student_id = ? AND topic_id IN ?", studentID, topicIDs).Find(&states)

	total := len(topicIDs)
	mastered := 0
	sumMastery := 0.0
	for _, st := range states {
		if st.State == StateStrong {
			mastered++
		}
		sumMastery += st.Mastery
	}
	var correct, attempted int64
	db.WithContext(ctx).Model(&models.StudentAnswer{}).
		Joins("JOIN quizzes ON quizzes.id = student_answers.quiz_id").
		Joins("JOIN topics ON topics.id = quizzes.topic_id").
		Where("student_answers.student_id = ? AND topics.subject_id = ?", studentID, subjectID).
		Count(&attempted)
	db.WithContext(ctx).Model(&models.StudentAnswer{}).
		Joins("JOIN quizzes ON quizzes.id = student_answers.quiz_id").
		Joins("JOIN topics ON topics.id = quizzes.topic_id").
		Where("student_answers.student_id = ? AND topics.subject_id = ? AND student_answers.is_correct = ?", studentID, subjectID, true).
		Count(&correct)

	percentage := 0.0
	if len(states) > 0 {
		percentage = sumMastery / float64(len(states))
	}
	now := time.Now()

	var prog models.StudentSubjectProgress
	err := db.WithContext(ctx).Where("student_id = ? AND subject_id = ?", studentID, subjectID).First(&prog).Error
	prog.StudentID = studentID
	prog.SubjectID = subjectID
	prog.TopicsTotal = total
	prog.TopicsMastered = mastered
	prog.CorrectCount = int(correct)
	prog.TotalCount = int(attempted)
	prog.Percentage = round2(percentage)
	prog.LastAssessedAt = &now
	if err == gorm.ErrRecordNotFound {
		return db.WithContext(ctx).Create(&prog).Error
	}
	return db.WithContext(ctx).Save(&prog).Error
}

func round2(v float64) float64 {
	return float64(int(v*10000)) / 10000
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
