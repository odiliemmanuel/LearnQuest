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

// bookFile is the on-disk shape of content/books/*.json. Each book is split
// into chapters; every chapter becomes one LibraryNote under the given subject.
type bookFile struct {
	Book     string        `json:"book"`
	Author   string        `json:"author"`
	Subject  string        `json:"subject"`
	Source   string        `json:"source"`
	Chapters []bookChapter `json:"chapters"`
}

type bookChapter struct {
	Chapter string `json:"chapter"`
	Content string `json:"content"`
}

// SeedBookNotes ingests full public-domain novels stored under
// content/books/ as per-chapter LibraryNotes, so students can read whole
// textbooks inside the Library. Idempotent per (subject, topic).
func SeedBookNotes(db *gorm.DB) error {
	dir := filepath.Join(contentDir(), "books")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var f bookFile
		if err := json.Unmarshal(data, &f); err != nil {
			return fmt.Errorf("seed: bad book json in %s: %w", e.Name(), err)
		}
		subject := strings.TrimSpace(f.Subject)
		if subject == "" {
			subject = f.Book
		}
		for _, ch := range f.Chapters {
			topic := fmt.Sprintf("Chapter %s", ch.Chapter)
			title := fmt.Sprintf("%s — %s", f.Book, topic)
			var existing models.LibraryNote
			err := db.Where("subject = ? AND topic = ?", subject, topic).First(&existing).Error
			if err == nil {
				continue
			}
			if err := db.Create(&models.LibraryNote{
				Subject: subject, Topic: topic, Title: title, Content: ch.Content,
			}).Error; err != nil {
				return fmt.Errorf("seed: book %s chapter %s: %w", f.Book, ch.Chapter, err)
			}
		}
	}
	return nil
}

// SeedLibraryNotes inserts a few hand-written starter study notes so the
// in-app Library is not empty before scripts/textbook_fetcher covers the
// rest of the curriculum. Idempotent per (subject, topic).
func SeedLibraryNotes(db *gorm.DB) error {
	notes := []models.LibraryNote{
		{
			Subject: "Physics", Topic: "Waves", Title: "Waves — Key Concepts and the Wave Equation",
			Content: `A wave carries energy from one place to another without carrying matter along with it. Every wave has four properties you need to know: amplitude (the height of the wave from its resting position, which determines how much energy it carries), wavelength — written as the Greek letter λ (the distance between two identical points on the wave, such as crest to crest), frequency — written as f (how many complete waves pass a point every second, measured in Hertz), and period — written as T (the time taken for one complete wave, and always equal to 1/f).

Waves fall into two broad types. Mechanical waves, like sound and water waves, need a medium — particles of air, water, or a solid — to travel through, because they work by making those particles vibrate. Electromagnetic waves, like light and radio waves, need no medium at all and can travel through empty space.

The single most useful formula for this topic is the wave equation: v = fλ, where v is the wave's speed. It tells you that speed depends only on frequency and wavelength — if one goes up and the other is held constant, speed goes up too.

Worked example: A wave has a frequency of 50 Hz and a wavelength of 4 m. Its speed is v = fλ = 50 × 4 = 200 m/s.

WAEC/NECO tip: examiners often give you two of the three values (v, f, λ) and ask for the third — always start by writing down v = fλ, then rearrange before substituting numbers.`,
		},
		{
			Subject: "Basic Science", Topic: "Living and Non-Living Things", Title: "Living and Non-Living Things — Characteristics of Life",
			Content: `Everything around us can be sorted into two groups: living things and non-living things. Living things are organisms that carry out life processes — they are born, they grow, and eventually they die. Non-living things, like a stone, a chair, or a bottle of water, do none of this on their own.

There are seven characteristics that, together, define a living thing. Growth means increasing permanently in size. Reproduction means producing new individuals of the same kind, so the species continues. Respiration is the process of releasing energy from food, usually using oxygen. Excretion is getting rid of waste products the body produces. Nutrition (or feeding) is taking in and using food for energy and growth. Movement is a change in position of the whole organism or part of it. Irritability (or sensitivity) is the ability to detect and respond to changes in the surroundings — like a plant's leaves turning toward sunlight.

A common exam trick is a one-off characteristic that looks biological but isn't — for example, a toy robot that "moves" is still non-living, because it can't reproduce, grow, or feed itself; it only does what it's programmed or powered to do.

WAEC/NECO tip: if a question asks you to prove something is alive, don't rely on just one characteristic — mention at least two or three (for example, growth and reproduction) since a single shared trait isn't always conclusive.`,
		},
	}

	for _, n := range notes {
		var existing models.LibraryNote
		err := db.Where("subject = ? AND topic = ?", n.Subject, n.Topic).First(&existing).Error
		if err == nil {
			continue
		}
		if err := db.Create(&n).Error; err != nil {
			return err
		}
	}
	return nil
}
