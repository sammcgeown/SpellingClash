package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"spellingclash/internal/config"
	"spellingclash/internal/database"
	"spellingclash/internal/service"
)

func main() {
	// Define subcommands
	exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
	importCmd := flag.NewFlagSet("import", flag.ExitOnError)

	// Export flags
	exportOutput := exportCmd.String("output", "", "Output file path (default: backup_YYYYMMDD_HHMMSS.json)")

	// Import flags
	importInput := importCmd.String("input", "", "Input file path (required)")
	importClear := importCmd.Bool("clear", false, "Clear existing data before import (WARNING: destructive)")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.InitializeWithConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations to ensure schema is up to date
	if err := db.RunMigrations(cfg.MigrationsPath); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create backup service
	backupService := service.NewBackupService(db)

	switch os.Args[1] {
	case "export":
		exportCmd.Parse(os.Args[2:])
		handleExport(backupService, *exportOutput)

	case "import":
		importCmd.Parse(os.Args[2:])
		if *importInput == "" {
			fmt.Println("Error: -input flag is required")
			importCmd.PrintDefaults()
			os.Exit(1)
		}
		handleImport(backupService, db, *importInput, *importClear)

	default:
		printUsage()
		os.Exit(1)
	}
}

func handleExport(backupService *service.BackupService, outputPath string) {
	// Generate default filename if not provided
	if outputPath == "" {
		timestamp := time.Now().Format("20060102_150405")
		outputPath = fmt.Sprintf("backup_%s.json", timestamp)
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
	}

	log.Printf("Exporting database to: %s", outputPath)
	if err := backupService.Export(outputPath); err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	// Get file size
	fileInfo, _ := os.Stat(outputPath)
	log.Printf("Export complete! File size: %.2f MB", float64(fileInfo.Size())/1024/1024)
}

func handleImport(backupService *service.BackupService, db *database.DB, inputPath string, clearData bool) {
	// Check if file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		log.Fatalf("Input file does not exist: %s", inputPath)
	}

	if clearData {
		fmt.Print("WARNING: This will delete all existing data. Type 'yes' to confirm: ")
		var confirmation string
		fmt.Scanln(&confirmation)
		if confirmation != "yes" {
			log.Println("Import cancelled")
			return
		}

		log.Println("Clearing existing data...")
		if err := clearDatabase(db); err != nil {
			log.Fatalf("Failed to clear database: %v", err)
		}
	}

	log.Printf("Importing database from: %s", inputPath)
	if err := backupService.Import(inputPath); err != nil {
		log.Fatalf("Import failed: %v", err)
	}

	log.Println("Import complete!")
}

func clearDatabase(db *database.DB) error {
	// Delete in reverse order of dependencies
	tables := []string{
		"practice_results",
		"practice_sessions",
		"list_assignments",
		"words",
		"spelling_lists",
		"kid_sessions",
		"kids",
		"family_members",
		"families",
		"password_reset_tokens",
		"sessions",
		"users",
	}

	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s", table)
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}
		log.Printf("Cleared table: %s", table)
	}

	return nil
}

func printUsage() {
	fmt.Println("SpellingClash Database Backup Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  backup export [options]    Export database to JSON file")
	fmt.Println("  backup import [options]    Import database from JSON file")
	fmt.Println()
	fmt.Println("Export Options:")
	fmt.Println("  -output <file>    Output file path (default: backup_YYYYMMDD_HHMMSS.json)")
	fmt.Println()
	fmt.Println("Import Options:")
	fmt.Println("  -input <file>     Input file path (required)")
	fmt.Println("  -clear            Clear existing data before import (WARNING: destructive)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Export database")
	fmt.Println("  backup export")
	fmt.Println("  backup export -output mybackup.json")
	fmt.Println()
	fmt.Println("  # Import database (merge with existing data)")
	fmt.Println("  backup import -input backup.json")
	fmt.Println()
	fmt.Println("  # Import database (replace all data)")
	fmt.Println("  backup import -input backup.json -clear")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  DATABASE_TYPE    Database type: sqlite, postgres, or mysql (default: sqlite)")
	fmt.Println("  DB_PATH          SQLite database path (default: ./spellingclash.db)")
	fmt.Println("  DATABASE_URL     PostgreSQL or MySQL connection URL")
}
