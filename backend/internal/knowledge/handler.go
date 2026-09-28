package knowledge

import (
	"github.com/gin-gonic/gin"
	"github.com/learnquest/backend/internal/middleware"
	"github.com/learnquest/backend/internal/models"
	"github.com/learnquest/backend/internal/response"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

type SubjectGroup struct {
	Subject    models.Subject `json:"subject"`
	Topics     []TopicEntry   `json:"topics"`
	Total      int            `json:"total"`
	Mastered   int            `json:"mastered"`
	Improving  int            `json:"improving"`
	Weak       int            `json:"weak"`
	Percentage float64        `json:"percentage"`
}

type TopicEntry struct {
	ID        uint    `json:"id"`
	Name      string  `json:"name"`
	State     string  `json:"state"`
	Mastery   float64 `json:"mastery"`
	Attempts  int     `json:"attempts"`
	Correct   int     `json:"correct"`
	ClassName string  `json:"className"`
	TermName  string  `json:"termName"`
}

func (h *Handler) KnowledgeMap(c *gin.Context) {
	userID, _ := middleware.UserID(c)

	var ks []models.KnowledgeState
	h.DB.Where("student_id = ?", userID).Find(&ks)

	ksByTopic := map[uint]models.KnowledgeState{}
	for _, k := range ks {
		ksByTopic[k.TopicID] = k
	}

	var topicIDs []uint
	for _, k := range ks {
		topicIDs = append(topicIDs, k.TopicID)
	}
	var topics []models.Topic
	if len(topicIDs) > 0 {
		h.DB.Where("id IN ?", topicIDs).Find(&topics)
	}
	topicByID := map[uint]models.Topic{}
	for _, t := range topics {
		topicByID[t.ID] = t
	}

	// Only include subjects the student has any state in, so the map is focused.
	subjects := map[uint]*SubjectGroup{}
	var order []uint
	for tID, k := range ksByTopic {
		topic := topicByID[tID]
		if topic.ID == 0 {
			continue
		}
		g, ok := subjects[topic.SubjectID]
		if !ok {
			var s models.Subject
			h.DB.First(&s, topic.SubjectID)
			g = &SubjectGroup{Subject: s}
			subjects[topic.SubjectID] = g
			order = append(order, topic.SubjectID)
		}
		var className, termName string
		var cl models.ClassLevel
		var tm models.Term
		if h.DB.First(&cl, topic.ClassLevelID).Error == nil {
			className = cl.Name
		}
		if h.DB.First(&tm, topic.TermID).Error == nil {
			termName = tm.Name
		}
		g.Topics = append(g.Topics, TopicEntry{
			ID: topic.ID, Name: topic.Name, State: k.State, Mastery: round2(k.Mastery),
			Attempts: k.Attempts, Correct: k.CorrectCount, ClassName: className, TermName: termName,
		})
	}
	for _, id := range order {
		g := subjects[id]
		for _, t := range g.Topics {
			g.Total++
			switch t.State {
			case StateStrong:
				g.Mastered++
			case StateImproving:
				g.Improving++
			case StateWeak:
				g.Weak++
			}
		}
		if g.Total > 0 {
			sum := 0.0
			for _, t := range g.Topics {
				sum += t.Mastery
			}
			g.Percentage = round2(sum / float64(g.Total))
		}
	}
	groups := make([]SubjectGroup, 0, len(order))
	for _, id := range order {
		groups = append(groups, *subjects[id])
	}
	response.OK(c, gin.H{"subjects": groups})
}

func (h *Handler) Progress(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	var progress []models.StudentSubjectProgress
	h.DB.Preload("Subject").Where("student_id = ?", userID).Order("percentage desc").Find(&progress)
	response.OK(c, progress)
}
