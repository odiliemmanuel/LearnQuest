package gamification

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/learnquest/backend/internal/middleware"
	"github.com/learnquest/backend/internal/models"
	"github.com/learnquest/backend/internal/response"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

type ChallengeView struct {
	models.Challenge
	Progress  int  `json:"progress"`
	Completed bool `json:"completed"`
}

func (h *Handler) Summary(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	ctx := c.Request.Context()

	xp, _ := TotalXP(ctx, h.DB, userID)
	level := LevelFor(xp)
	cur, longest, _ := Streak(ctx, h.DB, userID)
	var badges []models.StudentBadge
	h.DB.Preload("Badge").Where("student_id = ?", userID).Order("earned_at asc").Find(&badges)

	response.OK(c, gin.H{
		"xp":            xp,
		"level":         level,
		"streak":        cur,
		"longestStreak": longest,
		"badges":        badges,
	})
}

func (h *Handler) Badges(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	var all []models.Badge
	h.DB.Order("name asc").Find(&all)
	earned := map[uint]bool{}
	var earnedRows []models.StudentBadge
	h.DB.Where("student_id = ?", userID).Find(&earnedRows)
	for _, eb := range earnedRows {
		earned[eb.BadgeID] = true
	}
	type row struct {
		models.Badge
		Earned bool `json:"earned"`
	}
	rows := make([]row, 0, len(all))
	for _, b := range all {
		_, ok := earned[b.ID]
		rows = append(rows, row{Badge: b, Earned: ok})
	}
	response.OK(c, rows)
}

func (h *Handler) Challenges(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	ctx := c.Request.Context()

	var challenges []models.Challenge
	h.DB.Where("active = ?", true).Order("id asc").Find(&challenges)

	// One aggregate per distinct metric and one query for the student's
	// challenge rows, instead of two-plus queries per challenge.
	progressForMetric := progressByMetric(ctx, h.DB, userID, challenges)
	existing, err := loadStudentChallenges(ctx, h.DB, userID)
	if err != nil {
		// Surface it rather than silently reporting every challenge as
		// unstarted — that is what hid a missing student_challenges table.
		log.Printf("gamification: load student challenges: %v", err)
		existing = nil
	}

	rows := make([]ChallengeView, 0, len(challenges))
	for _, ch := range challenges {
		rows = append(rows, ChallengeView{
			Challenge: ch,
			Progress:  progressForMetric[ch.Metric],
			Completed: existing[ch.ID].Completed,
		})
	}
	response.OK(c, rows)
}
