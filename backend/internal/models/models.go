package models

import "gorm.io/gorm"
import "time"

type Role string

const (
	RoleStudent Role = "STUDENT"
	RoleAdmin   Role = "ADMIN"
)

type User struct {
	ID             uint            `gorm:"primarykey" json:"id"`
	Name           string          `gorm:"size:100;not null" json:"name"`
	Email          string          `gorm:"size:150;index:idx_users_email_deleted,priority:1;not null" json:"email"`
	PasswordHash   string          `gorm:"size:255;not null" json:"-"`
	Role           Role            `gorm:"size:20;default:STUDENT" json:"role"`
	EmailVerified  bool            `gorm:"default:false" json:"emailVerified"`
	StudentProfile *StudentProfile `json:"profile,omitempty"`
	DeletedAt      gorm.DeletedAt  `gorm:"index:idx_users_email_deleted,priority:2" json:"-"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

// EmailVerification is owned by the auth service. Codes are never stored in
// plaintext and each resend invalidates any previous unused code.
type EmailVerification struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	UserID       uint       `gorm:"index;not null" json:"userId"`
	Purpose      string     `gorm:"size:40;index;not null" json:"purpose"`
	CodeHash     string     `gorm:"size:64;not null" json:"-"`
	AttemptCount int        `gorm:"default:0" json:"-"`
	ExpiresAt    time.Time  `gorm:"index;not null" json:"expiresAt"`
	ResendAfter  time.Time  `json:"resendAfter"`
	UsedAt       *time.Time `gorm:"index" json:"usedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// OutboxEvent makes broker publication durable: auth data and its event are
// committed together, then a background dispatcher publishes the event.
type OutboxEvent struct {
	ID           string     `gorm:"primaryKey;size:36" json:"id"`
	Topic        string     `gorm:"size:160;index;not null" json:"topic"`
	Payload      string     `gorm:"type:text;not null" json:"payload"`
	AttemptCount int        `gorm:"default:0" json:"attemptCount"`
	AvailableAt  time.Time  `gorm:"index;not null" json:"availableAt"`
	PublishedAt  *time.Time `gorm:"index" json:"publishedAt,omitempty"`
	LastError    string     `gorm:"type:text" json:"lastError,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type StudentProfile struct {
	ID               uint            `gorm:"primarykey" json:"id"`
	UserID           uint            `gorm:"uniqueIndex;not null" json:"userId"`
	EducationLevel   *EducationLevel `json:"educationLevel,omitempty"`
	EducationLevelID *uint           `json:"educationLevelId"`
	ClassLevel       *ClassLevel     `json:"classLevel,omitempty"`
	ClassLevelID     *uint           `json:"classLevelId"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

func (s StudentProfile) Onboarded() bool {
	return s.EducationLevelID != nil && s.ClassLevelID != nil
}

type EducationLevel struct {
	ID       uint   `gorm:"primarykey" json:"id"`
	Name     string `gorm:"size:100;not null" json:"name"`
	Code     string `gorm:"size:50;uniqueIndex" json:"code"`
	Sequence int    `json:"sequence"`
}

type ClassLevel struct {
	ID               uint   `gorm:"primarykey" json:"id"`
	Name             string `gorm:"size:100;not null" json:"name"`
	Code             string `gorm:"size:50" json:"code"`
	Sequence         int    `json:"sequence"`
	EducationLevelID uint   `gorm:"not null" json:"educationLevelId"`
}

type Term struct {
	ID       uint   `gorm:"primarykey" json:"id"`
	Name     string `gorm:"size:50;not null" json:"name"`
	Code     string `gorm:"size:50" json:"code"`
	Sequence int    `json:"sequence"`
}

type Subject struct {
	ID   uint   `gorm:"primarykey" json:"id"`
	Name string `gorm:"size:100;not null" json:"name"`
	Code string `gorm:"size:50" json:"code"`
}

type ClassSubject struct {
	ID           uint `gorm:"primarykey" json:"id"`
	ClassLevelID uint `gorm:"uniqueIndex:idx_class_subject;not null" json:"classLevelId"`
	SubjectID    uint `gorm:"uniqueIndex:idx_class_subject;not null" json:"subjectId"`
}

type Topic struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	Name             string    `gorm:"size:150;not null" json:"name"`
	Code             string    `gorm:"size:80" json:"code"`
	Description      string    `gorm:"type:text" json:"description"`
	Sequence         int       `json:"sequence"`
	EstimatedMinutes int       `json:"estimatedMinutes"`
	ClassLevelID     uint      `gorm:"not null" json:"classLevelId"`
	TermID           uint      `gorm:"not null" json:"termId"`
	SubjectID        uint      `gorm:"not null" json:"subjectId"`
	CreatedAt        time.Time `json:"createdAt"`
}

type Example struct {
	Title    string `json:"title"`
	Problem  string `json:"problem"`
	Solution string `json:"solution"`
}

type Formula struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
	Meaning    string `json:"meaning"`
}

