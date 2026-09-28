package lesson

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

type LessonView struct {
	models.Lesson
	TopicName   string `json:"topicName"`
	SubjectName string `json:"subjectName"`
	ClassName   string `json:"className"`
	TermName    string `json:"termName"`
	TopicIDV    uint   `json:"-"`
}

func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid lesson id.")
		return
	}
	var lesson models.Lesson
	if err := h.DB.First(&lesson, uint(id)).Error; err != nil {
		response.NotFound(c, "Lesson not found.")
		return
	}
	h.respond(c, lesson)
}

func (h *Handler) GetByTopic(c *gin.Context) {
	topicID, err := strconv.ParseUint(c.Param("topicId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid topic id.")
		return
	}
	var lesson models.Lesson
	if err := h.DB.Where("topic_id = ?", uint(topicID)).First(&lesson).Error; err != nil {
		response.NotFound(c, "No lesson for this topic yet.")
		return
	}
	h.respond(c, lesson)
}

func (h *Handler) respond(c *gin.Context, lesson models.Lesson) {
	var topic models.Topic
	var subject models.Subject
	var classLevel models.ClassLevel
	var term models.Term
	h.DB.First(&topic, lesson.TopicID)
	if topic.ID != 0 {
		h.DB.First(&subject, topic.SubjectID)
		h.DB.First(&classLevel, topic.ClassLevelID)
		h.DB.First(&term, topic.TermID)
	}
	view := LessonView{
		Lesson:      lesson,
		TopicName:   topic.Name,
		SubjectName: subject.Name,
		ClassName:   classLevel.Name,
		TermName:    term.Name,
	}
	response.OK(c, view)
}
