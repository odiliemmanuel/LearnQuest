package recommendation

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/learnquest/backend/internal/middleware"
	"github.com/learnquest/backend/internal/models"
	"github.com/learnquest/backend/internal/response"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

const defaultSteps = `Review the lesson, Read worked examples, Practice 5 questions, Retake the assessment`

func (h *Handler) List(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	var recs []models.Recommendation
	h.DB.Preload("Topic").Where("student_id = ? AND status = ?", userID, StatusOpen).Order("priority asc, created_at desc").Find(&recs)
	views := make([]gin.H, 0, len(recs))
	for _, r := range recs {
		views = append(views, h.view(c, r))
	}
	response.OK(c, views)
}

func (h *Handler) Recovery(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	var recs []models.Recommendation
	h.DB.Preload("Topic").
		Where("student_id = ? AND status = ? AND kind = ?", userID, StatusOpen, KindRecovery).
		Order("priority asc, created_at desc").
		Find(&recs)
	views := make([]gin.H, 0, len(recs))
	for _, r := range recs {
		views = append(views, h.view(c, r))
	}
	response.OK(c, gin.H{"missions": views, "steps": defaultSteps})
}

func (h *Handler) Complete(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	var in struct {
		TopicID uint `json:"topicId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "topicId is required.")
		return
	}
	now := time.Now()
	err := h.DB.Model(&models.Recommendation{}).
		Where("student_id = ? AND topic_id = ? AND status = ?", userID, in.TopicID, StatusOpen).
		Updates(map[string]any{"status": StatusDone, "completed_at": now}).Error
	if err != nil {
		response.Internal(c, "Could not update recommendation.")
		return
	}
	response.OK(c, gin.H{"completed": in.TopicID})
}

func (h *Handler) view(c *gin.Context, r models.Recommendation) gin.H {
	var subject models.Subject
	var classLevel models.ClassLevel
	var term models.Term
	var ks models.KnowledgeState
	h.DB.First(&subject, r.Topic.SubjectID)
	h.DB.First(&classLevel, r.Topic.ClassLevelID)
	h.DB.First(&term, r.Topic.TermID)
	h.DB.Where("student_id = ? AND topic_id = ?", r.StudentID, r.TopicID).First(&ks)
	var lessonID *uint
	var lesson models.Lesson
	if h.DB.Where("topic_id = ?", r.TopicID).First(&lesson).Error == nil {
		lessonID = &lesson.ID
	}
	return gin.H{
		"id":        r.ID,
		"kind":      r.Kind,
		"priority":  r.Priority,
		"reason":    r.Reason,
		"status":    r.Status,
		"createdAt": r.CreatedAt,
		"topic": gin.H{
			"id":          r.Topic.ID,
			"name":        r.Topic.Name,
			"description": r.Topic.Description,
			"subject":     subject,
			"className":   classLevel.Name,
			"term":        term,
			"mastery":     ks.Mastery,
			"state":       ks.State,
			"lessonId":    lessonID,
		},
		"steps": defaultSteps,
	}
}
