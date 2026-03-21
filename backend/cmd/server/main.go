// cmd/server/main.go — full server bootstrap.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/cors"

	"comp-video-service/backend/internal/config"
	"comp-video-service/backend/internal/handler"
	"comp-video-service/backend/internal/middleware"
	"comp-video-service/backend/internal/repository"
	"comp-video-service/backend/internal/service"
	"comp-video-service/backend/internal/storage"
)

// @title           Video Comparison Service API
// @version         1.0
// @description     Controlled pairwise video comparison platform for flooding/explosion research.
// @host            localhost:8080
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @securityDefinitions.apikey CSRFToken
// @in              header
// @name            X-CSRF-Token
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("pgx pool: %v", err)
	}
	defer db.Close()

	if err := waitForDB(ctx, db); err != nil {
		log.Fatalf("database not ready: %v", err)
	}
	log.Println("database connected")
	if err := applyMigrations(cfg.MigrationsPath, cfg.DatabaseURL); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}
	log.Println("migrations applied")

	s3Client, err := storage.NewS3Client(ctx, cfg)
	if err != nil {
		log.Fatalf("s3 client: %v", err)
	}
	log.Println("s3 client ready")

	adminRepo := repository.NewAdminRepository(db)
	studyRepo := repository.NewStudyRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	sourceItemRepo := repository.NewSourceItemRepository(db)
	videoRepo := repository.NewVideoRepository(db)
	participantRepo := repository.NewParticipantRepository(db)
	pairRepo := repository.NewPairPresentationRepository(db)
	responseRepo := repository.NewResponseRepository(db)
	interactionRepo := repository.NewInteractionLogRepository(db)

	assignmentSvc := service.NewAssignmentService(sourceItemRepo, groupRepo, videoRepo, pairRepo)
	assetSvc := service.NewAssetService(videoRepo, s3Client)
	qcSvc := service.NewQCService(responseRepo, participantRepo)
	sessionSvc := service.NewSessionService(studyRepo, participantRepo, pairRepo, videoRepo, responseRepo, assignmentSvc, qcSvc, s3Client)
	studySvc := service.NewStudyService(studyRepo, groupRepo, sourceItemRepo, videoRepo)
	analyticsSvc := service.NewAnalyticsService(db, responseRepo)
	exportSvc := service.NewExportService(db)
	importSvc := service.NewImportService(studyRepo, groupRepo, sourceItemRepo, videoRepo, s3Client)

	authH := handler.NewAuthHandler(adminRepo, cfg.JWTSecret)
	sessionH := handler.NewSessionHandler(sessionSvc)
	taskH := handler.NewTaskHandler(sessionSvc, pairRepo, interactionRepo)
	adminH := handler.NewAdminHandlerWithImport(studySvc, assetSvc, analyticsSvc, qcSvc, exportSvc, importSvc)

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.NewIPRateLimiter(300, time.Minute))

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
	})
	r.Use(func(c *gin.Context) {
		corsHandler.HandlerFunc(c.Writer, c.Request)
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")

	api.POST("/session/start", sessionH.Start)
	api.GET("/session/:token/next-task", sessionH.NextTask)
	api.POST("/session/:token/complete", sessionH.Complete)
	api.POST("/task/:id/response", taskH.SubmitResponse)
	api.POST("/task/:id/event", taskH.LogEvent)

	api.POST("/admin/login", authH.Login)

	adminGroup := api.Group("/admin")
	adminGroup.Use(middleware.RequireAuth(cfg.JWTSecret))
	adminGroup.Use(middleware.RequireCSRF())
	{
		adminGroup.GET("/studies", adminH.ListStudies)
		adminGroup.POST("/studies", adminH.CreateStudy)
		adminGroup.PATCH("/studies/:id", adminH.UpdateStudy)
		adminGroup.DELETE("/studies/:id", adminH.DeleteStudy)
		adminGroup.POST("/studies/import-archive", adminH.ImportArchive)

		adminGroup.GET("/studies/:id/groups", adminH.ListGroups)
		adminGroup.POST("/studies/:id/groups", adminH.CreateGroup)
		adminGroup.POST("/assets/upload", adminH.UploadAsset)
		adminGroup.GET("/source-items", adminH.ListSourceItems)
		adminGroup.GET("/assets/free", adminH.ListFreeAssets)
		adminGroup.GET("/assets", adminH.ListAssets)
		adminGroup.POST("/studies/:id/pairs", adminH.CreatePair)
		adminGroup.DELETE("/source-items/:id", adminH.DeletePair)
		adminGroup.PATCH("/source-items/:id", adminH.UpdateSourceItem)
		adminGroup.DELETE("/assets/:id", adminH.DeleteAsset)
		adminGroup.GET("/assets/:id/url", adminH.GetAssetURL)

		adminGroup.GET("/analytics/overview", adminH.AnalyticsOverview)
		adminGroup.GET("/analytics/study/:id", adminH.AnalyticsStudy)
		adminGroup.GET("/analytics/study/:id/pairs", adminH.AnalyticsPairs)
		adminGroup.GET("/analytics/qc", adminH.AnalyticsQC)

		adminGroup.GET("/export/csv", adminH.ExportCSV)
		adminGroup.GET("/export/study/:id/csv", adminH.ExportStudyCSV)
	}

	log.Printf("starting server on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func waitForDB(ctx context.Context, db *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		if err := db.Ping(ctx); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func applyMigrations(migrationsPath, databaseURL string) error {
	// golang-migrate pgx/v5 driver registers under "pgx5" scheme
	migrateURL := strings.Replace(databaseURL, "postgres://", "pgx5://", 1)
	m, err := migrate.New(migrationsPath, migrateURL)
	if err != nil {
		return err
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Printf("migrate source close: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("migrate db close: %v", dbErr)
		}
	}()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
