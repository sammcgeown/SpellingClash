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

	"spellingclash/internal/config"
	"spellingclash/internal/database"
	"spellingclash/internal/handlers"
	"spellingclash/internal/repository"
	"spellingclash/internal/service"
	"spellingclash/internal/utils"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database with config (supports sqlite, postgres, mysql)
	db, err := database.InitializeWithConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Printf("Database connection established (type: %s)", cfg.DatabaseType)

	// Run migrations
	if err := db.RunMigrations(cfg.MigrationsPath); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations completed successfully")

	// Seed bad words filter
	if err := db.SeedBadWords(); err != nil {
		log.Printf("Warning: Failed to seed bad words filter: %v", err)
	}

	// Load templates
	templates, err := loadTemplates(cfg.TemplatesPath)
	if err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	log.Println("Templates loaded successfully")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	familyRepo := repository.NewFamilyRepository(db)
	kidRepo := repository.NewKidRepository(db)
	listRepo := repository.NewListRepository(db)
	practiceRepo := repository.NewPracticeRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, familyRepo, cfg.SessionDuration)
	familyService := service.NewFamilyService(familyRepo, kidRepo)

	oauthProviders := map[string]handlers.OAuthProvider{
		"google": {
			Name:  "google",
			Label: "Google",
			Config: &oauth2.Config{
				ClientID:     cfg.GoogleClientID,
				ClientSecret: cfg.GoogleClientSecret,
				Endpoint:     google.Endpoint,
				Scopes:       []string{"openid", "email", "profile"},
			},
			UserInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
		},
		"facebook": {
			Name:  "facebook",
			Label: "Facebook",
			Config: &oauth2.Config{
				ClientID:     cfg.FacebookClientID,
				ClientSecret: cfg.FacebookClientSecret,
				Endpoint:     facebook.Endpoint,
				Scopes:       []string{"email", "public_profile"},
			},
			UserInfoURL: "https://graph.facebook.com/me?fields=id,name,email",
		},
		"apple": {
			Name:  "apple",
			Label: "Apple",
			Config: &oauth2.Config{
				ClientID:     cfg.AppleClientID,
				ClientSecret: cfg.AppleClientSecret,
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://appleid.apple.com/auth/authorize",
					TokenURL: "https://appleid.apple.com/auth/token",
				},
				Scopes: []string{"name", "email"},
			},
			AuthParams: map[string]string{
				"response_mode": "query",
			},
		},
	}
	
	// Initialize TTS service with audio directory
	ttsService := utils.NewTTSService(filepath.Join(cfg.StaticFilesPath, "audio"))
	listService := service.NewListService(listRepo, familyRepo, ttsService)
	practiceService := service.NewPracticeService(practiceRepo, listRepo)

	// Seed default public lists
	if err := listService.SeedDefaultPublicLists(); err != nil {
		log.Printf("Warning: Failed to seed default public lists: %v", err)
	}

	// Generate any missing audio files
	if err := listService.GenerateMissingAudio(); err != nil {
		log.Printf("Warning: Failed to generate missing audio files: %v", err)
	}

	// Clean up orphaned audio files
	if err := listService.CleanupOrphanedAudioFiles(); err != nil {
		log.Printf("Warning: Failed to cleanup orphaned audio files: %v", err)
	}

	// Initialize handlers
	middleware := handlers.NewMiddleware(authService, familyService)
	authHandler := handlers.NewAuthHandler(authService, templates, oauthProviders, cfg.OAuthRedirectBaseURL)
	parentHandler := handlers.NewParentHandler(familyService, listService, middleware, templates)
	kidHandler := handlers.NewKidHandler(familyService, listService, practiceService, middleware, templates)
	listHandler := handlers.NewListHandler(listService, familyService, middleware, templates)
	practiceHandler := handlers.NewPracticeHandler(practiceService, listService, templates)
	hangmanHandler := handlers.NewHangmanHandler(db, listService, templates)
	missingLetterHandler := handlers.NewMissingLetterHandler(db, listService, templates)
	adminHandler := handlers.NewAdminHandler(templates, authService, listService, listRepo, userRepo, familyRepo, kidRepo, middleware)

	// Setup routes
	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticFilesPath))))

	// Public routes
	mux.HandleFunc("GET /", authHandler.Home)
	mux.HandleFunc("GET /login", authHandler.ShowLogin)
	mux.HandleFunc("POST /login", middleware.RateLimit(authHandler.Login))
	mux.HandleFunc("GET /register", authHandler.ShowRegister)
	mux.HandleFunc("POST /register", middleware.RateLimit(authHandler.Register))
	mux.HandleFunc("POST /logout", authHandler.Logout)
	mux.HandleFunc("GET /auth/{provider}/start", authHandler.StartOAuth)
	mux.HandleFunc("GET /auth/{provider}/callback", authHandler.OAuthCallback)

	// Protected parent routes
	mux.HandleFunc("GET /parent/dashboard", middleware.RequireAuth(parentHandler.Dashboard))
	mux.HandleFunc("GET /parent/family", middleware.RequireAuth(parentHandler.ShowFamily))
	mux.HandleFunc("POST /parent/family/create", middleware.RequireAuth(middleware.CSRFProtect(parentHandler.CreateFamily)))
	mux.HandleFunc("POST /parent/family/join", middleware.RequireAuth(middleware.CSRFProtect(parentHandler.JoinFamily)))
	mux.HandleFunc("POST /parent/family/{familyCode}/leave", middleware.RequireAuth(middleware.CSRFProtect(parentHandler.LeaveFamily)))
	mux.HandleFunc("GET /parent/children", middleware.RequireAuth(parentHandler.ShowKids))
	mux.HandleFunc("POST /parent/children/create", middleware.RequireAuth(middleware.CSRFProtect(parentHandler.CreateKid)))
	mux.HandleFunc("POST /parent/children/{id}/update", middleware.RequireAuth(middleware.CSRFProtect(parentHandler.UpdateKid)))
	mux.HandleFunc("POST /parent/children/{id}/regenerate-password", middleware.RequireAuth(middleware.CSRFProtect(parentHandler.RegenerateKidPassword)))
	mux.HandleFunc("POST /parent/children/{id}/delete", middleware.RequireAuth(middleware.CSRFProtect(parentHandler.DeleteKid)))
	mux.HandleFunc("GET /parent/children/{id}", middleware.RequireAuth(kidHandler.GetKidDetails))
	mux.HandleFunc("GET /parent/children/{childId}/struggling-words", middleware.RequireAuth(kidHandler.GetKidStrugglingWords))

	// Spelling list routes
	mux.HandleFunc("GET /parent/lists", middleware.RequireAuth(listHandler.ShowLists))
	mux.HandleFunc("POST /parent/lists/create", middleware.RequireAuth(middleware.CSRFProtect(listHandler.CreateList)))
	mux.HandleFunc("GET /parent/lists/{id}", middleware.RequireAuth(listHandler.ViewList))
	mux.HandleFunc("PUT /parent/lists/{id}", middleware.RequireAuth(middleware.CSRFProtect(listHandler.UpdateList)))
	mux.HandleFunc("POST /parent/lists/{id}/delete", middleware.RequireAuth(middleware.CSRFProtect(listHandler.DeleteList)))
	mux.HandleFunc("POST /parent/lists/{id}/words/add", middleware.RequireAuth(middleware.CSRFProtect(listHandler.AddWord)))
	mux.HandleFunc("POST /parent/lists/{id}/words/bulk-add", middleware.RequireAuth(middleware.CSRFProtect(listHandler.BulkAddWords)))
	mux.HandleFunc("GET /parent/lists/{id}/words/bulk-add/progress", middleware.RequireAuth(listHandler.GetBulkImportProgress))
	mux.HandleFunc("POST /parent/lists/{listId}/words/{wordId}/update", middleware.RequireAuth(middleware.CSRFProtect(listHandler.UpdateWord)))
	mux.HandleFunc("POST /parent/lists/{listId}/words/{wordId}/delete", middleware.RequireAuth(middleware.CSRFProtect(listHandler.DeleteWord)))
	mux.HandleFunc("POST /parent/lists/{listId}/assign/{childId}", middleware.RequireAuth(middleware.CSRFProtect(listHandler.AssignList)))
	mux.HandleFunc("POST /parent/lists/{listId}/unassign/{childId}", middleware.RequireAuth(middleware.CSRFProtect(listHandler.UnassignList)))
	mux.HandleFunc("POST /parent/lists/assign-to-child", middleware.RequireAuth(middleware.CSRFProtect(listHandler.AssignListToKid)))

	// Child routes
	mux.HandleFunc("GET /child/select", kidHandler.ShowKidSelect)
	mux.HandleFunc("POST /child/login", kidHandler.KidLogin)
	mux.HandleFunc("GET /child/login/{id}", kidHandler.KidLogin)
	mux.HandleFunc("POST /child/login/{id}", kidHandler.KidLogin)
	mux.HandleFunc("GET /child/dashboard", middleware.RequireKidAuth(kidHandler.KidDashboard))
	mux.HandleFunc("POST /child/logout", kidHandler.KidLogout)

	// Practice routes
	mux.HandleFunc("POST /child/practice/start/{listId}", middleware.RequireKidAuth(practiceHandler.StartPractice))
	mux.HandleFunc("GET /child/practice", middleware.RequireKidAuth(practiceHandler.ShowPractice))
	mux.HandleFunc("POST /child/practice/submit", middleware.RequireKidAuth(practiceHandler.SubmitAnswer))
	mux.HandleFunc("POST /child/practice/exit", middleware.RequireKidAuth(practiceHandler.ExitPractice))
	mux.HandleFunc("GET /child/practice/results", middleware.RequireKidAuth(practiceHandler.ShowResults))

	// Hangman routes
	mux.HandleFunc("POST /child/hangman/start/{listId}", middleware.RequireKidAuth(hangmanHandler.StartHangman))
	mux.HandleFunc("GET /child/hangman/play", middleware.RequireKidAuth(hangmanHandler.PlayHangman))
	mux.HandleFunc("POST /child/hangman/guess", middleware.RequireKidAuth(hangmanHandler.GuessLetter))
	mux.HandleFunc("POST /child/hangman/next", middleware.RequireKidAuth(hangmanHandler.NextWord))
	mux.HandleFunc("POST /child/hangman/exit", middleware.RequireKidAuth(hangmanHandler.ExitGame))
	mux.HandleFunc("GET /child/hangman/results", middleware.RequireKidAuth(hangmanHandler.ShowResults))

	// Missing Letter Mayhem routes
	mux.HandleFunc("POST /child/missing-letter/start/{listId}", middleware.RequireKidAuth(missingLetterHandler.StartMissingLetter))
	mux.HandleFunc("GET /child/missing-letter/play", middleware.RequireKidAuth(missingLetterHandler.PlayMissingLetter))
	mux.HandleFunc("POST /child/missing-letter/guess", middleware.RequireKidAuth(missingLetterHandler.GuessLetter))
	mux.HandleFunc("POST /child/missing-letter/next", middleware.RequireKidAuth(missingLetterHandler.NextWord))
	mux.HandleFunc("POST /child/missing-letter/exit", middleware.RequireKidAuth(missingLetterHandler.ExitGame))
	mux.HandleFunc("GET /child/missing-letter/results", middleware.RequireKidAuth(missingLetterHandler.ShowResults))

	// 
	// Admin routes
	mux.HandleFunc("GET /admin/dashboard", middleware.RequireAdmin(adminHandler.ShowAdminDashboard))
	mux.HandleFunc("POST /admin/regenerate-lists", middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.RegeneratePublicLists)))
	mux.HandleFunc("GET /admin/parents", middleware.RequireAdmin(adminHandler.ShowManageParents))
	mux.HandleFunc("POST /admin/parents/create", middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.CreateParent)))
	mux.HandleFunc("POST /admin/parents/{id}/update", middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.UpdateParent)))
	mux.HandleFunc("POST /admin/parents/{id}/delete", middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.DeleteParent)))
	mux.HandleFunc("GET /admin/children", middleware.RequireAdmin(adminHandler.ShowManageKids))
	mux.HandleFunc("POST /admin/children/{id}/update", middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.UpdateKid)))
	mux.HandleFunc("POST /admin/children/{id}/delete", middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.DeleteKid)))

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
	go cleanupExpiredSessions(authService, familyService)

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
		filepath.Join(templatesPath, "admin/*.tmpl"),
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

	// Define template functions
	funcMap := template.FuncMap{
		"mult": func(a, b float64) float64 {
			return a * b
		},
		"formatDate": func(t time.Time) string {
			return t.Format("Jan 2, 2006")
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"list": func(items ...string) []string {
			return items
		},
		"until": func(count int) []int {
			result := make([]int, count)
			for i := 0; i < count; i++ {
				result[i] = i
			}
			return result
		},
		"index": func(s string, i int) byte {
			if i >= 0 && i < len(s) {
				return s[i]
			}
			return 0
		},
		"contains": func(slice []int, val int) bool {
			for _, item := range slice {
				if item == val {
					return true
				}
			}
			return false
		},
		"deref": func(b *bool) bool {
			if b == nil {
				return false
			}
			return *b
		},
	}

	// Parse all templates with functions
	tmpl, err := template.New("").Funcs(funcMap).ParseFiles(files...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return tmpl, nil
}

// cleanupExpiredSessions periodically removes expired sessions
func cleanupExpiredSessions(authService *service.AuthService, familyService *service.FamilyService) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// Cleanup parent sessions
		if err := authService.CleanupExpiredSessions(); err != nil {
			log.Printf("Error cleaning up expired sessions: %v", err)
		} else {
			log.Println("Expired parent sessions cleaned up")
		}

		// Cleanup kid sessions
		if err := familyService.CleanupExpiredKidSessions(); err != nil {
			log.Printf("Error cleaning up expired kid sessions: %v", err)
		} else {
			log.Println("Expired kid sessions cleaned up")
		}
	}
}
