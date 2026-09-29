package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/learnquest/backend/internal/middleware"
	"github.com/learnquest/backend/internal/models"
	"github.com/learnquest/backend/internal/response"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	DB         *gorm.DB
	Secret     string
	OTPSecret  string
	TTL        time.Duration
	AdminEmail string
}

func NewService(db *gorm.DB, secret, otpSecret string, ttl time.Duration, adminEmail string) *Service {
	return &Service{DB: db, Secret: secret, OTPSecret: otpSecret, TTL: ttl, AdminEmail: adminEmail}
}

type RegisterInput struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email,max=150"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type VerifyEmailInput struct {
	Email string `json:"email" binding:"required,email,max=150"`
	Code  string `json:"code" binding:"required,len=6,numeric"`
}

type ResendVerificationInput struct {
	Email string `json:"email" binding:"required,email,max=150"`
}

type ProfileInput struct {
	EducationLevelID *uint `json:"educationLevelId"`
	ClassLevelID     *uint `json:"classLevelId"`
}

type AuthResponse struct {
	Token     string      `json:"token"`
	User      models.User `json:"user"`
	Onboarded bool        `json:"onboarded"`
}

type RegisterResponse struct {
	Email                string `json:"email"`
	VerificationRequired bool   `json:"verificationRequired"`
}

