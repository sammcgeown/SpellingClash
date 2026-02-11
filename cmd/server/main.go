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

	"spellingclash/internal/audio"
	"spellingclash/internal/config"
	"spellingclash/internal/database"
	"spellingclash/internal/handlers"
	"spellingclash/internal/repository"
	"spellingclash/internal/service"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
)

// Version can be set at build time using -ldflags "-X main.Version=x.y.z"
var Version = "dev"

func main() {
	// Load .env file if it exists (ignore error if not found)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()
	cfg.Version = Version

	// Start HTTP server early with startup status page
	addr := ":" + cfg.ServerPort
	mux := http.NewServeMux()

	// Startup status route (always available)
	mux.HandleFunc("/", handlers.ShowStartupStatus)
	mux.HandleFunc("/startup", handlers.ShowStartupStatus)

	server := &http.Server{
		Addr:         addr,
		Handler:      handlers.Logging(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		log.Printf("Server starting on http://localhost%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Variable to hold database connection (must be available after initialization completes)
	var db *database.DB

	// Initialize everything in background
	go func() {
		handlers.SetCurrentStep("Connecting to database...")

		// Initialize database with config (supports sqlite, postgres, mysql)
		var err error
		db, err = database.InitializeWithConfig(cfg)
		if err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}

		log.Printf("Database connection established (type: %s)", cfg.DatabaseType)
		handlers.CompleteStep("Database connection")

		handlers.SetCurrentStep("Running database migrations...")
		// Run migrations
		if err := db.RunMigrations(cfg.MigrationsPath); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}

		log.Println("Migrations completed successfully")
		handlers.CompleteStep("Running migrations")

		// Seed bad words filter
		if err := db.SeedBadWords(); err != nil {
			log.Printf("Warning: Failed to seed bad words filter: %v", err)
		}

		handlers.SetCurrentStep("Loading templates...")
		// Load templates
		templates, err := loadTemplates(cfg.TemplatesPath)
		if err != nil {
			log.Fatalf("Failed to load templates: %v", err)
		}

		log.Println("Templates loaded successfully")
		handlers.CompleteStep("Loading templates")

		handlers.SetCurrentStep("Initializing services...")
		// Initialize repositories
		userRepo := repository.NewUserRepository(db)
		familyRepo := repository.NewFamilyRepository(db)
		kidRepo := repository.NewKidRepository(db)
		listRepo := repository.NewListRepository(db)
		practiceRepo := repository.NewPracticeRepository(db)
		settingsRepo := repository.NewSettingsRepository(db)
		invitationRepo := repository.NewInvitationRepository(db)

		// Initialize services
		authService := service.NewAuthService(userRepo, familyRepo, cfg.SessionDuration)
		familyService := service.NewFamilyService(familyRepo, kidRepo)

		// Initialize email service (Amazon SES)
		emailService, err := service.NewEmailService(cfg.AWSRegion, cfg.SESFromEmail, cfg.SESFromName, cfg.AppBaseURL, cfg.DebugLogging)
		if err != nil {
			log.Printf("Warning: Email service initialization failed: %v", err)
			log.Println("Continuing without email notifications")
		}

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
		ttsService := audio.NewTTSService(filepath.Join(cfg.StaticFilesPath, "audio"))
		listService := service.NewListService(listRepo, familyRepo, ttsService)
		practiceService := service.NewPracticeService(practiceRepo, listRepo)

		handlers.CompleteStep("Initializing services")

		handlers.SetCurrentStep("Seeding default lists...")
		// Seed default public lists
		if err := listService.SeedDefaultPublicLists(); err != nil {
			log.Printf("Warning: Failed to seed default public lists: %v", err)
		}
		handlers.CompleteStep("Seeding default lists")

		handlers.SetCurrentStep("Generating audio files (this may take a while)...")
		// Generate any missing audio files
		if err := listService.GenerateMissingAudio(); err != nil {
			log.Printf("Warning: Failed to generate missing audio files: %v", err)
		}

		// Clean up orphaned audio files
		if err := listService.CleanupOrphanedAudioFiles(); err != nil {
			log.Printf("Warning: Failed to cleanup orphaned audio files: %v", err)
		}
		handlers.CompleteStep("Generating audio files")

		handlers.SetCurrentStep("Setting up routes...")
		// Initialize handlers
		middleware := handlers.NewMiddleware(authService, familyService)
		backupService := service.NewBackupService(db)
		authHandler := handlers.NewAuthHandler(authService, emailService, templates, oauthProviders, cfg.OAuthRedirectBaseURL, settingsRepo, invitationRepo)
		parentHandler := handlers.NewParentHandler(familyService, listService, middleware, templates)
		kidHandler := handlers.NewKidHandler(familyService, listService, practiceService, middleware, templates)
		listHandler := handlers.NewListHandler(listService, familyService, middleware, templates)
		practiceHandler := handlers.NewPracticeHandler(practiceService, listService, templates)
		hangmanHandler := handlers.NewHangmanHandler(db, listService, templates)
		missingLetterHandler := handlers.NewMissingLetterHandler(db, listService, templates)
		adminHandler := handlers.NewAdminHandler(templates, authService, emailService, listService, backupService, listRepo, userRepo, familyRepo, kidRepo, settingsRepo, invitationRepo, middleware, cfg.Version, cfg.AppBaseURL)

		// Setup new routes
		newMux := http.NewServeMux()

		// Static files
		newMux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticFilesPath))))

		// Public routes
		newMux.HandleFunc("GET /", handlers.RequireReady(authHandler.Home))
		newMux.HandleFunc("GET /login", handlers.RequireReady(authHandler.ShowLogin))
		newMux.HandleFunc("POST /login", handlers.RequireReady(middleware.RateLimit(authHandler.Login)))
		newMux.HandleFunc("GET /register", handlers.RequireReady(authHandler.ShowRegister))
		newMux.HandleFunc("POST /register", handlers.RequireReady(middleware.RateLimit(authHandler.Register)))
		newMux.HandleFunc("POST /logout", handlers.RequireReady(authHandler.Logout))
		newMux.HandleFunc("GET /auth/{provider}/start", handlers.RequireReady(authHandler.StartOAuth))
		newMux.HandleFunc("GET /auth/{provider}/callback", handlers.RequireReady(authHandler.OAuthCallback))
		newMux.HandleFunc("GET /auth/forgot-password", handlers.RequireReady(authHandler.ShowForgotPassword))
		newMux.HandleFunc("POST /auth/forgot-password", handlers.RequireReady(middleware.RateLimit(authHandler.ForgotPassword)))
		newMux.HandleFunc("GET /auth/reset-password", handlers.RequireReady(authHandler.ShowResetPassword))
		newMux.HandleFunc("POST /auth/reset-password", handlers.RequireReady(middleware.RateLimit(authHandler.ResetPassword)))

		// Protected parent routes
		newMux.HandleFunc("GET /parent/dashboard", handlers.RequireReady(middleware.RequireAuth(parentHandler.Dashboard)))
		newMux.HandleFunc("GET /parent/family", handlers.RequireReady(middleware.RequireAuth(parentHandler.ShowFamily)))
		newMux.HandleFunc("POST /parent/family/create", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(parentHandler.CreateFamily))))
		newMux.HandleFunc("POST /parent/family/join", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(parentHandler.JoinFamily))))
		newMux.HandleFunc("POST /parent/family/{familyCode}/leave", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(parentHandler.LeaveFamily))))
		newMux.HandleFunc("GET /parent/children", handlers.RequireReady(middleware.RequireAuth(parentHandler.ShowKids)))
		newMux.HandleFunc("POST /parent/children/create", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(parentHandler.CreateKid))))
		newMux.HandleFunc("POST /parent/children/{id}/update", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(parentHandler.UpdateKid))))
		newMux.HandleFunc("POST /parent/children/{id}/regenerate-password", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(parentHandler.RegenerateKidPassword))))
		newMux.HandleFunc("POST /parent/children/{id}/delete", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(parentHandler.DeleteKid))))
		newMux.HandleFunc("GET /parent/children/{id}", handlers.RequireReady(middleware.RequireAuth(kidHandler.GetKidDetails)))
		newMux.HandleFunc("GET /parent/children/{childId}/struggling-words", handlers.RequireReady(middleware.RequireAuth(kidHandler.GetKidStrugglingWords)))

		// Spelling list routes
		newMux.HandleFunc("GET /parent/lists", handlers.RequireReady(middleware.RequireAuth(listHandler.ShowLists)))
		newMux.HandleFunc("POST /parent/lists/create", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(listHandler.CreateList))))
		newMux.HandleFunc("GET /parent/lists/{id}", handlers.RequireReady(middleware.RequireAuth(listHandler.ViewList)))
		newMux.HandleFunc("PUT /parent/lists/{id}", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(listHandler.UpdateList))))
		newMux.HandleFunc("POST /parent/lists/{id}/delete", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(listHandler.DeleteList))))
		newMux.HandleFunc("POST /parent/lists/{id}/words/add", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(listHandler.AddWord))))
		newMux.HandleFunc("POST /parent/lists/{id}/words/bulk-add", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(listHandler.BulkAddWords))))
		newMux.HandleFunc("GET /parent/lists/{id}/words/bulk-add/progress", handlers.RequireReady(middleware.RequireAuth(listHandler.GetBulkImportProgress)))
		newMux.HandleFunc("POST /parent/lists/{listId}/words/{wordId}/update", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(listHandler.UpdateWord))))
		newMux.HandleFunc("POST /parent/lists/{listId}/words/{wordId}/delete", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(listHandler.DeleteWord))))
		newMux.HandleFunc("POST /parent/lists/{listId}/assign/{childId}", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(listHandler.AssignList))))
		newMux.HandleFunc("POST /parent/lists/{listId}/unassign/{childId}", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(listHandler.UnassignList))))
		newMux.HandleFunc("POST /parent/lists/assign-to-child", handlers.RequireReady(middleware.RequireAuth(middleware.CSRFProtect(listHandler.AssignListToKid))))

		// Child routes
		newMux.HandleFunc("GET /child/select", handlers.RequireReady(kidHandler.ShowKidSelect))
		newMux.HandleFunc("POST /child/login", handlers.RequireReady(kidHandler.KidLogin))
		newMux.HandleFunc("GET /child/login/{id}", handlers.RequireReady(kidHandler.KidLogin))
		newMux.HandleFunc("POST /child/login/{id}", handlers.RequireReady(kidHandler.KidLogin))
		newMux.HandleFunc("GET /child/dashboard", handlers.RequireReady(middleware.RequireKidAuth(kidHandler.KidDashboard)))
		newMux.HandleFunc("POST /child/logout", handlers.RequireReady(kidHandler.KidLogout))

		// Practice routes
		newMux.HandleFunc("POST /child/practice/start/{listId}", handlers.RequireReady(middleware.RequireKidAuth(practiceHandler.StartPractice)))
		newMux.HandleFunc("GET /child/practice", handlers.RequireReady(middleware.RequireKidAuth(practiceHandler.ShowPractice)))
		newMux.HandleFunc("POST /child/practice/submit", handlers.RequireReady(middleware.RequireKidAuth(practiceHandler.SubmitAnswer)))
		newMux.HandleFunc("POST /child/practice/exit", handlers.RequireReady(middleware.RequireKidAuth(practiceHandler.ExitPractice)))
		newMux.HandleFunc("GET /child/practice/results", handlers.RequireReady(middleware.RequireKidAuth(practiceHandler.ShowResults)))

		// Hangman routes
		newMux.HandleFunc("POST /child/hangman/start/{listId}", handlers.RequireReady(middleware.RequireKidAuth(hangmanHandler.StartHangman)))
		newMux.HandleFunc("GET /child/hangman/play", handlers.RequireReady(middleware.RequireKidAuth(hangmanHandler.PlayHangman)))
		newMux.HandleFunc("POST /child/hangman/guess", handlers.RequireReady(middleware.RequireKidAuth(hangmanHandler.GuessLetter)))
		newMux.HandleFunc("POST /child/hangman/next", handlers.RequireReady(middleware.RequireKidAuth(hangmanHandler.NextWord)))
		newMux.HandleFunc("POST /child/hangman/exit", handlers.RequireReady(middleware.RequireKidAuth(hangmanHandler.ExitGame)))
		newMux.HandleFunc("GET /child/hangman/results", handlers.RequireReady(middleware.RequireKidAuth(hangmanHandler.ShowResults)))

		// Missing Letter Mayhem routes
		newMux.HandleFunc("POST /child/missing-letter/start/{listId}", handlers.RequireReady(middleware.RequireKidAuth(missingLetterHandler.StartMissingLetter)))
		newMux.HandleFunc("GET /child/missing-letter/play", handlers.RequireReady(middleware.RequireKidAuth(missingLetterHandler.PlayMissingLetter)))
		newMux.HandleFunc("POST /child/missing-letter/guess", handlers.RequireReady(middleware.RequireKidAuth(missingLetterHandler.GuessLetter)))
		newMux.HandleFunc("POST /child/missing-letter/next", handlers.RequireReady(middleware.RequireKidAuth(missingLetterHandler.NextWord)))
		newMux.HandleFunc("POST /child/missing-letter/exit", handlers.RequireReady(middleware.RequireKidAuth(missingLetterHandler.ExitGame)))
		newMux.HandleFunc("GET /child/missing-letter/results", handlers.RequireReady(middleware.RequireKidAuth(missingLetterHandler.ShowResults)))

		// Admin routes
		newMux.HandleFunc("GET /admin/dashboard", handlers.RequireReady(middleware.RequireAdmin(adminHandler.ShowAdminDashboard)))
		newMux.HandleFunc("POST /admin/regenerate-lists", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.RegeneratePublicLists))))
		newMux.HandleFunc("GET /admin/parents", handlers.RequireReady(middleware.RequireAdmin(adminHandler.ShowManageParents)))
		newMux.HandleFunc("POST /admin/parents/create", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.CreateParent))))
		newMux.HandleFunc("POST /admin/parents/{id}/update", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.UpdateParent))))
		newMux.HandleFunc("POST /admin/parents/{id}/delete", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.DeleteParent))))
		newMux.HandleFunc("GET /admin/children", handlers.RequireReady(middleware.RequireAdmin(adminHandler.ShowManageKids)))
		newMux.HandleFunc("POST /admin/children/{id}/update", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.UpdateKid))))
		newMux.HandleFunc("POST /admin/children/{id}/delete", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.DeleteKid))))
		newMux.HandleFunc("GET /admin/database", handlers.RequireReady(middleware.RequireAdmin(adminHandler.ShowDatabaseManagement)))
		newMux.HandleFunc("GET /admin/export", handlers.RequireReady(middleware.RequireAdmin(adminHandler.ExportDatabase)))
		newMux.HandleFunc("POST /admin/import", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.ImportDatabase))))
		newMux.HandleFunc("GET /admin/invitations", handlers.RequireReady(middleware.RequireAdmin(adminHandler.ShowInvitations)))
		newMux.HandleFunc("POST /admin/invitations/toggle", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.ToggleInviteOnlyMode))))
		newMux.HandleFunc("POST /admin/invitations/send", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.SendInvitation))))
		newMux.HandleFunc("POST /admin/invitations/delete-used", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.DeleteUsedInvitations))))
		newMux.HandleFunc("POST /admin/invitations/delete-expired", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.DeleteExpiredInvitations))))
		newMux.HandleFunc("POST /admin/invitations/{id}/resend", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.ResendInvitation))))
		newMux.HandleFunc("POST /admin/invitations/{id}", handlers.RequireReady(middleware.RequireAdmin(middleware.CSRFProtect(adminHandler.DeleteInvitation))))

		// Replace the handler with the new one
		server.Handler = handlers.Logging(newMux)

		// Start background session cleanup
		go cleanupExpiredSessions(authService, familyService)

		// Mark as ready
		handlers.MarkReady()
		handlers.CompleteStep("Server ready")
		log.Println("Server initialization complete - ready to serve requests")
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Close database connection if it was initialized
	if db != nil {
		db.Close()
		log.Println("Database connection closed")
	}
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

		// Cleanup password reset tokens
		if err := authService.CleanupExpiredPasswordResetTokens(); err != nil {
			log.Printf("Error cleaning up expired password reset tokens: %v", err)
		} else {
			log.Println("Expired password reset tokens cleaned up")
		}
	}
}
