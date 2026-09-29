package admin

import (
	"errors"
	"strconv"
	"strings"
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

// Ensure promotes the configured owner email to ADMIN. It is idempotent and
// also covers accounts created before ADMIN_EMAIL was introduced.
func Ensure(db *gorm.DB, adminEmail string) error {
	if adminEmail == "" {
		return nil
	}
	return db.Model(&models.User{}).
		Where("lower(email) = lower(?)", adminEmail).
		Update("role", models.RoleAdmin).Error
}

type RecentUser struct {
	ID            uint        `json:"id"`
	Name          string      `json:"name"`
	Email         string      `json:"email"`
	Role          models.Role `json:"role"`
	EmailVerified bool        `json:"emailVerified"`
	CreatedAt     time.Time   `json:"createdAt"`
}

type Overview struct {
	TotalUsers          int64        `json:"totalUsers"`
	VerifiedUsers       int64        `json:"verifiedUsers"`
	OnboardedUsers      int64        `json:"onboardedUsers"`
	TotalQuizzes        int64        `json:"totalQuizzes"`
	SubmittedQuizzes    int64        `json:"submittedQuizzes"`
	TotalAnswers        int64        `json:"totalAnswers"`
	AIAnalyses          int64        `json:"aiAnalyses"`
	ActiveStudentsToday int64        `json:"activeStudentsToday"`
	AvgScore            float64      `json:"avgScore"`
	RecentRegistrations []RecentUser `json:"recentRegistrations"`
}

func (h *Handler) Overview(c *gin.Context) {
	var ov Overview
	if err := h.DB.Model(&models.User{}).Count(&ov.TotalUsers).Error; err != nil {
		response.Internal(c, "Could not load dashboard stats.")
		return
	}
	if err := h.DB.Model(&models.User{}).Where("email_verified = ?", true).Count(&ov.VerifiedUsers).Error; err != nil {
		response.Internal(c, "Could not load dashboard stats.")
		return
	}
	if err := h.DB.Model(&models.StudentProfile{}).
		Where("education_level_id IS NOT NULL AND class_level_id IS NOT NULL").
		Count(&ov.OnboardedUsers).Error; err != nil {
		response.Internal(c, "Could not load dashboard stats.")
		return
	}
	if err := h.DB.Model(&models.Quiz{}).Count(&ov.TotalQuizzes).Error; err != nil {
		response.Internal(c, "Could not load dashboard stats.")
		return
	}
	if err := h.DB.Model(&models.Quiz{}).Where("status = ?", "SUBMITTED").Count(&ov.SubmittedQuizzes).Error; err != nil {
		response.Internal(c, "Could not load dashboard stats.")
		return
	}
	if err := h.DB.Model(&models.StudentAnswer{}).Count(&ov.TotalAnswers).Error; err != nil {
		response.Internal(c, "Could not load dashboard stats.")
		return
	}
	if err := h.DB.Model(&models.AIAnalysis{}).Count(&ov.AIAnalyses).Error; err != nil {
		response.Internal(c, "Could not load dashboard stats.")
		return
	}
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if err := h.DB.Model(&models.DailyActivity{}).
		Where("date >= ?", startOfDay).Distinct("student_id").
		Count(&ov.ActiveStudentsToday).Error; err != nil {
		response.Internal(c, "Could not load dashboard stats.")
		return
	}
	if err := h.DB.Model(&models.Quiz{}).
		Where("status = ?", "SUBMITTED").
		Select("COALESCE(AVG(score),0)").Scan(&ov.AvgScore).Error; err != nil {
		response.Internal(c, "Could not load dashboard stats.")
		return
	}
	if err := h.DB.Model(&models.User{}).
		Order("created_at desc").Limit(5).Scan(&ov.RecentRegistrations).Error; err != nil {
		response.Internal(c, "Could not load dashboard stats.")
		return
	}
	response.OK(c, ov)
}

type UserRow struct {
	ID             uint        `json:"id"`
	Name           string      `json:"name"`
	Email          string      `json:"email"`
	Role           models.Role `json:"role"`
	EmailVerified  bool        `json:"emailVerified"`
	Onboarded      bool        `json:"onboarded"`
	QuizCount      int64       `json:"quizCount"`
	SubmittedCount int64       `json:"submittedCount"`
	AvgScore       float64     `json:"avgScore"`
	XP             int64       `json:"xp"`
	MasteredTopics int64       `json:"masteredTopics"`
	LastActiveAt   *time.Time  `json:"lastActiveAt"`
	CreatedAt      time.Time   `json:"createdAt"`
}

func (h *Handler) Users(c *gin.Context) {
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	const query = `
SELECT u.id, u.name, u.email, u.role, u.email_verified,
       (sp.education_level_id IS NOT NULL AND sp.class_level_id IS NOT NULL) AS onboarded,
       COUNT(DISTINCT q.id) AS quiz_count,
       COUNT(DISTINCT q.id) FILTER (WHERE q.status = 'SUBMITTED') AS submitted_count,
       COALESCE(AVG(q.score) FILTER (WHERE q.status = 'SUBMITTED'), 0) AS avg_score,
       COALESCE(SUM(x.amount), 0) AS xp,
       (SELECT COUNT(*) FROM knowledge_states ks WHERE ks.student_id = u.id AND ks.state = 'STRONG') AS mastered_topics,
       MAX(x.created_at) AS last_active_at,
       u.created_at
FROM users u
LEFT JOIN student_profiles sp ON sp.user_id = u.id
LEFT JOIN quizzes q ON q.student_id = u.id
LEFT JOIN xp_transactions x ON x.student_id = u.id
WHERE u.deleted_at IS NULL AND (? = '' OR LOWER(u.name) LIKE ? OR LOWER(u.email) LIKE ?)
GROUP BY u.id, sp.education_level_id, sp.class_level_id
ORDER BY u.created_at DESC`

	var rows []UserRow
	if err := h.DB.Raw(query, q, "%"+q+"%", "%"+q+"%").Scan(&rows).Error; err != nil {
		response.Internal(c, "Could not load users.")
		return
	}
	response.OK(c, rows)
}

// DeleteUser soft-deletes an account. The row and its history stay in the
// database (deleted_at is set) so no data is actually lost; GORM then
// excludes it from normal queries and its email becomes reusable.
func (h *Handler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.BadRequest(c, "Invalid user id.")
		return
	}
	selfID, _ := middleware.UserID(c)
	if selfID == uint(id) {
		response.Forbidden(c, "You cannot delete your own account.")
		return
	}

	var user models.User
	err = h.DB.First(&user, uint(id)).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "User not found.")
			return
		}
		response.Internal(c, "Could not load user.")
		return
	}
	if user.Role == models.RoleAdmin {
		response.Forbidden(c, "Admin accounts cannot be deleted.")
		return
	}
	if err := h.DB.Delete(&user).Error; err != nil {
		response.Internal(c, "Could not delete user.")
		return
	}
	response.OK(c, gin.H{"deleted": true, "id": user.ID})
}
