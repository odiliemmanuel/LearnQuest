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
//
// All of the per-topic reads and writes are done set-based (a handful of
// queries total) rather than 3+ queries per topic, since this runs after every
// quiz submission.
func GenerateFromQuiz(ctx context.Context, db *gorm.DB, studentID, subjectID uint) error {
	var topics []models.Topic
	if err := db.WithContext(ctx).Where("subject_id = ?", subjectID).Find(&topics).Error; err != nil {
		return err
	}
	topicIDs := make([]uint, 0, len(topics))
	for _, t := range topics {
		topicIDs = append(topicIDs, t.ID)
	}

	now := time.Now()

	// One query for every knowledge state in this subject instead of one per topic.
	stateByTopic := map[uint]models.KnowledgeState{}
	if len(topicIDs) > 0 {
		var states []models.KnowledgeState
		if err := db.WithContext(ctx).
			Where("student_id = ? AND topic_id IN ?", studentID, topicIDs).
			Find(&states).Error; err == nil {
			for _, ks := range states {
				stateByTopic[ks.TopicID] = ks
			}
		}
	}

	// One query for all currently-open recommendations on those topics.
	openByTopic := map[uint]models.Recommendation{}
	if len(topicIDs) > 0 {
		var open []models.Recommendation
		if err := db.WithContext(ctx).
			Where("student_id = ? AND topic_id IN ? AND status = ?", studentID, topicIDs, StatusOpen).
			Find(&open).Error; err == nil {
			for _, rec := range open {
				openByTopic[rec.TopicID] = rec
			}
		}
	}

	var toCreate []models.Recommendation
	var toUpdate []models.Recommendation
	var strongTopicIDs []uint

	for _, t := range topics {
		ks, ok := stateByTopic[t.ID]
		if !ok || ks.ID == 0 {
			continue
		}
		switch {
		case ks.State == knowledge.StateWeak || ks.State == knowledge.StateImproving:
			// Skip topics never touched; leave existing recs alone if still OPEN.
			rec, exists := openByTopic[t.ID]
			if !exists {
				priority := 0
				if ks.State == knowledge.StateWeak {
					priority = 1
				}
				toCreate = append(toCreate, models.Recommendation{
					StudentID: studentID,
					TopicID:   t.ID,
					Kind:      KindWeak,
					Reason:    recommendationReason(ks.State),
					Priority:  priority,
					Status:    StatusOpen,
					CreatedAt: now,
				})
			} else if rec.Kind == KindWeak && ks.State == knowledge.StateImproving {
				rec.Priority = 2
				toUpdate = append(toUpdate, rec)
			}
		case ks.State == knowledge.StateStrong:
			strongTopicIDs = append(strongTopicIDs, t.ID)
		case ks.State == knowledge.StateNew:
			// no-op
		}
	}

	// Close out every strong topic's open recs in a single UPDATE.
	if len(strongTopicIDs) > 0 {
		db.WithContext(ctx).Model(&models.Recommendation{}).
			Where("student_id = ? AND topic_id IN ? AND status = ?", studentID, strongTopicIDs, StatusOpen).
			Updates(map[string]any{"status": StatusDone, "completed_at": now})
	}
	if len(toUpdate) > 0 {
		db.WithContext(ctx).Save(&toUpdate)
	}
	if len(toCreate) > 0 {
		db.WithContext(ctx).CreateInBatches(toCreate, 50)
	}

	// Fresh weak topics (<50%) get a RECOVERY mission if they have content.
	var weakTopics []uint
	db.WithContext(ctx).Model(&models.KnowledgeState{}).
		Where("student_id = ? AND state = ?", studentID, knowledge.StateWeak).
		Pluck("topic_id", &weakTopics)
	if len(weakTopics) > 0 {
		// Which weak topics actually have questions — one grouped query.
		hasQuestions := map[uint]bool{}
		var rows []struct {
			TopicID uint
		}
		db.WithContext(ctx).Model(&models.Question{}).
			Select("DISTINCT topic_id").
			Where("topic_id IN ?", weakTopics).
			Scan(&rows)
		for _, r := range rows {
			hasQuestions[r.TopicID] = true
		}

		// Existing open recs for weak topics — one query.
		weakOpen := map[uint]bool{}
		var openRecs []models.Recommendation
		if err := db.WithContext(ctx).
			Where("student_id = ? AND topic_id IN ? AND status = ?", studentID, weakTopics, StatusOpen).
			Find(&openRecs).Error; err == nil {
			for _, rec := range openRecs {
				weakOpen[rec.TopicID] = true
			}
		}

		var recoveries []models.Recommendation
		for _, tID := range weakTopics {
			if !hasQuestions[tID] || weakOpen[tID] {
				continue
			}
			recoveries = append(recoveries, models.Recommendation{
				StudentID: studentID,
				TopicID:   tID,
				Kind:      KindRecovery,
				Reason:    "Recovery mission: re-learn and retake this topic.",
				Priority:  1,
				Status:    StatusOpen,
				CreatedAt: now,
			})
		}
		if len(recoveries) > 0 {
			db.WithContext(ctx).CreateInBatches(recoveries, 50)
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
