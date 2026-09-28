package student

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

func (h *Handler) GetProfile(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	var user models.User
	if err := h.DB.
		Preload("StudentProfile.EducationLevel").
		Preload("StudentProfile.ClassLevel").
		First(&user, userID).Error; err != nil {
		response.NotFound(c, "Student not found.")
		return
	}
	onboarded := false
	if user.StudentProfile != nil {
		onboarded = user.StudentProfile.Onboarded()
	}
	response.OK(c, gin.H{"profile": user.StudentProfile, "onboarded": onboarded})
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	var in authProfileInput
	if err := c.ShouldBindJSON(&in); err != nil || (in.EducationLevelID == nil && in.ClassLevelID == nil) {
		response.BadRequest(c, "Provide at least educationLevelId or classLevelId.")
		return
	}
	var profile models.StudentProfile
	if err := h.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		response.NotFound(c, "Student profile not found.")
		return
	}
	if in.EducationLevelID != nil {
		var level models.EducationLevel
		if err := h.DB.First(&level, *in.EducationLevelID).Error; err != nil {
			response.BadRequest(c, "Unknown education level.")
			return
		}
		profile.EducationLevelID = in.EducationLevelID
	}
	if in.ClassLevelID != nil {
		var class models.ClassLevel
		if err := h.DB.First(&class, *in.ClassLevelID).Error; err != nil {
			response.BadRequest(c, "Unknown class.")
			return
		}
		if profile.EducationLevelID != nil && class.EducationLevelID != *profile.EducationLevelID {
			response.BadRequest(c, "Class does not belong to the selected education level.")
			return
		}
		profile.ClassLevelID = in.ClassLevelID
	}
	if err := h.DB.Save(&profile).Error; err != nil {
		response.Internal(c, "Could not update profile.")
		return
	}
	h.DB.Preload("EducationLevel").Preload("ClassLevel").First(&profile, profile.ID)
	response.OK(c, gin.H{"profile": profile, "onboarded": profile.Onboarded()})
}

type authProfileInput struct {
	EducationLevelID *uint `json:"educationLevelId"`
	ClassLevelID     *uint `json:"classLevelId"`
}
