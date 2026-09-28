package gamification

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
	rows := make([]ChallengeView, 0, len(challenges))
	for _, ch := range challenges {
		progress := challengeProgress(ctx, h.DB, userID, ch)
		var sc models.StudentChallenge
		h.DB.Where("student_id = ? AND challenge_id = ?", userID, ch.ID).First(&sc)
		completed := sc.Completed
		rows = append(rows, ChallengeView{Challenge: ch, Progress: progress, Completed: completed})
	}
	response.OK(c, rows)
}
