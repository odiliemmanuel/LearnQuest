package database

import (
	"context"
	"log"
	"time"

	"github.com/learnquest/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.EmailVerification{},
		&models.OutboxEvent{},
		&models.StudentProfile{},
		&models.EducationLevel{},
		&models.ClassLevel{},
		&models.Term{},
		&models.Subject{},
		&models.ClassSubject{},
		&models.Topic{},
		&models.Lesson{},
		&models.Question{},
		&models.QuestionOption{},
		&models.Quiz{},
		&models.StudentAnswer{},
		&models.AIAnalysis{},
		&models.KnowledgeState{},
		&models.StudentSubjectProgress{},
		&models.Recommendation{},
		&models.XPTransaction{},
		&models.Badge{},
		&models.StudentBadge{},
		&models.StudentChallenge{},
		&models.DailyActivity{},
		&models.Challenge{},
		&models.LibraryNote{},
	); err != nil {
		return err
	}
	// Replace the pre-soft-delete unique email index with the composite
	// (email, deleted_at) index so a soft-deleted account's email can be
	// registered again. AutoMigrate creates the new index; the old one is
	// dropped here if it still exists.
	if err := db.Exec(`DROP INDEX IF EXISTS idx_users_email`).Error; err != nil {
		return err
	}

	// daily_activities' unique index was declared on student_id alone, which
	// capped each student at a single row for all time — so daily activity
	// (and the questions_daily / streak_days challenges) silently stopped
	// recording after the first day. AutoMigrate adds the correct composite
	// (student_id, date) index; drop the old single-column one.
	if err := db.Exec(`DROP INDEX IF EXISTS uidx_std_date`).Error; err != nil {
		return err
	}
	return db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uidx_std_date ON daily_activities (student_id, date)`).Error
}

func Count(db *gorm.DB, model any) (int64, error) {
	var n int64
	err := db.Model(model).Count(&n).Error
	return n, err
}

var _ = log.Printf
