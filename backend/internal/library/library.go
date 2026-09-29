package library

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/learnquest/backend/internal/models"
	"github.com/learnquest/backend/internal/response"
	"gorm.io/gorm"
)

type Handler struct {
	DB        *gorm.DB
	IngestKey string
}

type subjectOut struct {
	Name   string   `json:"name"`
	Topics []string `json:"topics"`
}

type curriculumOut struct {
	Classes  []string     `json:"classes"`
	Terms    []string     `json:"terms"`
	Subjects []subjectOut `json:"subjects"`
}

// CurriculumSync builds the subject/topic list scripts/textbook_fetcher
// expects, flattened from the real curriculum tables.
func (h *Handler) CurriculumSync(c *gin.Context) {
	var rows []struct {
		Subject string
		Topic   string
	}
	if err := h.DB.Table("topics").
		Select("subjects.name AS subject, topics.name AS topic").
		Joins("JOIN subjects ON subjects.id = topics.subject_id").
		Order("subjects.name ASC, topics.sequence ASC").
		Scan(&rows).Error; err != nil {
		response.Internal(c, "Could not load curriculum.")
		return
	}

	grouped := map[string][]string{}
	var order []string
	for _, r := range rows {
		if _, ok := grouped[r.Subject]; !ok {
			order = append(order, r.Subject)
		}
		grouped[r.Subject] = append(grouped[r.Subject], r.Topic)
	}

	out := make([]subjectOut, 0, len(order))
	for _, name := range order {
		out = append(out, subjectOut{Name: name, Topics: grouped[name]})
	}
	response.OK(c, curriculumOut{Classes: []string{"JSS1", "JSS2", "JSS3", "SS1", "SS2", "SS3"}, Terms: []string{"1st Term", "2nd Term", "3rd Term"}, Subjects: out})
}

type noteOut struct {
	ID        uint   `json:"id"`
	Subject   string `json:"subject"`
	Topic     string `json:"topic"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type librarySubjectOut struct {
	Name  string    `json:"name"`
	Notes []noteOut `json:"notes"`
	Count int       `json:"count"`
}

// List returns study notes grouped by subject, for the in-app Library.
func (h *Handler) List(c *gin.Context) {
	var notes []models.LibraryNote
	// Insertion order (id) keeps seeded novelties like chapter notes sequential,
	// since lexical topic ordering would scramble roman numerals.
	if err := h.DB.Order("subject ASC, id ASC").Find(&notes).Error; err != nil {
		response.Internal(c, "Could not load the library.")
		return
	}

	grouped := map[string][]noteOut{}
	var order []string
	for _, n := range notes {
		if _, ok := grouped[n.Subject]; !ok {
			order = append(order, n.Subject)
		}
		grouped[n.Subject] = append(grouped[n.Subject], noteOut{
			ID: n.ID, Subject: n.Subject, Topic: n.Topic, Title: n.Title, Content: n.Content,
			CreatedAt: n.CreatedAt.UTC().Format("2006-01-02"),
		})
	}

	result := make([]librarySubjectOut, 0, len(order))
	for _, name := range order {
		result = append(result, librarySubjectOut{Name: name, Notes: grouped[name], Count: len(grouped[name])})
	}
	response.OK(c, result)
}

type ingestRequest struct {
	Subject string `json:"subject" binding:"required"`
	Topic   string `json:"topic" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// Ingest is called by scripts/textbook_fetcher, never by the frontend. It is
// protected by a shared secret header rather than user JWTs because no
// logged-in user is involved. (subject, topic) is unique, so re-running the
// fetcher refreshes an existing note instead of duplicating it.
func (h *Handler) Ingest(c *gin.Context) {
	if h.IngestKey == "" || c.GetHeader("X-Ingest-Key") != h.IngestKey {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid or missing X-Ingest-Key"})
		return
	}

	var req ingestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "subject, topic, title and content are required"})
		return
	}

	var existing models.LibraryNote
	err := h.DB.Where("subject = ? AND topic = ?", req.Subject, req.Topic).First(&existing).Error
	if err == nil {
		if err := h.DB.Model(&existing).Updates(map[string]any{"title": req.Title, "content": req.Content}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "could not save note"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "updated", "id": existing.ID}})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "could not save note"})
		return
	}

	note := models.LibraryNote{
		Subject: req.Subject, Topic: req.Topic, Title: req.Title, Content: req.Content,
	}
	if err := h.DB.Create(&note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "could not save note"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": gin.H{"message": "added", "id": note.ID}})
}
