package database

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const badWordsURL = "https://raw.githubusercontent.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words/refs/heads/master/en"

// SeedBadWords fetches and seeds the bad words list from GitHub
func (db *DB) SeedBadWords() error {
	// Check if bad words already exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM bad_words").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check bad words count: %w", err)
	}

	if count > 0 {
		log.Printf("Bad words filter already populated with %d words", count)
		return nil
	}

	log.Println("Downloading bad words list...")

	// Fetch the bad words list
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(badWordsURL)
	if err != nil {
		return fmt.Errorf("failed to download bad words list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code from bad words URL: %d", resp.StatusCode)
	}

	// Read and insert words
	scanner := bufio.NewScanner(resp.Body)
	wordsAdded := 0

	// Start transaction for bulk insert
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare insert statement (use dialect-aware query rewriting)
	insertQuery := "INSERT INTO bad_words (word) VALUES (?)"
	rewrittenQuery := db.Dialect.RewriteQuery(insertQuery)

	stmt, err := tx.Prepare(rewrittenQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for scanner.Scan() {
		word := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if word == "" {
			continue
		}

		_, err := stmt.Exec(word)
		if err != nil {
			// Skip duplicates or errors, continue adding others
			continue
		}
		wordsAdded++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading bad words: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Bad words filter populated with %d words", wordsAdded)
	return nil
}

// IsBadWord checks if a word is in the bad words list
func (db *DB) IsBadWord(word string) (bool, error) {
	cleanWord := strings.TrimSpace(strings.ToLower(word))

	var count int
	query := "SELECT COUNT(*) FROM bad_words WHERE word = ?"
	err := db.QueryRow(query, cleanWord).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check bad word: %w", err)
	}

	if count > 0 {
		log.Printf("Bad word detected: '%s'", word)
	}

	return count > 0, nil
}

// ValidateWords checks a list of words against the bad words filter
// Returns the list of bad words found
func (db *DB) ValidateWords(words []string) ([]string, error) {
	if len(words) == 0 {
		return nil, nil
	}

	var badWords []string
	for _, word := range words {
		isBad, err := db.IsBadWord(word)
		if err != nil {
			return nil, err
		}
		if isBad {
			badWords = append(badWords, word)
		}
	}

	return badWords, nil
}
