package recommendation

import (
	"context"
	"time"

	"github.com/learnquest/backend/internal/knowledge"
	"github.com/learnquest/backend/internal/models"
	"gorm.io/gorm"
)

const (
	KindWeak     = "WEAK"
	KindRecovery = "RECOVERY"
	StatusOpen   = "OPEN"
	StatusDone   = "COMPLETED"
)

// GenerateFromQuiz inspects all topics of the subject just assessed and keeps
// recommendations aligned with the current knowledge map. Weak/improving topics
// get OPEN recommendations; stronger topics get closed out.
func GenerateFromQuiz(ctx context.Context, db *gorm.DB, studentID, subjectID uint) error {
	var topics []models.Topic
	if err := db.WithContext(ctx).Where("subject_id = ?", subjectID).Find(&topics).Error; err != nil {
		return err
	}
	now := time.Now()
	for _, t := range topics {
		var ks models.KnowledgeState
		db.WithContext(ctx).Where("student_id = ? AND topic_id = ?", studentID, t.ID).First(&ks)
		if ks.ID == 0 {
			continue
		}
		if ks.State == knowledge.StateWeak || ks.State == knowledge.StateImproving {
			// Skip topics never touched; leave existing recs alone if still OPEN.
			var rec models.Recommendation
			err := db.WithContext(ctx).
				Where("student_id = ? AND topic_id = ? AND status = ?", studentID, t.ID, StatusOpen).
				First(&rec).Error
			if err == gorm.ErrRecordNotFound {
				recommendation := models.Recommendation{
					StudentID: studentID,
					TopicID:   t.ID,
					Kind:      KindWeak,
					Reason:    recommendationReason(ks.State),
					Priority:  0,
					Status:    StatusOpen,
					CreatedAt: now,
				}
				if ks.State == knowledge.StateWeak {
					recommendation.Priority = 1
				}
				db.WithContext(ctx).Create(&recommendation)
			} else if err == nil && rec.Kind == KindWeak && ks.State == knowledge.StateImproving {
				rec.Priority = 2
				db.WithContext(ctx).Save(&rec)
			}
		} else if ks.State == knowledge.StateStrong {
			db.WithContext(ctx).Model(&models.Recommendation{}).
				Where("student_id = ? AND topic_id = ? AND status = ?", studentID, t.ID, StatusOpen).
				Updates(map[string]any{"status": StatusDone, "completed_at": now})
		} else if ks.State == knowledge.StateNew {
			// no-op
		}
	}

	// Fresh weak topics (<50%) get a RECOVERY mission if they have content.
	var weakTopics []uint
	db.WithContext(ctx).Model(&models.KnowledgeState{}).
		Where("student_id = ? AND state = ?", studentID, knowledge.StateWeak).
		Pluck("topic_id", &weakTopics)
	if len(weakTopics) > 0 {
		var qCount, lCount int64
		for _, tID := range weakTopics {
			qCount = 0
			lCount = 0
			db.WithContext(ctx).Model(&models.Question{}).Where("topic_id = ?", tID).Count(&qCount)
			db.WithContext(ctx).Model(&models.Lesson{}).Where("topic_id = ?", tID).Count(&lCount)
			if qCount == 0 {
				continue
			}
			var rec models.Recommendation
			err := db.WithContext(ctx).
				Where("student_id = ? AND topic_id = ? AND status = ?", studentID, tID, StatusOpen).
				First(&rec).Error
			if err == gorm.ErrRecordNotFound {
				db.WithContext(ctx).Create(&models.Recommendation{
					StudentID: studentID,
					TopicID:   tID,
					Kind:      KindRecovery,
					Reason:    "Recovery mission: re-learn and retake this topic.",
					Priority:  1,
					Status:    StatusOpen,
					CreatedAt: now,
				})
			}
		}
	}
	return nil
}

func recommendationReason(state string) string {
	if state == knowledge.StateWeak {
		return "This topic needs practice. Review the lesson, then retake the assessment."
	}
	return "Keep practicing this topic to move it to Strong."
}

// CompleteForQuiz closes open recommendations for the topic the student just re-took.
func CompleteForQuiz(ctx context.Context, db *gorm.DB, studentID, topicID uint, passed bool) error {
	now := time.Now()
	if passed {
		return db.WithContext(ctx).Model(&models.Recommendation{}).
			Where("student_id = ? AND topic_id = ? AND status = ?", studentID, topicID, StatusOpen).
			Updates(map[string]any{"status": StatusDone, "completed_at": now}).Error
	}
	return nil
}
