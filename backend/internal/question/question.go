package question

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/learnquest/backend/internal/models"
	"github.com/learnquest/backend/internal/response"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

type PublicView struct {
	ID         uint                    `json:"id"`
	TopicID    uint                    `json:"topicId"`
	Type       string                  `json:"type"`
	Prompt     string                  `json:"prompt"`
	Difficulty int                     `json:"difficulty"`
	Marks      int                     `json:"marks"`
	Options    []models.QuestionOption `json:"options"`
}

// ToPublic strips correct answers / working / explanation before returning.
func ToPublic(q models.Question) PublicView {
	opts := make([]models.QuestionOption, 0, len(q.Options))
	for _, o := range q.Options {
		opts = append(opts, models.QuestionOption{ID: o.ID, Key: o.Key, Text: o.Text, Order: o.Order})
	}
	return PublicView{
		ID:         q.ID,
		TopicID:    q.TopicID,
		Type:       q.Type,
		Prompt:     q.Prompt,
		Difficulty: q.Difficulty,
		Marks:      q.Marks,
		Options:    opts,
	}
}

func (h *Handler) ByTopic(c *gin.Context) {
	topicID, err := strconv.ParseUint(c.Param("topicId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid topic id.")
		return
	}
	var questions []models.Question
	if err := h.DB.Preload("Options", func(db *gorm.DB) *gorm.DB {
		return db.Order("question_options.\"order\" asc")
	}).Where("topic_id = ?", uint(topicID)).Order("difficulty asc").Find(&questions).Error; err != nil {
		response.Internal(c, "Could not load questions.")
		return
	}
	views := make([]PublicView, 0, len(questions))
	for _, q := range questions {
		views = append(views, ToPublic(q))
	}
	response.OK(c, views)
}
