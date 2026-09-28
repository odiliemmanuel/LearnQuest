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
	return db.AutoMigrate(
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
		&models.DailyActivity{},
		&models.Challenge{},
	)
}

func Count(db *gorm.DB, model any) (int64, error) {
	var n int64
	err := db.Model(model).Count(&n).Error
	return n, err
}

var _ = log.Printf
