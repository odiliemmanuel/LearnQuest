package gamification

import (
	"context"
	"time"

	"github.com/learnquest/backend/internal/models"
	"gorm.io/gorm"
)

// progressByMetric computes each distinct challenge metric exactly once.
// Several challenges typically share a metric (and some metrics, like
// streak_days, cost multiple queries), so memoising by metric avoids
// re-running the same aggregate query for every challenge row.
func progressByMetric(ctx context.Context, db *gorm.DB, studentID uint, challenges []models.Challenge) map[string]int {
	out := make(map[string]int, len(challenges))
	for _, ch := range challenges {
		if _, done := out[ch.Metric]; done {
			continue
		}
		out[ch.Metric] = challengeProgress(ctx, db, studentID, ch)
	}
	return out
}

// loadStudentChallenges fetches every StudentChallenge for a student in one
// query, keyed by challenge id.
func loadStudentChallenges(ctx context.Context, db *gorm.DB, studentID uint) (map[uint]models.StudentChallenge, error) {
	var rows []models.StudentChallenge
	if err := db.WithContext(ctx).Where("student_id = ?", studentID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]models.StudentChallenge, len(rows))
	for _, r := range rows {
		out[r.ChallengeID] = r
	}
	return out, nil
}

// UpdateChallenges reconciles challenge progress after a quiz and awards XP once.
func UpdateChallenges(ctx context.Context, db *gorm.DB, studentID uint) error {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)

	var challenges []models.Challenge
	if err := db.WithContext(ctx).Where("active = ?", true).Find(&challenges).Error; err != nil {
		return err
	}
	if len(challenges) == 0 {
		return nil
	}

	progressForMetric := progressByMetric(ctx, db, studentID, challenges)
	existing, err := loadStudentChallenges(ctx, db, studentID)
	if err != nil {
		return err
	}

	for _, ch := range challenges {
		progress := progressForMetric[ch.Metric]
		// A missing map entry yields the zero value, which is exactly the
		// "no row yet" case the original ErrRecordNotFound check handled.
		sc, found := existing[ch.ID]
		isNew := !found
		if isNew {
			sc = models.StudentChallenge{StudentID: studentID, ChallengeID: ch.ID}
		}
		sc.Progress = progress

		// Daily challenges must be re-earned each day.
		if ch.Metric == "questions_daily" || ch.Metric == "perfect_quiz_daily" {
			if isNew {
				sc.Completed = false
				sc.CompletedAt = nil
			} else if sc.Completed && (sc.CompletedAt == nil || !sameDay(*sc.CompletedAt, today)) {
				sc.Completed = false
				sc.CompletedAt = nil
			}
		}

		if !sc.Completed && progress >= ch.Target {
			sc.Completed = true
			sc.CompletedAt = &now
			if err := saveChallenge(ctx, db, &sc); err != nil {
				return err
			}
			awardChallengeXP(ctx, db, studentID, ch)
			continue
		}
		if err := saveChallenge(ctx, db, &sc); err != nil {
			return err
		}
	}
	return nil
}

func saveChallenge(ctx context.Context, db *gorm.DB, sc *models.StudentChallenge) error {
	if sc.ID == 0 {
		return db.WithContext(ctx).Create(sc).Error
	}
	return db.WithContext(ctx).Save(sc).Error
}

func challengeProgress(ctx context.Context, db *gorm.DB, studentID uint, ch models.Challenge) int {
	today := time.Now().Truncate(24 * time.Hour)
	switch ch.Metric {
	case "questions_daily":
		var n int64
		db.WithContext(ctx).Model(&models.DailyActivity{}).
			Where("student_id = ? AND date = ?", studentID, today).
			Select("COALESCE(SUM(question_count),0)").Scan(&n)
		return int(n)
	case "perfect_quiz_daily":
		var n int64
		db.WithContext(ctx).Model(&models.Quiz{}).
			Where("student_id = ? AND status = ? AND score >= 99.9 AND submitted_at >= ?", studentID, "SUBMITTED", today).
			Count(&n)
		return int(n)
	case "streak_days":
		cur, _, _ := Streak(ctx, db, studentID)
		return cur
	case "recovery_missions":
		var n int64
		db.WithContext(ctx).Model(&models.Quiz{}).Where("student_id = ? AND is_recovery = ? AND status = ?", studentID, true, "SUBMITTED").Count(&n)
		return int(n)
	}
	return 0
}

func awardChallengeXP(ctx context.Context, db *gorm.DB, studentID uint, ch models.Challenge) {
	db.WithContext(ctx).Create(&models.XPTransaction{
		StudentID: studentID,
		Amount:    ch.XPReward,
		Reason:    "Challenge completed: " + ch.Title,
		Source:    "CHALLENGE",
		RefID:     ch.ID,
		CreatedAt: time.Now(),
	})
}

func sameDay(a, b time.Time) bool {
	return a.Truncate(24 * time.Hour).Equal(b)
}
