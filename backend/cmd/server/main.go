package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/learnquest/backend/internal/admin"
	"github.com/learnquest/backend/internal/ai"
	"github.com/learnquest/backend/internal/auth"
	"github.com/learnquest/backend/internal/config"
	"github.com/learnquest/backend/internal/curriculum"
	"github.com/learnquest/backend/internal/database"
	"github.com/learnquest/backend/internal/events"
	"github.com/learnquest/backend/internal/gamification"
	"github.com/learnquest/backend/internal/knowledge"
	"github.com/learnquest/backend/internal/lesson"
	"github.com/learnquest/backend/internal/library"
	"github.com/learnquest/backend/internal/middleware"
	"github.com/learnquest/backend/internal/notifications"
	"github.com/learnquest/backend/internal/outbox"
	"github.com/learnquest/backend/internal/question"
	"github.com/learnquest/backend/internal/quiz"
	"github.com/learnquest/backend/internal/recommendation"
	"github.com/learnquest/backend/internal/response"
	"github.com/learnquest/backend/internal/seed"
	"github.com/learnquest/backend/internal/student"
	"github.com/learnquest/backend/internal/workers"
)

func main() {
	// `-healthcheck` is used by the container healthcheck. The runtime image
	// is distroless, so there is no shell or curl available to probe /health.
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(healthcheck())
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := seed.Seed(db); err != nil {
		log.Fatalf("seed: %v", err)
	}
	if err := seed.SeedLibraryNotes(db); err != nil {
		log.Fatalf("seed library: %v", err)
	}
	if err := seed.SeedBookNotes(db); err != nil {
		log.Fatalf("seed books: %v", err)
	}
	if err := admin.Ensure(db, cfg.AdminEmail); err != nil {
		log.Fatalf("admin: %v", err)
	}

	aiSvc := ai.New(cfg, db)
	aiHandler := ai.NewHandler(aiSvc)

	bus := events.NewBroker(8)
	processor := &workers.QuizProcessor{DB: db, AI: aiSvc, Bus: bus}
	bus.Subscribe(events.QuizCompleted, processor.HandleQuizCompleted)

	authSvc := auth.NewService(db, cfg.JWTSecret, cfg.OTPSecret, cfg.JWTTTL, cfg.AdminEmail)
	quizSvc := &quiz.Service{DB: db, Bus: bus}
	curriculumHandler := &curriculum.Handler{DB: db}
	lessonHandler := &lesson.Handler{DB: db}
	questionHandler := &question.Handler{DB: db}
	knowledgeHandler := &knowledge.Handler{DB: db}
	gamificationHandler := &gamification.Handler{DB: db}
	recommendationHandler := &recommendation.Handler{DB: db}
	studentHandler := &student.Handler{DB: db}
	adminHandler := &admin.Handler{DB: db}
	libraryHandler := &library.Handler{DB: db, IngestKey: cfg.IngestKey}

	gin.SetMode(ginMode(cfg.AppEnv))
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS(cfg.CORSOrigins))

	r.GET("/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			response.Internal(c, "database unavailable")
			return
		}
		if err := sqlDB.PingContext(c.Request.Context()); err != nil {
			response.Internal(c, "database unavailable")
			return
		}
		response.OK(c, gin.H{"status": "ok", "ai": aiSvc.Enabled()})
	})

	api := r.Group("/api/v1")
	{
		public := api.Group("")
		public.POST("/auth/register", authSvc.Register)
		public.POST("/auth/login", authSvc.Login)
		public.POST("/auth/verify-email", authSvc.VerifyEmail)
		public.POST("/auth/resend-verification", authSvc.ResendVerification)

		cur := public.Group("/curriculum")
		cur.GET("/levels", curriculumHandler.Levels)
		cur.GET("/classes", curriculumHandler.Classes)
		cur.GET("/terms", curriculumHandler.Terms)
		cur.GET("/subjects", curriculumHandler.SubjectsAll)
		cur.GET("/subjects-for-class", curriculumHandler.SubjectsForClass)
		cur.GET("/topics", middleware.OptionalAuth(cfg.JWTSecret), curriculumHandler.Topics)
		cur.GET("/topics/:id", middleware.OptionalAuth(cfg.JWTSecret), curriculumHandler.TopicDetail)

		authGroup := api.Group("", middleware.RequireAuth(cfg.JWTSecret))
		authGroup.GET("/student/profile", studentHandler.GetProfile)
		authGroup.PUT("/student/profile", studentHandler.UpdateProfile)
		authGroup.GET("/library", libraryHandler.List)

		authGroup.GET("/lessons/:id", lessonHandler.GetByID)
		authGroup.GET("/topics/:topicId/lesson", lessonHandler.GetByTopic)
		authGroup.GET("/topics/:topicId/questions", questionHandler.ByTopic)

		authGroup.POST("/topics/:topicId/quiz/start", quizSvc.Start)
		authGroup.POST("/quizzes/:id/submit", quizSvc.Submit)
		authGroup.GET("/quizzes/:id/result", quizSvc.Result)

		authGroup.GET("/progress", knowledgeHandler.Progress)
		authGroup.GET("/knowledge-map", knowledgeHandler.KnowledgeMap)

		authGroup.GET("/recommendations", recommendationHandler.List)
		authGroup.GET("/recommendations/recovery", recommendationHandler.Recovery)
		authGroup.POST("/recommendations/complete", recommendationHandler.Complete)

		authGroup.GET("/gamification/summary", gamificationHandler.Summary)
		authGroup.GET("/gamification/badges", gamificationHandler.Badges)
		authGroup.GET("/gamification/challenges", gamificationHandler.Challenges)

		authGroup.POST("/ai/analyze-mistake", aiHandler.AnalyzeMistake)
		authGroup.POST("/ai/analyze-working", aiHandler.AnalyzeWorking)
		authGroup.POST("/ai/explain", aiHandler.Explain)
		authGroup.POST("/ai/generate-practice", aiHandler.GeneratePractice)

		adminGroup := api.Group("/admin", middleware.RequireAuth(cfg.JWTSecret), middleware.RequireAdmin())
		adminGroup.GET("/overview", adminHandler.Overview)
		adminGroup.GET("/users", adminHandler.Users)
		adminGroup.DELETE("/users/:id", adminHandler.DeleteUser)
	}

	// Public, non-versioned endpoints used by scripts/textbook_fetcher
	// (which reads the curriculum and writes generated notes back).
	ingest := r.Group("/api")
	ingest.GET("/curriculum", libraryHandler.CurriculumSync)
	ingest.POST("/library/notes", libraryHandler.Ingest)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	outboxCtx, stopOutbox := context.WithCancel(context.Background())
	defer stopOutbox()
	(&outbox.Dispatcher{DB: db, RabbitMQURL: cfg.RabbitMQURL}).Start(outboxCtx)

	// Hosts without background workers (e.g. a free Render web service) run the
	// email verification consumer inside the API process instead.
	if cfg.RunEmailConsumer {
		(&notifications.Consumer{
			RabbitMQURL:  cfg.RabbitMQURL,
			EmailFrom:    cfg.EmailFrom,
			AppURL:       cfg.AppURL,
			Provider:     cfg.EmailProvider,
			BrevoAPIKey:  cfg.BrevoAPIKey,
			SMTPHost:     cfg.SMTPHost,
			SMTPPort:     cfg.SMTPPort,
			SMTPUsername: cfg.SMTPUsername,
			SMTPPassword: cfg.SMTPPassword,
		}).Start(outboxCtx)
		log.Printf("learnquest: embedded email consumer enabled (provider=%s)", cfg.EmailProvider)
	}

	go func() {
		log.Printf("learnquest: listening on :%s (env=%s ai=%v cors=%v)", cfg.Port, cfg.AppEnv, aiSvc.Enabled(), cfg.CORSOrigins)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("learnquest: shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
	bus.Shutdown(shutdownCtx)
	stopOutbox()
	log.Println("learnquest: stopped.")
}

func ginMode(env string) string {
	if env == "production" {
		return gin.ReleaseMode
	}
	return gin.DebugMode
}

// healthcheck probes this process's own /health endpoint, which also verifies
// the database connection. Exits 0 when healthy, 1 otherwise.
func healthcheck() int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck: status %d\n", resp.StatusCode)
		return 1
	}
	return 0
}