type verificationEmailEvent struct {
	EventID   string    `json:"eventId"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

const (
	verificationPurpose  = "EMAIL_VERIFICATION"
	verificationTopic    = "identity.email_verification_requested"
	verificationTTL      = 10 * time.Minute
	resendCooldown       = time.Minute
	maxVerificationTries = 5
)

func (s *Service) Register(c *gin.Context) {
	var in RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Please provide a valid name, email and password (min 6 characters).")
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	var existing models.User
	if err := s.DB.Where("email = ?", email).First(&existing).Error; err == nil {
		response.Fail(c, http.StatusConflict, "An account with this email already exists.")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Internal(c, "Could not secure password.")
		return
	}

	user := models.User{
		Name:         strings.TrimSpace(in.Name),
		Email:        email,
		PasswordHash: string(hash),
		Role:         models.RoleStudent,
	}
	if s.AdminEmail != "" && strings.EqualFold(s.AdminEmail, email) {
		user.Role = models.RoleAdmin
	}
	tx := s.DB.Begin()
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		response.Internal(c, "Could not create account.")
		return
	}
	if err := tx.Create(&models.StudentProfile{UserID: user.ID}).Error; err != nil {
		tx.Rollback()
		response.Internal(c, "Could not create student profile.")
		return
	}
	if err := s.createVerification(tx, user); err != nil {
		tx.Rollback()
		response.Internal(c, "Could not start email verification.")
		return
	}
	if err := tx.Commit().Error; err != nil {
		response.Internal(c, "Could not create account.")
		return
	}
	response.Created(c, RegisterResponse{Email: user.Email, VerificationRequired: true})
}

func (s *Service) Login(c *gin.Context) {
	var in LoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Please provide your email and password.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	var user models.User
	if err := s.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Unauthorized(c, "Invalid email or password.")
			return
		}
		response.Internal(c, "Something went wrong.")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)) != nil {
		response.Unauthorized(c, "Invalid email or password.")
		return
	}
	if !user.EmailVerified {
		response.Forbidden(c, "Verify your email before signing in.")
		return
	}

	var profile models.StudentProfile
	s.DB.Where("user_id = ?", user.ID).First(&profile)

	token, err := middleware.IssueToken(s.Secret, s.TTL, user.ID, string(user.Role))
	if err != nil {
		response.Internal(c, "Could not create session.")
		return
	}
	response.OK(c, AuthResponse{Token: token, User: user, Onboarded: profile.Onboarded()})
}

func (s *Service) VerifyEmail(c *gin.Context) {
	var in VerifyEmailInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Provide the email address and 6-digit verification code.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	var user models.User
	if err := s.DB.Where("email = ?", email).First(&user).Error; err != nil {
		response.BadRequest(c, "The verification code is invalid or expired.")
		return
	}
	if user.EmailVerified {
		response.OK(c, gin.H{"verified": true})
		return
	}

	var verification models.EmailVerification
	err := s.DB.Where("user_id = ? AND purpose = ? AND used_at IS NULL", user.ID, verificationPurpose).
		Order("created_at desc").First(&verification).Error
	if err != nil || time.Now().After(verification.ExpiresAt) || verification.AttemptCount >= maxVerificationTries {
		response.BadRequest(c, "The verification code is invalid or expired. Request a new code.")
		return
	}

	verification.AttemptCount++
	if !hmac.Equal([]byte(verification.CodeHash), []byte(s.hashCode(user.ID, in.Code))) {
		s.DB.Model(&verification).Update("attempt_count", verification.AttemptCount)
		if verification.AttemptCount >= maxVerificationTries {
			response.TooMany(c, "Too many attempts. Request a new verification code.")
			return
		}
		response.BadRequest(c, "The verification code is invalid or expired.")
		return
	}

	now := time.Now()
	tx := s.DB.Begin()
	if err := tx.Model(&verification).Updates(map[string]any{"used_at": now, "attempt_count": verification.AttemptCount}).Error; err != nil {
		tx.Rollback()
		response.Internal(c, "Could not verify email.")
		return
	}
	if err := tx.Model(&user).Update("email_verified", true).Error; err != nil {
		tx.Rollback()
		response.Internal(c, "Could not verify email.")
		return
	}
	user.EmailVerified = true
	if err := tx.Commit().Error; err != nil {
		response.Internal(c, "Could not verify email.")
		return
	}

	var profile models.StudentProfile
	s.DB.Where("user_id = ?", user.ID).First(&profile)
	token, err := middleware.IssueToken(s.Secret, s.TTL, user.ID, string(user.Role))
	if err != nil {
		response.Internal(c, "Email verified, but could not create a session.")
		return
	}
	response.OK(c, AuthResponse{Token: token, User: user, Onboarded: profile.Onboarded()})
}

func (s *Service) ResendVerification(c *gin.Context) {
	var in ResendVerificationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Provide a valid email address.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	var user models.User
	if err := s.DB.Where("email = ?", email).First(&user).Error; err != nil || user.EmailVerified {
		response.OK(c, gin.H{"sent": true})
		return
	}
	var latest models.EmailVerification
	if err := s.DB.Where("user_id = ? AND purpose = ? AND used_at IS NULL", user.ID, verificationPurpose).
		Order("created_at desc").First(&latest).Error; err == nil && time.Now().Before(latest.ResendAfter) {
		response.TooMany(c, "Wait a minute before requesting another code.")
		return
	}

	tx := s.DB.Begin()
	if err := s.createVerification(tx, user); err != nil {
		tx.Rollback()
		response.Internal(c, "Could not send a new verification code.")
		return
	}
	if err := tx.Commit().Error; err != nil {
		response.Internal(c, "Could not send a new verification code.")
		return
	}
	response.OK(c, gin.H{"sent": true})
}

func (s *Service) createVerification(tx *gorm.DB, user models.User) error {
	now := time.Now()
	if err := tx.Model(&models.EmailVerification{}).
		Where("user_id = ? AND purpose = ? AND used_at IS NULL", user.ID, verificationPurpose).
		Update("used_at", now).Error; err != nil {
		return err
	}
	code, err := generateCode()
	if err != nil {
		return err
	}
	eventID := uuid.NewString()
	verification := models.EmailVerification{
		UserID: user.ID, Purpose: verificationPurpose, CodeHash: s.hashCode(user.ID, code),
		ExpiresAt: now.Add(verificationTTL), ResendAfter: now.Add(resendCooldown),
	}
	if err := tx.Create(&verification).Error; err != nil {
		return err
	}
	payload, err := json.Marshal(verificationEmailEvent{EventID: eventID, Email: user.Email, Name: user.Name, Code: code, ExpiresAt: verification.ExpiresAt})
	if err != nil {
		return err
	}
	return tx.Create(&models.OutboxEvent{ID: eventID, Topic: verificationTopic, Payload: string(payload), AvailableAt: now}).Error
}

func (s *Service) hashCode(userID uint, code string) string {
	mac := hmac.New(sha256.New, []byte(s.OTPSecret))
	_, _ = fmt.Fprintf(mac, "%d:%s:%s", userID, verificationPurpose, code)
	return hex.EncodeToString(mac.Sum(nil))
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
