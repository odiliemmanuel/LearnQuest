package seed

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/learnquest/backend/internal/models"
	"gorm.io/gorm"
)

// Seed populates reference data (levels, classes, terms, badges, challenges)
// and the curriculum from JSON content files. It is idempotent.
func Seed(db *gorm.DB) error {
	var count int64
	db.Model(&models.EducationLevel{}).Count(&count)
	if count > 0 {
		log.Println("seed: education levels already present — skipping curriculum seed. Set SEED_FORCE=1 to reseed.")
		if os.Getenv("SEED_FORCE") == "1" {
			return forceSeed(db)
		}
		return nil
	}
	log.Println("seed: starting curriculum seed...")

	if err := seedReference(db); err != nil {
		return err
	}
	contentDir := contentDir()
	entries, err := os.ReadDir(contentDir)
	if err != nil {
		return fmt.Errorf("seed: cannot read content dir %s: %w", contentDir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(contentDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var file CurriculumFile
		if err := json.Unmarshal(data, &file); err != nil {
			return fmt.Errorf("seed: bad json in %s: %w", e.Name(), err)
		}
		for _, classEntry := range file.Curriculum {
			if err := loadClassEntry(db, classEntry); err != nil {
				return fmt.Errorf("seed: failed %s: %w", e.Name(), err)
			}
		}
	}

	if err := seedBadges(db); err != nil {
		return err
	}
	if err := seedChallenges(db); err != nil {
		return err
	}
	log.Println("seed: done.")
	return nil
}

func forceSeed(db *gorm.DB) error {
	db.Where("1 = 1").Delete(&models.Topic{})
	db.Where("1 = 1").Delete(&models.Lesson{})
	db.Where("1 = 1").Delete(&models.Question{})
	db.Where("1 = 1").Delete(&models.QuestionOption{})
	db.Where("1 = 1").Delete(&models.ClassSubject{})
	db.Where("1 = 1").Delete(&models.Subject{})
	return Seed(db)
}

func seedReference(db *gorm.DB) error {
	levels := []models.EducationLevel{
		{Name: "Primary", Code: "PRI", Sequence: 1},
		{Name: "Secondary", Code: "SEC", Sequence: 2},
		{Name: "University", Code: "UNI", Sequence: 3},
	}
	for i := range levels {
		if err := db.Create(&levels[i]).Error; err != nil {
			return err
		}
	}

	primary := []models.ClassLevel{}
	for i := 1; i <= 6; i++ {
		name := fmt.Sprintf("Primary %d", i)
		primary = append(primary, models.ClassLevel{Name: name, Code: name, Sequence: i, EducationLevelID: levels[0].ID})
	}
	secondary := []string{"JSS1", "JSS2", "JSS3", "SS1", "SS2", "SS3"}
	for i, name := range secondary {
		primary = append(primary, models.ClassLevel{Name: name, Code: name, Sequence: i + 1, EducationLevelID: levels[1].ID})
	}
	for i := range primary {
		if err := db.Create(&primary[i]).Error; err != nil {
			return err
		}
	}

	terms := []models.Term{
		{Name: "First Term", Code: "T1", Sequence: 1},
		{Name: "Second Term", Code: "T2", Sequence: 2},
		{Name: "Third Term", Code: "T3", Sequence: 3},
	}
	for i := range terms {
		if err := db.Create(&terms[i]).Error; err != nil {
			return err
		}
	}
	log.Println("seed: levels, classes, terms created.")
	return nil
}

func seedBadges(db *gorm.DB) error {
	badges := []models.Badge{
		{Code: "first_quiz", Name: "First Steps", Description: "Submitted your first quiz."},
		{Code: "perfect_quiz", Name: "Perfect Score", Description: "Scored 100% on a quiz."},
		{Code: "rising_star", Name: "Rising Star", Description: "Got 10 answers correct in total."},
		{Code: "calc_wizard", Name: "Calculation Wizard", Description: "Got 5 calculation questions correct."},
		{Code: "streak_3", Name: "On a Roll", Description: "Maintained a 3-day streak."},
		{Code: "streak_7", Name: "A Whole Week", Description: "Maintained a 7-day streak."},
		{Code: "recovery_hero", Name: "Comeback Kid", Description: "Completed 3 recovery missions."},
		{Code: "subject_explorer", Name: "Subject Explorer", Description: "Mastered 3 topics in one subject."},
		{Code: "physics_beginner", Name: "Physics Beginner", Description: "Scored 60%+ on a Physics quiz."},
	}
	for i := range badges {
		if err := db.Create(&badges[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedChallenges(db *gorm.DB) error {
	challenges := []models.Challenge{
		{Code: "questions_daily", Title: "Daily Drill", Description: "Answer 10 questions today.", Metric: "questions_daily", Target: 10, XPReward: 30, Active: true},
		{Code: "perfect_quiz_daily", Title: "Flawless", Description: "Score 100% on a quiz today.", Metric: "perfect_quiz_daily", Target: 1, XPReward: 40, Active: true},
		{Code: "streak_3", Title: "Three in a Row", Description: "Reach a 3-day learning streak.", Metric: "streak_days", Target: 3, XPReward: 50, Active: true},
		{Code: "recovery_missions", Title: "Comeback", Description: "Complete 3 recovery missions.", Metric: "recovery_missions", Target: 3, XPReward: 60, Active: true},
	}
	for i := range challenges {
		if err := db.Create(&challenges[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

// — content loading —

type CurriculumFile struct {
	Curriculum []ClassTermEntry `json:"curriculum"`
}

type ClassTermEntry struct {
	Class    string         `json:"class"`
	Term     string         `json:"term"`
	Subjects []SubjectEntry `json:"subjects"`
}

type SubjectEntry struct {
	Name   string       `json:"name"`
	Code   string       `json:"code"`
	Topics []TopicEntry `json:"topics"`
}

type TopicEntry struct {
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	EstimatedMinutes int             `json:"estimatedMinutes"`
	Lesson           *LessonEntry    `json:"lesson,omitempty"`
	Questions        []QuestionEntry `json:"questions,omitempty"`
}

type LessonEntry struct {
	Title        string         `json:"title"`
	Introduction string         `json:"introduction"`
	Explanation  string         `json:"explanation"`
	Examples     []ExampleEntry `json:"examples,omitempty"`
	KeyPoints    []string       `json:"keyPoints,omitempty"`
	Formulas     []FormulaEntry `json:"formulas,omitempty"`
}

type ExampleEntry struct {
	Title    string `json:"title"`
	Problem  string `json:"problem"`
	Solution string `json:"solution"`
}

type FormulaEntry struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
	Meaning    string `json:"meaning"`
}

type QuestionEntry struct {
	Type            string        `json:"type"`
	Prompt          string        `json:"prompt"`
	Difficulty      int           `json:"difficulty"`
	Marks           int           `json:"marks"`
	Options         []OptionEntry `json:"options,omitempty"`
	CorrectKey      string        `json:"correctKey,omitempty"`
	CorrectValue    string        `json:"correctValue,omitempty"`
	CorrectUnit     string        `json:"correctUnit,omitempty"`
	Tolerance       float64       `json:"tolerance,omitempty"`
	ExpectedWorking string        `json:"expectedWorking,omitempty"`
	Explanation     string        `json:"explanation,omitempty"`
}

type OptionEntry struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

func loadClassEntry(db *gorm.DB, entry ClassTermEntry) error {
	var classLevel models.ClassLevel
	if err := db.Where("code = ?", entry.Class).First(&classLevel).Error; err != nil {
		return fmt.Errorf("class %q not found", entry.Class)
	}
	var term models.Term
	if err := db.Where("name = ?", entry.Term).First(&term).Error; err != nil {
		return fmt.Errorf("term %q not found", entry.Term)
	}
	for _, sub := range entry.Subjects {
		if err := loadSubject(db, classLevel.ID, term.ID, sub); err != nil {
			return err
		}
	}
	return nil
}

func loadSubject(db *gorm.DB, classID, termID uint, sub SubjectEntry) error {
	var subject models.Subject
	if err := db.Where("name = ?", sub.Name).First(&subject).Error; err != nil {
		subject = models.Subject{Name: sub.Name, Code: sub.Code}
		if sub.Code == "" {
			subject.Code = deriveCode(sub.Name)
		}
		if err := db.Create(&subject).Error; err != nil {
			return err
		}
	}
	// Register subject for the class (used by the "subjects for class" browser).
	var cs int64
	db.Model(&models.ClassSubject{}).Where("class_level_id = ? AND subject_id = ?", classID, subject.ID).Count(&cs)
	if cs == 0 {
		db.Create(&models.ClassSubject{ClassLevelID: classID, SubjectID: subject.ID})
	}

	for idx, t := range sub.Topics {
		topic := models.Topic{
			Name:             t.Name,
			Description:      t.Description,
			Sequence:         idx + 1,
			EstimatedMinutes: defaultMinutes(t.EstimatedMinutes),
			ClassLevelID:     classID,
			TermID:           termID,
			SubjectID:        subject.ID,
		}
		if err := db.Create(&topic).Error; err != nil {
			return err
		}
		if t.Lesson != nil {
			lesson := models.Lesson{
				TopicID:      topic.ID,
				Title:        t.Lesson.Title,
				Introduction: t.Lesson.Introduction,
				Explanation:  t.Lesson.Explanation,
				Examples:     toExamples(t.Lesson.Examples),
				KeyPoints:    t.Lesson.KeyPoints,
				Formulas:     toFormulas(t.Lesson.Formulas),
			}
			if lesson.Title == "" {
				lesson.Title = t.Name
			}
			if err := db.Create(&lesson).Error; err != nil {
				return err
			}
		}
		for qi, q := range t.Questions {
			if err := createQuestion(db, topic.ID, qi, q); err != nil {
				return err
			}
		}
	}
	return nil
}

func createQuestion(db *gorm.DB, topicID uint, seq int, q QuestionEntry) error {
	qtype := strings.ToUpper(strings.TrimSpace(q.Type))
	if qtype == "" {
		qtype = "MCQ"
	}
	marks := q.Marks
	if marks <= 0 {
		marks = 1
	}
	difficulty := q.Difficulty
	if difficulty <= 0 {
		difficulty = 1
	}
	question := models.Question{
		TopicID:         topicID,
		Type:            qtype,
		Prompt:          q.Prompt,
		Difficulty:      difficulty,
		Marks:           marks,
		CorrectValue:    q.CorrectValue,
		CorrectUnit:     q.CorrectUnit,
		Tolerance:       q.Tolerance,
		ExpectedWorking: q.ExpectedWorking,
		Explanation:     q.Explanation,
	}
	if err := db.Create(&question).Error; err != nil {
		return err
	}
	if qtype == "MCQ" {
		for i, o := range q.Options {
			opt := models.QuestionOption{
				QuestionID: question.ID,
				Key:        o.Key,
				Text:       o.Text,
				IsCorrect:  o.Key == q.CorrectKey,
				Order:      i + 1,
			}
			if opt.Key == "" {
				opt.Key = string(rune('A' + i))
			}
			if err := db.Create(&opt).Error; err != nil {
				return err
			}
			if opt.IsCorrect {
				question.CorrectOptionID = &opt.ID
			}
		}
		if question.CorrectOptionID != nil {
			db.Save(&question)
		}
	}
	_ = seq
	return nil
}

func toExamples(es []ExampleEntry) []models.Example {
	out := make([]models.Example, 0, len(es))
	for _, e := range es {
		out = append(out, models.Example{Title: e.Title, Problem: e.Problem, Solution: e.Solution})
	}
	return out
}

func toFormulas(fs []FormulaEntry) []models.Formula {
	out := make([]models.Formula, 0, len(fs))
	for _, f := range fs {
		out = append(out, models.Formula{Name: f.Name, Expression: f.Expression, Meaning: f.Meaning})
	}
	return out
}

func defaultMinutes(m int) int {
	if m <= 0 {
		return 20
	}
	return m
}

func contentDir() string {
	// In a built binary the content folder is expected next to the working dir.
	dir := os.Getenv("SEED_CONTENT_DIR")
	if dir != "" {
		return dir
	}
	return "internal/seed/content"
}

func deriveCode(name string) string {
	words := strings.Fields(strings.TrimSpace(name))
	switch len(words) {
	case 0:
		return "SUBJ"
	case 1:
		s := strings.ToUpper(words[0])
		if len(s) > 4 {
			return s[:4]
		}
		return s
	default:
		code := ""
		for _, w := range words {
			code += strings.ToUpper(w[:1])
		}
		if len(code) > 4 {
			code = code[:4]
		}
		return code
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
