package gamification

import (
	"context"
	"time"

	"github.com/learnquest/backend/internal/models"
	"gorm.io/gorm"
)

const (
	XPPerCorrect      = 10
	XPQuizCompletion  = 5
	XPPerfectBonus    = 15
	XPRecoveryBonus   = 20
	XPChallengeReward = 30
)

type hint = models.XPTransaction

// AwardQuizXP records XP earned from a completed quiz. Called by workers only.
func AwardQuizXP(ctx context.Context, db *gorm.DB, studentID, quizID uint, correct int, perfect bool, recovery bool, passed bool) error {
	now := time.Now()
	tx := db.WithContext(ctx).Begin()
	for i := 0; i < correct; i++ {
		rec := models.XPTransaction{StudentID: studentID, Amount: XPPerCorrect, Reason: "Correct answer", Source: "QUIZ", RefID: quizID, CreatedAt: now}
		if err := tx.Create(&rec).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Create(&models.XPTransaction{StudentID: studentID, Amount: XPQuizCompletion, Reason: "Quiz completed", Source: "QUIZ", RefID: quizID, CreatedAt: now}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if perfect {
		tx.Create(&models.XPTransaction{StudentID: studentID, Amount: XPPerfectBonus, Reason: "Perfect quiz", Source: "QUIZ", RefID: quizID, CreatedAt: now})
	}
	if recovery && passed {
		tx.Create(&models.XPTransaction{StudentID: studentID, Amount: XPRecoveryBonus, Reason: "Recovery mission passed", Source: "QUIZ", RefID: quizID, CreatedAt: now})
	}
	return tx.Commit().Error
}

func TotalXP(ctx context.Context, db *gorm.DB, studentID uint) (int, error) {
	var sum struct{ Total int }
	err := db.WithContext(ctx).Model(&models.XPTransaction{}).
		Select("COALESCE(SUM(amount),0) AS total").
		Where("student_id = ?", studentID).Scan(&sum).Error
	return sum.Total, err
}

func LevelFor(xp int) int { return xp/100 + 1 }

// Streak computes current and longest streak (in days) from daily activity.
func Streak(ctx context.Context, db *gorm.DB, studentID uint) (current, longest int, err error) {
	var days []time.Time
	db.WithContext(ctx).Model(&models.DailyActivity{}).
		Where("student_id = ?", studentID).Order("date desc").Pluck("date", &days)
	set := map[string]bool{}
	for _, d := range days {
		set[d.Truncate(24*time.Hour).Format("2006-01-02")] = true
	}

	today := time.Now().Truncate(24 * time.Hour)
	cur := 0
	if set[today.Format("2006-01-02")] {
		cur = 1
	}
	d := today
	for {
		d = d.AddDate(0, 0, -1)
		if set[d.Format("2006-01-02")] {
			cur++
			continue
		}
		break
	}

	// longest run
	longest = 0
	run := 0
	d2 := today.AddDate(0, 0, -100)
	for i := 0; i <= 400; i++ {
		if set[d2.Format("2006-01-02")] {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
		d2 = d2.AddDate(0, 0, 1)
	}
	return cur, longest, nil
}

// CheckBadges awards any newly-earned badges for the given student.
func CheckBadges(ctx context.Context, db *gorm.DB, studentID uint, subjectID uint, calcCorrect bool) ([]models.StudentBadge, error) {
	earned := []models.StudentBadge{}
	award := func(code string) error {
		var badge models.Badge
		if err := db.WithContext(ctx).Where("code = ?", code).First(&badge).Error; err != nil {
			return nil // badge not seeded
		}
		var n int64
		db.WithContext(ctx).Model(&models.StudentBadge{}).Where("student_id = ? AND badge_id = ?", studentID, badge.ID).Count(&n)
		if n > 0 {
			return nil
		}
		sb := models.StudentBadge{StudentID: studentID, BadgeID: badge.ID, EarnedAt: time.Now()}
		if err := db.WithContext(ctx).Create(&sb).Error; err != nil {
			return err
		}
		earned = append(earned, sb)
		return nil
	}

	var quizCount int64
	db.WithContext(ctx).Model(&models.Quiz{}).Where("student_id = ? AND status = ?", studentID, "SUBMITTED").Count(&quizCount)
	if quizCount >= 1 {
		award("first_quiz")
	}

	var correctCount int64
	db.WithContext(ctx).Model(&models.StudentAnswer{}).Where("student_id = ? AND is_correct = ?", studentID, true).Count(&correctCount)
	if correctCount >= 10 {
		award("rising_star")
	}

	if calcCorrect {
		var calcCorrectCount int64
		db.WithContext(ctx).Model(&models.StudentAnswer{}).
			Joins("JOIN questions ON questions.id = student_answers.question_id").
			Where("student_answers.student_id = ? AND student_answers.is_correct = ? AND questions.type = ?", studentID, true, "CALCULATION").
			Count(&calcCorrectCount)
		if calcCorrectCount >= 5 {
			award("calc_wizard")
		}
	}

	cur, _, _ := Streak(ctx, db, studentID)
	if cur >= 3 {
		award("streak_3")
	}
	if cur >= 7 {
		award("streak_7")
	}

	var recoveryCount int64
	db.WithContext(ctx).Model(&models.Quiz{}).Where("student_id = ? AND status = ? AND is_recovery = ?", studentID, "SUBMITTED", true).Count(&recoveryCount)
	if recoveryCount >= 3 {
		award("recovery_hero")
	}

	if subjectID != 0 {
		var mastered int64
		db.WithContext(ctx).Model(&models.KnowledgeState{}).
			Joins("JOIN topics ON topics.id = knowledge_states.topic_id").
			Where("knowledge_states.student_id = ? AND topics.subject_id = ? AND knowledge_states.state = ?", studentID, subjectID, "STRONG").
			Count(&mastered)
		if mastered >= 3 {
			award("subject_explorer")
		}
		var physicsScore float64
		db.WithContext(ctx).Model(&models.Quiz{}).
			Joins("JOIN topics ON topics.id = quizzes.topic_id").
			Joins("JOIN subjects ON subjects.id = topics.subject_id").
			Where("quizzes.student_id = ? AND subjects.code = ? AND quizzes.status = ? AND quizzes.score >= ?", studentID, "PHY", "SUBMITTED", 60).
			Select("MAX(quizzes.score)").Scan(&physicsScore)
		if physicsScore >= 60 {
			award("physics_beginner")
		}
	}
	return earned, nil
}