type Lesson struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	TopicID      uint      `gorm:"uniqueIndex;not null" json:"topicId"`
	Title        string    `gorm:"size:200;not null" json:"title"`
	Introduction string    `gorm:"type:text" json:"introduction"`
	Explanation  string    `gorm:"type:text" json:"explanation"`
	Examples     []Example `gorm:"type:jsonb;serializer:json" json:"examples"`
	KeyPoints    []string  `gorm:"type:jsonb;serializer:json" json:"keyPoints"`
	Formulas     []Formula `gorm:"type:jsonb;serializer:json" json:"formulas"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Question struct {
	ID              uint             `gorm:"primarykey" json:"id"`
	TopicID         uint             `gorm:"index;not null" json:"topicId"`
	Type            string           `gorm:"size:20;not null" json:"type"` // MCQ | CALCULATION
	Prompt          string           `gorm:"type:text;not null" json:"prompt"`
	Difficulty      int              `gorm:"default:1" json:"difficulty"`
	Marks           int              `gorm:"default:1" json:"marks"`
	CorrectOptionID *uint            `json:"-"`
	CorrectValue    string           `json:"-"`
	CorrectUnit     string           `json:"-"`
	Tolerance       float64          `gorm:"default:0.5" json:"-"`
	ExpectedWorking string           `gorm:"type:text" json:"-"`
	Explanation     string           `gorm:"type:text" json:"explanation"`
	Options         []QuestionOption `json:"options"`
}

type QuestionOption struct {
	ID         uint   `gorm:"primarykey" json:"id"`
	QuestionID uint   `gorm:"index;not null" json:"-"`
	Key        string `gorm:"size:5" json:"key"`
	Text       string `gorm:"type:text" json:"text"`
	IsCorrect  bool   `json:"-"`
	Order      int    `json:"order"`
}

type Quiz struct {
	ID           uint            `gorm:"primarykey" json:"id"`
	StudentID    uint            `gorm:"index;not null" json:"studentId"`
	TopicID      uint            `gorm:"index;not null" json:"topicId"`
	Status       string          `gorm:"size:20;default:STARTED" json:"status"` // STARTED | SUBMITTED
	IsRecovery   bool            `gorm:"default:false" json:"isRecovery"`
	TotalMarks   int             `json:"totalMarks"`
	ScoreMarks   int             `json:"scoreMarks"`
	Score        float64         `json:"score"`
	CorrectCount int             `json:"correctCount"`
	TotalCount   int             `json:"totalCount"`
	StartedAt    *time.Time      `json:"startedAt"`
	SubmittedAt  *time.Time      `json:"submittedAt"`
	Answers      []StudentAnswer `json:"answers,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
}

type StudentAnswer struct {
	ID               uint        `gorm:"primarykey" json:"id"`
	QuizID           uint        `gorm:"index;not null" json:"quizId"`
	StudentID        uint        `gorm:"index;not null" json:"studentId"`
	QuestionID       uint        `gorm:"index;not null" json:"questionId"`
	SelectedOptionID *uint       `json:"selectedOptionId"`
	StudentAnswer    string      `gorm:"type:text" json:"studentAnswer"`
	Working          string      `gorm:"type:text" json:"working"`
	IsCorrect        bool        `json:"isCorrect"`
	ReceivedMarks    int         `json:"receivedMarks"`
	AIAnalysis       *AIAnalysis `gorm:"-" json:"aiAnalysis,omitempty"`
	CreatedAt        time.Time   `json:"createdAt"`
}

