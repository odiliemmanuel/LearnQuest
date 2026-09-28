package curriculum

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/learnquest/backend/internal/middleware"
	"github.com/learnquest/backend/internal/models"
	"github.com/learnquest/backend/internal/response"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

type TopicView struct {
	models.Topic
	HasLesson    bool    `json:"hasLesson"`
	HasQuestions bool    `json:"hasQuestions"`
	Mastery      float64 `json:"mastery,omitempty"`
	State        string  `json:"state,omitempty"`
}

func (h *Handler) Levels(c *gin.Context) {
	var levels []models.EducationLevel
	h.DB.Order("sequence asc").Find(&levels)
	response.OK(c, levels)
}

func (h *Handler) Classes(c *gin.Context) {
	levelID := c.Query("levelId")
	q := h.DB.Order("sequence asc")
	if levelID != "" {
		if id, err := strconv.ParseUint(levelID, 10, 64); err == nil {
			q = q.Where("education_level_id = ?", uint(id))
		} else {
			response.BadRequest(c, "Invalid levelId.")
			return
		}
	}
	var classes []models.ClassLevel
	q.Find(&classes)
	response.OK(c, classes)
}

func (h *Handler) Terms(c *gin.Context) {
	var terms []models.Term
	h.DB.Order("sequence asc").Find(&terms)
	response.OK(c, terms)
}

func (h *Handler) SubjectsAll(c *gin.Context) {
	var subjects []models.Subject
	h.DB.Order("name asc").Find(&subjects)
	response.OK(c, subjects)
}

func (h *Handler) SubjectsForClass(c *gin.Context) {
	classIDs := c.QueryArray("classId")
	if len(classIDs) == 0 || classIDs[0] == "" {
		response.BadRequest(c, "classId is required.")
		return
	}
	var ids []uint
	for _, raw := range classIDs {
		if id, err := strconv.ParseUint(raw, 10, 64); err == nil {
			ids = append(ids, uint(id))
		}
	}
	if len(ids) == 0 {
		response.BadRequest(c, "Invalid classId.")
		return
	}
	var subjects []models.Subject
	err := h.DB.
		Joins("JOIN class_subjects cs ON cs.subject_id = subjects.id AND cs.class_level_id IN ?", ids).
		Distinct().
		Order("subjects.name asc").
		Find(&subjects).Error
	if err != nil {
		response.Internal(c, "Could not load subjects.")
		return
	}
	response.OK(c, subjects)
}

func (h *Handler) Topics(c *gin.Context) {
	classID := c.Query("classId")
	termID := c.Query("termId")
	subjectID := c.Query("subjectId")

	q := h.DB.Model(&models.Topic{})
	if classID != "" {
		if id, err := strconv.ParseUint(classID, 10, 64); err == nil {
			q = q.Where("class_level_id = ?", uint(id))
		}
	}
	if termID != "" {
		if id, err := strconv.ParseUint(termID, 10, 64); err == nil {
			q = q.Where("term_id = ?", uint(id))
		}
	}
	if subjectID != "" {
		if id, err := strconv.ParseUint(subjectID, 10, 64); err == nil {
			q = q.Where("subject_id = ?", uint(id))
		}
	}

	var topics []models.Topic
	if err := q.Order("sequence asc").Find(&topics).Error; err != nil {
		response.Internal(c, "Could not load topics.")
		return
	}

	views := make([]TopicView, 0, len(topics))
	for i := range topics {
		view := TopicView{Topic: topics[i]}
		var lessonCount, questionCount int64
		h.DB.Model(&models.Lesson{}).Where("topic_id = ?", topics[i].ID).Count(&lessonCount)
		h.DB.Model(&models.Question{}).Where("topic_id = ?", topics[i].ID).Count(&questionCount)
		view.HasLesson = lessonCount > 0
		view.HasQuestions = questionCount > 0
		if userID, ok := middleware.UserID(c); ok {
			var ks models.KnowledgeState
			h.DB.Where("student_id = ? AND topic_id = ?", userID, topics[i].ID).First(&ks)
			if ks.ID != 0 {
				view.Mastery = ks.Mastery
				view.State = ks.State
			}
		}
		views = append(views, view)
	}
	response.OK(c, views)
}

func (h *Handler) TopicDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid topic id.")
		return
	}
	var topic models.Topic
	if err := h.DB.First(&topic, uint(id)).Error; err != nil {
		response.NotFound(c, "Topic not found.")
		return
	}
	type Detail struct {
		models.Topic
		Subject       models.Subject    `json:"subject"`
		ClassLevel    models.ClassLevel `json:"classLevel"`
		Term          models.Term       `json:"term"`
		HasLesson     bool              `json:"hasLesson"`
		LessonID      *uint             `json:"lessonId,omitempty"`
		QuestionCount int64             `json:"questionCount"`
		Mastery       float64           `json:"mastery,omitempty"`
		State         string            `json:"state,omitempty"`
	}
	var subject models.Subject
	var classLevel models.ClassLevel
	var term models.Term
	var lesson models.Lesson
	var questionCount int64
	h.DB.First(&subject, topic.SubjectID)
	h.DB.First(&classLevel, topic.ClassLevelID)
	h.DB.First(&term, topic.TermID)
	h.DB.Model(&models.Question{}).Where("topic_id = ?", topic.ID).Count(&questionCount)

	var lessonID *uint
	if err := h.DB.Where("topic_id = ?", topic.ID).First(&lesson).Error; err == nil {
		lessonID = &lesson.ID
	}

	detail := Detail{
		Topic:         topic,
		Subject:       subject,
		ClassLevel:    classLevel,
		Term:          term,
		HasLesson:     lessonID != nil,
		LessonID:      lessonID,
		QuestionCount: questionCount,
	}
	if userID, ok := middleware.UserID(c); ok {
		var ks models.KnowledgeState
		h.DB.Where("student_id = ? AND topic_id = ?", userID, topic.ID).First(&ks)
		if ks.ID != 0 || userID != 0 {
			if ks.ID != 0 {
				detail.Mastery = ks.Mastery
				detail.State = ks.State
			}
		}
	}
	response.OK(c, detail)
}
