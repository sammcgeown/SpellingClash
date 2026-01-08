package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"wordclash/internal/config"
	"wordclash/internal/database"
	"wordclash/internal/handlers"
	"wordclash/internal/repository"
	"wordclash/internal/service"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Database connection established")

	// Run migrations
	if err := db.RunMigrations(cfg.MigrationsPath); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations completed successfully")

	// Load templates
	templates, err := loadTemplates(cfg.TemplatesPath)
	if err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	log.Println("Templates loaded successfully")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.DB)
	familyRepo := repository.NewFamilyRepository(db.DB)
	kidRepo := repository.NewKidRepository(db.DB)
	listRepo := repository.NewListRepository(db.DB)

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.SessionDuration)
	familyService := service.NewFamilyService(familyRepo, kidRepo)
	listService := service.NewListService(listRepo, familyRepo)

	// Initialize handlers
	middleware := handlers.NewMiddleware(authService, familyService)
	authHandler := handlers.NewAuthHandler(authService, templates)
	parentHandler := handlers.NewParentHandler(familyService, templates)
	kidHandler := handlers.NewKidHandler(familyService, templates)
	listHandler := handlers.NewListHandler(listService, familyService, templates)

	// Setup routes
	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticFilesPath))))

	// Public routes
	mux.HandleFunc("GET /", authHandler.Home)
	mux.HandleFunc("GET /login", authHandler.ShowLogin)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("GET /register", authHandler.ShowRegister)
	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /logout", authHandler.Logout)

	// Protected parent routes
	mux.HandleFunc("GET /parent/dashboard", middleware.RequireAuth(parentHandler.Dashboard))
	mux.HandleFunc("GET /parent/family", middleware.RequireAuth(parentHandler.ShowFamily))
	mux.HandleFunc("POST /parent/family/create", middleware.RequireAuth(parentHandler.CreateFamily))
	mux.HandleFunc("GET /parent/kids", middleware.RequireAuth(parentHandler.ShowKids))
	mux.HandleFunc("POST /parent/kids/create", middleware.RequireAuth(parentHandler.CreateKid))
	mux.HandleFunc("PUT /parent/kids/{id}", middleware.RequireAuth(parentHandler.UpdateKid))
	mux.HandleFunc("POST /parent/kids/{id}/delete", middleware.RequireAuth(parentHandler.DeleteKid))

	// Spelling list routes
	mux.HandleFunc("GET /parent/lists", middleware.RequireAuth(listHandler.ShowLists))
	mux.HandleFunc("POST /parent/lists/create", middleware.RequireAuth(listHandler.CreateList))
	mux.HandleFunc("GET /parent/lists/{id}", middleware.RequireAuth(listHandler.ViewList))
	mux.HandleFunc("PUT /parent/lists/{id}", middleware.RequireAuth(listHandler.UpdateList))
	mux.HandleFunc("POST /parent/lists/{id}/delete", middleware.RequireAuth(listHandler.DeleteList))
	mux.HandleFunc("POST /parent/lists/{id}/words/add", middleware.RequireAuth(listHandler.AddWord))
	mux.HandleFunc("POST /parent/lists/{listId}/words/{wordId}/delete", middleware.RequireAuth(listHandler.DeleteWord))
	mux.HandleFunc("POST /parent/lists/{listId}/assign/{kidId}", middleware.RequireAuth(listHandler.AssignList))
	mux.HandleFunc("POST /parent/lists/{listId}/unassign/{kidId}", middleware.RequireAuth(listHandler.UnassignList))

	// Kid routes
	mux.HandleFunc("GET /kid/select", kidHandler.ShowKidSelect)
	mux.HandleFunc("POST /kid/login/{id}", kidHandler.KidLogin)
	mux.HandleFunc("GET /kid/dashboard", middleware.RequireKidAuth(kidHandler.KidDashboard))
	mux.HandleFunc("POST /kid/logout", kidHandler.KidLogout)

	// Wrap with logging middleware
	handler := handlers.Logging(mux)

	// Start server
	addr := ":" + cfg.ServerPort
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start background session cleanup
	go cleanupExpiredSessions(authService)

	// Graceful shutdown
	go func() {
		log.Printf("Server starting on http://localhost%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")
}

// loadTemplates loads all template files
func loadTemplates(templatesPath string) (*template.Template, error) {
	// Get all template files
	baseTemplate := filepath.Join(templatesPath, "base.tmpl")

	// Load all template files
	patterns := []string{
		filepath.Join(templatesPath, "auth/*.tmpl"),
		filepath.Join(templatesPath, "parent/*.tmpl"),
		filepath.Join(templatesPath, "kid/*.tmpl"),
		filepath.Join(templatesPath, "components/*.tmpl"),
	}

	var files []string
	files = append(files, baseTemplate)

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob pattern %s: %w", pattern, err)
		}
		files = append(files, matches...)
	}

	// Parse all templates
	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return tmpl, nil
}

// cleanupExpiredSessions periodically removes expired sessions
func cleanupExpiredSessions(authService *service.AuthService) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if err := authService.CleanupExpiredSessions(); err != nil {
			log.Printf("Error cleaning up expired sessions: %v", err)
		} else {
			log.Println("Expired sessions cleaned up")
		}
	}
}