type KnowledgeState struct {
	ID             uint       `gorm:"primarykey" json:"id"`
	StudentID      uint       `gorm:"uniqueIndex:uidx_student_topic;not null" json:"studentId"`
	TopicID        uint       `gorm:"uniqueIndex:uidx_student_topic;not null" json:"topicId"`
	Attempts       int        `json:"attempts"`
	CorrectCount   int        `json:"correctCount"`
	Mastery        float64    `json:"mastery"`
	Confidence     float64    `json:"confidence"`
	State          string     `gorm:"size:20" json:"state"` // STRONG | IMPROVING | WEAK | NEW
	LastAssessedAt *time.Time `json:"lastAssessedAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type StudentSubjectProgress struct {
	ID             uint       `gorm:"primarykey" json:"id"`
	StudentID      uint       `gorm:"uniqueIndex:uidx_student_subject;not null" json:"studentId"`
	SubjectID      uint       `gorm:"uniqueIndex:uidx_student_subject;not null" json:"subjectId"`
	Subject        *Subject   `json:"subject,omitempty"`
	TopicsTotal    int        `json:"topicsTotal"`
	TopicsMastered int        `json:"topicsMastered"`
	CorrectCount   int        `json:"correctCount"`
	TotalCount     int        `json:"totalCount"`
	Percentage     float64    `json:"percentage"`
	LastAssessedAt *time.Time `json:"lastAssessedAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type Recommendation struct {
	ID          uint       `gorm:"primarykey" json:"id"`
	StudentID   uint       `gorm:"index;not null" json:"studentId"`
	TopicID     uint       `gorm:"index;not null" json:"topicId"`
	Topic       *Topic     `json:"topic,omitempty"`
	Kind        string     `gorm:"size:20" json:"kind"` // WEAK | FORGOTTEN | RECOVERY
	Reason      string     `gorm:"type:text" json:"reason"`
	Priority    int        `json:"priority"`
	Status      string     `gorm:"size:20;default:OPEN" json:"status"` // OPEN | COMPLETED | DISMISSED
	CompletedAt *time.Time `json:"completedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type XPTransaction struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	StudentID uint      `gorm:"index;not null" json:"studentId"`
	Amount    int       `json:"amount"`
	Reason    string    `gorm:"size:150" json:"reason"`
	Source    string    `gorm:"size:50" json:"source"`
	RefID     uint      `json:"refId"`
	CreatedAt time.Time `json:"createdAt"`
}

type Badge struct {
	ID          uint   `gorm:"primarykey" json:"id"`
	Code        string `gorm:"size:60;uniqueIndex" json:"code"`
	Name        string `gorm:"size:100" json:"name"`
	Description string `gorm:"size:255" json:"description"`
}

type StudentBadge struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	StudentID uint      `gorm:"uniqueIndex:uidx_std_badge;not null" json:"studentId"`
	BadgeID   uint      `gorm:"uniqueIndex:uidx_std_badge;not null" json:"badgeId"`
	Badge     *Badge    `json:"badge,omitempty"`
	EarnedAt  time.Time `json:"earnedAt"`
}

type DailyActivity struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	StudentID     uint      `gorm:"uniqueIndex:uidx_std_date;not null" json:"studentId"`
	Date          time.Time `json:"date"`
	QuestionCount int       `json:"questionCount"`
}

type Challenge struct {
	ID          uint   `gorm:"primarykey" json:"id"`
	Code        string `gorm:"size:60;uniqueIndex" json:"code"`
	Title       string `gorm:"size:120" json:"title"`
	Description string `gorm:"size:255" json:"description"`
	Metric      string `gorm:"size:60" json:"metric"`
	Target      int    `json:"target"`
	XPReward    int    `json:"xpReward"`
	Active      bool   `gorm:"default:true" json:"active"`
}

type StudentChallenge struct {
	ID          uint       `gorm:"primarykey" json:"id"`
	StudentID   uint       `gorm:"uniqueIndex:uidx_std_challenge;not null" json:"studentId"`
	ChallengeID uint       `gorm:"uniqueIndex:uidx_std_challenge;not null" json:"challengeId"`
	Progress    int        `json:"progress"`
	Completed   bool       `gorm:"default:false" json:"completed"`
	CompletedAt *time.Time `json:"completedAt"`
}

type AIAnalysis struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	StudentID    uint      `gorm:"index;not null" json:"studentId"`
	QuestionID   uint      `gorm:"index;not null" json:"questionId"`
	AnswerID     *uint     `gorm:"index" json:"answerId"`
	QuizID       *uint     `gorm:"index" json:"quizId"`
	Kind         string    `gorm:"size:20" json:"kind"` // MISTAKE | WORKING | EXPLANATION | PRACTICE
	Status       string    `gorm:"size:20" json:"status"`
	RequestJSON  string    `gorm:"type:text" json:"-"`
	ResponseJSON string    `gorm:"type:text" json:"-"`
	Error        string    `gorm:"type:text" json:"error,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// LibraryNote is a condensed, original study note for one subject/topic,
// written for LearnQuest (never copied from an existing textbook). Read
// entirely inside the app. Populated by scripts/textbook_fetcher, which
// asks Gemini to write a note for any topic that doesn't have one yet.
type LibraryNote struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Subject   string    `gorm:"size:100;uniqueIndex:idx_subject_topic;not null" json:"subject"`
	Topic     string    `gorm:"size:150;uniqueIndex:idx_subject_topic;not null" json:"topic"`
	Title     string    `gorm:"size:200;not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
