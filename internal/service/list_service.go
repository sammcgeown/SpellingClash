package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"spellingclash/internal/models"
	"spellingclash/internal/repository"
	"spellingclash/internal/utils"
	"strings"
)

var (
	ErrListNotFound = errors.New("list not found")
	ErrWordNotFound = errors.New("word not found")
)

// WordListData represents the structure of word list JSON files
type WordListData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Difficulty  int    `json:"difficulty"`
	Words       []struct {
		Word       string `json:"word"`
		Definition string `json:"definition"`
	} `json:"words"`
}

// ListService handles spelling list business logic
type ListService struct {
	listRepo   *repository.ListRepository
	familyRepo *repository.FamilyRepository
	ttsService *utils.TTSService
	dataPath   string
}

// NewListService creates a new list service
func NewListService(listRepo *repository.ListRepository, familyRepo *repository.FamilyRepository, ttsService *utils.TTSService) *ListService {
	return &ListService{
		listRepo:   listRepo,
		familyRepo: familyRepo,
		ttsService: ttsService,
		dataPath:   "data", // Default data path
	}
}

// SetDataPath sets the path to the data directory
func (s *ListService) SetDataPath(path string) {
	s.dataPath = path
}

// hasAccessToList checks if a user can access a list (either it's public or they're in the family)
func (s *ListService) hasAccessToList(userID int64, list *models.SpellingList) (bool, error) {
	// Public lists are accessible to everyone
	if list.IsPublic {
		return true, nil
	}

	// For private lists, check family membership
	if list.FamilyID == nil {
		return false, nil
	}

	isMember, err := s.familyRepo.IsFamilyMember(userID, *list.FamilyID)
	if err != nil {
		return false, fmt.Errorf("failed to verify family access: %w", err)
	}

	return isMember, nil
}

// SeedDefaultPublicLists creates default public lists if they don't exist
func (s *ListService) SeedDefaultPublicLists() error {
	// List of JSON files to seed
	listFiles := []string{
		"year_1_2_words.json",
		"year_3_4_words.json",
		"year_5_6_words.json",
	}

	for _, filename := range listFiles {
		if err := s.seedListFromFile(filename); err != nil {
			return fmt.Errorf("failed to seed list from %s: %w", filename, err)
		}
	}

	// Seed Year 8 words from multiple parts
	year8Parts := []string{
		"year_8_words_part1.json",
		"year_8_words_part2.json",
		"year_8_words_part3.json",
		"year_8_words_part4.json",
		"year_8_words_part5.json",
		"year_8_words_part6.json",
	}

	if err := s.seedCombinedList("Year 8 Words", "Year 8 spelling words for KS3 students", 4, year8Parts); err != nil {
		return fmt.Errorf("failed to seed Year 8 words: %w", err)
	}

	return nil
}

// seedListFromFile loads a word list from a JSON file and seeds it
func (s *ListService) seedListFromFile(filename string) error {
	// Read the JSON file
	filePath := filepath.Join(s.dataPath, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse JSON
	var listData WordListData
	if err := json.Unmarshal(data, &listData); err != nil {
		return fmt.Errorf("failed to parse JSON from %s: %w", filename, err)
	}

	// Check if list already exists
	exists, err := s.listRepo.PublicListExists(listData.Name)
	if err != nil {
		return fmt.Errorf("failed to check if %s list exists: %w", listData.Name, err)
	}

	if exists {
		log.Printf("Default public list '%s' already exists, skipping seed", listData.Name)
		return nil
	}

	log.Printf("Creating default public list '%s'...", listData.Name)

	// Create the public list
	list, err := s.listRepo.CreatePublicList(listData.Name, listData.Description)
	if err != nil {
		return fmt.Errorf("failed to create %s public list: %w", listData.Name, err)
	}

	log.Printf("Adding %d words to %s list...", len(listData.Words), listData.Name)

	// Add each word with definition and audio generation
	for i, wordData := range listData.Words {
		word, err := s.listRepo.AddWord(list.ID, wordData.Word, listData.Difficulty, i+1, wordData.Definition)
		if err != nil {
			log.Printf("Warning: Failed to add word '%s': %v", wordData.Word, err)
			continue
		}

		// Generate audio file for the word
		if s.ttsService != nil {
			audioFilename, err := s.ttsService.GenerateAudioFile(wordData.Word)
			if err != nil {
				log.Printf("Warning: Failed to generate audio for '%s': %v", wordData.Word, err)
			} else {
				if err := s.listRepo.UpdateWordAudio(word.ID, audioFilename); err != nil {
					log.Printf("Warning: Failed to update audio filename for word %d: %v", word.ID, err)
				} else {
					log.Printf("Generated audio for '%s': %s", wordData.Word, audioFilename)
				}
			}

			// Generate audio for definition if provided
			if wordData.Definition != "" {
				definitionPrefix := fmt.Sprintf("definition_%s", wordData.Word)
				definitionAudioFilename, err := s.ttsService.GenerateAudioFileWithPrefix(wordData.Definition, definitionPrefix)
				if err != nil {
					log.Printf("Warning: Failed to generate definition audio for '%s': %v", wordData.Word, err)
				} else {
					if err := s.listRepo.UpdateWordDefinitionAudio(word.ID, definitionAudioFilename); err != nil {
						log.Printf("Warning: Failed to update definition audio filename for word %d: %v", word.ID, err)
					} else {
						log.Printf("Generated definition audio for '%s': %s", wordData.Word, definitionAudioFilename)
					}
				}
			}
		}
	}

	log.Printf("Successfully created default public list '%s' with %d words", listData.Name, len(listData.Words))
	return nil
}

// seedCombinedList loads multiple JSON part files and combines them into a single list
func (s *ListService) seedCombinedList(listName, description string, difficulty int, partFiles []string) error {
	// Check if list already exists
	exists, err := s.listRepo.PublicListExists(listName)
	if err != nil {
		return fmt.Errorf("failed to check if %s list exists: %w", listName, err)
	}

	if exists {
		log.Printf("Default public list '%s' already exists, skipping seed", listName)
		return nil
	}

	log.Printf("Creating default public list '%s'...", listName)

	// Create the public list
	list, err := s.listRepo.CreatePublicList(listName, description)
	if err != nil {
		return fmt.Errorf("failed to create %s public list: %w", listName, err)
	}

	// Combine words from all part files
	type WordData struct {
		Word       string `json:"word"`
		Definition string `json:"definition"`
		Difficulty int    `json:"difficulty"`
	}

	allWords := []WordData{}
	for _, filename := range partFiles {
		filePath := filepath.Join(s.dataPath, filename)
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		var words []WordData
		if err := json.Unmarshal(data, &words); err != nil {
			return fmt.Errorf("failed to parse JSON from %s: %w", filename, err)
		}

		allWords = append(allWords, words...)
	}

	log.Printf("Adding %d words to %s list...", len(allWords), listName)

	// Add each word with definition and audio generation
	for i, wordData := range allWords {
		// Use the difficulty from the word data if it exists, otherwise use the default
		wordDifficulty := difficulty
		if wordData.Difficulty > 0 {
			wordDifficulty = wordData.Difficulty
		}

		word, err := s.listRepo.AddWord(list.ID, wordData.Word, wordDifficulty, i+1, wordData.Definition)
		if err != nil {
			log.Printf("Warning: Failed to add word '%s': %v", wordData.Word, err)
			continue
		}

		// Generate audio file for the word
		if s.ttsService != nil {
			audioFilename, err := s.ttsService.GenerateAudioFile(wordData.Word)
			if err != nil {
				log.Printf("Warning: Failed to generate audio for '%s': %v", wordData.Word, err)
			} else {
				if err := s.listRepo.UpdateWordAudio(word.ID, audioFilename); err != nil {
					log.Printf("Warning: Failed to update audio filename for word %d: %v", word.ID, err)
				} else {
					log.Printf("Generated audio for '%s': %s", wordData.Word, audioFilename)
				}
			}

			// Generate audio for definition if provided
			if wordData.Definition != "" {
				definitionPrefix := fmt.Sprintf("definition_%s", wordData.Word)
				definitionAudioFilename, err := s.ttsService.GenerateAudioFileWithPrefix(wordData.Definition, definitionPrefix)
				if err != nil {
					log.Printf("Warning: Failed to generate definition audio for '%s': %v", wordData.Word, err)
				} else {
					if err := s.listRepo.UpdateWordDefinitionAudio(word.ID, definitionAudioFilename); err != nil {
						log.Printf("Warning: Failed to update definition audio filename for word %d: %v", word.ID, err)
					} else {
						log.Printf("Generated definition audio for '%s': %s", wordData.Word, definitionAudioFilename)
					}
				}
			}
		}
	}

	log.Printf("Successfully created default public list '%s' with %d words", listName, len(allWords))
	return nil
}

// CreateList creates a new spelling list
func (s *ListService) CreateList(familyID, userID int64, name, description string) (*models.SpellingList, error) {
	// Verify user has access to family
	isMember, err := s.familyRepo.IsFamilyMember(userID, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return nil, ErrNotFamilyMember
	}

	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("list name is required")
	}

	// Create list
	list, err := s.listRepo.CreateList(familyID, name, description, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create list: %w", err)
	}

	return list, nil
}

// GetList retrieves a spelling list by ID
func (s *ListService) GetList(listID int64) (*models.SpellingList, error) {
	list, err := s.listRepo.GetListByID(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}
	if list == nil {
		return nil, ErrListNotFound
	}
	return list, nil
}

// GetFamilyLists retrieves all lists for a family
func (s *ListService) GetFamilyLists(familyID, userID int64) ([]models.SpellingList, error) {
	// Verify user has access to family
	isMember, err := s.familyRepo.IsFamilyMember(userID, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return nil, ErrNotFamilyMember
	}

	lists, err := s.listRepo.GetFamilyLists(familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get family lists: %w", err)
	}

	return lists, nil
}

// GetAllUserLists retrieves all lists from all families a user has access to, plus public lists
func (s *ListService) GetAllUserLists(userID int64) ([]models.SpellingList, error) {
	// Get user's families
	families, err := s.familyRepo.GetUserFamilies(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user families: %w", err)
	}

	// Collect all lists from all families
	var allLists []models.SpellingList
	for _, family := range families {
		lists, err := s.listRepo.GetFamilyLists(family.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get lists for family %d: %w", family.ID, err)
		}
		allLists = append(allLists, lists...)
	}

	// Add public lists
	publicLists, err := s.listRepo.GetPublicLists()
	if err != nil {
		return nil, fmt.Errorf("failed to get public lists: %w", err)
	}
	allLists = append(allLists, publicLists...)

	return allLists, nil
}

// GetAllUserListsWithAssignments retrieves all lists with assignment counts, including public lists
func (s *ListService) GetAllUserListsWithAssignments(userID int64) ([]models.ListSummary, error) {
	// Get user's families
	families, err := s.familyRepo.GetUserFamilies(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user families: %w", err)
	}

	// Collect all lists from all families with assignment counts
	var allLists []models.ListSummary
	for _, family := range families {
		lists, err := s.listRepo.GetFamilyListsWithAssignmentCounts(family.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get lists for family %d: %w", family.ID, err)
		}
		allLists = append(allLists, lists...)
	}

	// Add public lists with assignment counts
	publicLists, err := s.listRepo.GetPublicLists()
	if err != nil {
		return nil, fmt.Errorf("failed to get public lists: %w", err)
	}
	for _, publicList := range publicLists {
		// Get word count for public list
		wordCount, err := s.listRepo.GetWordCount(publicList.ID)
		if err != nil {
			log.Printf("Warning: Failed to get word count for public list %d: %v", publicList.ID, err)
			wordCount = 0
		}

		allLists = append(allLists, models.ListSummary{
			SpellingList:     publicList,
			AssignedKidCount: 0, // We don't count cross-family assignments for public lists in this view
			WordCount:        wordCount,
		})
	}

	return allLists, nil
}

// UpdateList updates a list's name and description
func (s *ListService) UpdateList(listID, userID int64, name, description string) error {
	// Get list to verify family access
	list, err := s.GetList(listID)
	if err != nil {
		return err
	}

	// Public lists cannot be modified
	if list.IsPublic {
		return errors.New("cannot modify public lists")
	}

	// Verify user has access to the list's family
	if list.FamilyID == nil {
		return ErrNotFamilyMember
	}
	isMember, err := s.familyRepo.IsFamilyMember(userID, *list.FamilyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("list name is required")
	}

	// Update list
	if err := s.listRepo.UpdateList(listID, name, description); err != nil {
		return fmt.Errorf("failed to update list: %w", err)
	}

	return nil
}

// DeleteList deletes a spelling list
func (s *ListService) DeleteList(listID, userID int64) error {
	// Get list to verify family access
	list, err := s.GetList(listID)
	if err != nil {
		return err
	}

	// Public lists cannot be deleted
	if list.IsPublic {
		return errors.New("cannot delete public lists")
	}

	// Verify user has access to the list's family
	if list.FamilyID == nil {
		return ErrNotFamilyMember
	}
	isMember, err := s.familyRepo.IsFamilyMember(userID, *list.FamilyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Get all words in the list to clean up their audio files
	words, err := s.listRepo.GetListWords(listID)
	if err != nil {
		log.Printf("Warning: Failed to get words for audio cleanup: %v", err)
		words = []models.Word{} // Continue with deletion even if we can't get words
	}

	// Delete list (this will cascade delete all words due to foreign key)
	if err := s.listRepo.DeleteList(listID); err != nil {
		return fmt.Errorf("failed to delete list: %w", err)
	}

	// Clean up audio files after successful deletion
	if s.ttsService != nil && len(words) > 0 {
		for _, word := range words {
			// Check and delete word audio file if not used elsewhere
			if word.AudioFilename != "" {
				isUsed, err := s.listRepo.IsAudioFileUsedByOtherWords(word.AudioFilename, word.ID)
				if err != nil {
					log.Printf("Warning: Failed to check if audio file is used: %v", err)
				} else if !isUsed {
					if err := s.ttsService.DeleteAudioFile(word.AudioFilename); err != nil {
						log.Printf("Warning: Failed to delete audio file '%s': %v", word.AudioFilename, err)
					} else {
						log.Printf("Deleted unused audio file: %s", word.AudioFilename)
					}
				}
			}

			// Check and delete definition audio file if not used elsewhere
			if word.DefinitionAudioFilename != "" {
				isUsed, err := s.listRepo.IsDefinitionAudioFileUsedByOtherWords(word.DefinitionAudioFilename, word.ID)
				if err != nil {
					log.Printf("Warning: Failed to check if definition audio file is used: %v", err)
				} else if !isUsed {
					if err := s.ttsService.DeleteAudioFile(word.DefinitionAudioFilename); err != nil {
						log.Printf("Warning: Failed to delete definition audio file '%s': %v", word.DefinitionAudioFilename, err)
					} else {
						log.Printf("Deleted unused definition audio file: %s", word.DefinitionAudioFilename)
					}
				}
			}
		}
	}

	return nil
}

// AddWord adds a word to a spelling list
func (s *ListService) AddWord(listID, userID int64, wordText string, difficulty int, definition string) (*models.Word, error) {
	// Get list to verify access
	list, err := s.GetList(listID)
	if err != nil {
		return nil, err
	}

	// Public lists cannot be modified
	if list.IsPublic {
		return nil, errors.New("cannot modify public lists")
	}

	// Verify user has access to the list's family
	if list.FamilyID == nil {
		return nil, ErrNotFamilyMember
	}
	isMember, err := s.familyRepo.IsFamilyMember(userID, *list.FamilyID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return nil, ErrNotFamilyMember
	}

	// Validate word
	wordText = strings.TrimSpace(wordText)
	if wordText == "" {
		return nil, errors.New("word text is required")
	}

	// Validate difficulty
	if difficulty < 1 || difficulty > 5 {
		difficulty = 1
	}

	// Trim definition if provided
	definition = strings.TrimSpace(definition)

	// Get current word count to determine position
	count, err := s.listRepo.GetWordCount(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get word count: %w", err)
	}

	// Add word at the end
	word, err := s.listRepo.AddWord(listID, wordText, difficulty, count+1, definition)
	if err != nil {
		return nil, fmt.Errorf("failed to add word: %w", err)
	}

	// Automatically generate audio file for the word
	if s.ttsService != nil {
		audioFilename, err := s.ttsService.GenerateAudioFile(wordText)
		if err != nil {
			log.Printf("Warning: Failed to generate audio for '%s': %v", wordText, err)
			// Don't fail the word creation, just log the warning
		} else {
			// Update the word with the audio filename
			if err := s.listRepo.UpdateWordAudio(word.ID, audioFilename); err != nil {
				log.Printf("Warning: Failed to update audio filename for word %d: %v", word.ID, err)
			} else {
				word.AudioFilename = audioFilename
				log.Printf("Generated audio for '%s': %s", wordText, audioFilename)
			}
		}

		// Generate audio for definition if provided
		if definition != "" {
			definitionPrefix := fmt.Sprintf("definition_%s", wordText)
			definitionAudioFilename, err := s.ttsService.GenerateAudioFileWithPrefix(definition, definitionPrefix)
			if err != nil {
				log.Printf("Warning: Failed to generate definition audio for '%s': %v", wordText, err)
			} else {
				if err := s.listRepo.UpdateWordDefinitionAudio(word.ID, definitionAudioFilename); err != nil {
					log.Printf("Warning: Failed to update definition audio filename for word %d: %v", word.ID, err)
				} else {
					word.DefinitionAudioFilename = definitionAudioFilename
					log.Printf("Generated definition audio for '%s': %s", wordText, definitionAudioFilename)
				}
			}
		}
	}

	return word, nil
}

// BulkAddWords adds multiple words at once from a comma-separated or newline-separated list
func (s *ListService) BulkAddWords(listID, userID int64, wordsText string, difficulty int) error {
	// Get list to verify access
	list, err := s.GetList(listID)
	if err != nil {
		return err
	}

	// Public lists cannot be modified
	if list.IsPublic {
		return errors.New("cannot modify public lists")
	}

	// Verify user has access to the list's family
	if list.FamilyID == nil {
		return ErrNotFamilyMember
	}
	isMember, err := s.familyRepo.IsFamilyMember(userID, *list.FamilyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Validate difficulty
	if difficulty < 1 || difficulty > 5 {
		difficulty = 3
	}

	// Parse words - handle both comma-separated and newline-separated
	wordsText = strings.TrimSpace(wordsText)
	if wordsText == "" {
		return errors.New("no words provided")
	}

	var words []string

	// Check if comma-separated
	if strings.Contains(wordsText, ",") {
		words = strings.Split(wordsText, ",")
	} else {
		// Assume newline-separated
		words = strings.Split(wordsText, "\n")
	}

	// Clean up and deduplicate words
	wordSet := make(map[string]bool)
	var cleanWords []string
	for _, word := range words {
		cleaned := strings.TrimSpace(word)
		if cleaned != "" && !wordSet[strings.ToLower(cleaned)] {
			wordSet[strings.ToLower(cleaned)] = true
			cleanWords = append(cleanWords, cleaned)
		}
	}

	if len(cleanWords) == 0 {
		return errors.New("no valid words found")
	}

	// Get current word count for positioning
	count, err := s.listRepo.GetWordCount(listID)
	if err != nil {
		return fmt.Errorf("failed to get word count: %w", err)
	}

	// Add each word
	addedCount := 0
	for i, wordText := range cleanWords {
		word, err := s.listRepo.AddWord(listID, wordText, difficulty, count+i+1, "")
		if err != nil {
			log.Printf("Warning: Failed to add word '%s': %v", wordText, err)
			continue
		}
		addedCount++

		// Automatically generate audio file
		if s.ttsService != nil {
			audioFilename, err := s.ttsService.GenerateAudioFile(wordText)
			if err != nil {
				log.Printf("Warning: Failed to generate audio for '%s': %v", wordText, err)
			} else {
				if err := s.listRepo.UpdateWordAudio(word.ID, audioFilename); err != nil {
					log.Printf("Warning: Failed to update audio filename for word %d: %v", word.ID, err)
				} else {
					log.Printf("Generated audio for '%s': %s", wordText, audioFilename)
				}
			}
		}
	}

	if addedCount == 0 {
		return errors.New("failed to add any words")
	}

	log.Printf("Bulk added %d words to list %d", addedCount, listID)
	return nil
}

// BulkAddWordsWithProgress adds multiple words with progress reporting
func (s *ListService) BulkAddWordsWithProgress(listID, userID int64, wordsText string, difficulty int, progressCallback func(total, processed, failed int)) error {
	// Get list to verify access
	list, err := s.GetList(listID)
	if err != nil {
		return err
	}

	// Public lists cannot be modified
	if list.IsPublic {
		return errors.New("cannot modify public lists")
	}

	// Verify user has access to the list's family
	if list.FamilyID == nil {
		return ErrNotFamilyMember
	}
	isMember, err := s.familyRepo.IsFamilyMember(userID, *list.FamilyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Validate difficulty
	if difficulty < 1 || difficulty > 5 {
		difficulty = 3
	}

	// Parse words - handle both comma-separated and newline-separated
	wordsText = strings.TrimSpace(wordsText)
	if wordsText == "" {
		return errors.New("no words provided")
	}

	var words []string

	// Check if comma-separated
	if strings.Contains(wordsText, ",") {
		words = strings.Split(wordsText, ",")
	} else {
		// Assume newline-separated
		words = strings.Split(wordsText, "\n")
	}

	// Clean up and deduplicate words
	wordSet := make(map[string]bool)
	var cleanWords []string
	for _, word := range words {
		cleaned := strings.TrimSpace(word)
		if cleaned != "" && !wordSet[strings.ToLower(cleaned)] {
			wordSet[strings.ToLower(cleaned)] = true
			cleanWords = append(cleanWords, cleaned)
		}
	}

	if len(cleanWords) == 0 {
		return errors.New("no valid words found")
	}

	// Get current word count for positioning
	count, err := s.listRepo.GetWordCount(listID)
	if err != nil {
		return fmt.Errorf("failed to get word count: %w", err)
	}

	total := len(cleanWords)
	processed := 0
	failed := 0

	// Report initial progress
	if progressCallback != nil {
		progressCallback(total, processed, failed)
	}

	// Add each word
	for i, wordText := range cleanWords {
		word, err := s.listRepo.AddWord(listID, wordText, difficulty, count+i+1, "")
		if err != nil {
			log.Printf("Warning: Failed to add word '%s': %v", wordText, err)
			failed++
			processed++
			if progressCallback != nil {
				progressCallback(total, processed, failed)
			}
			continue
		}

		// Automatically generate audio file
		if s.ttsService != nil {
			audioFilename, err := s.ttsService.GenerateAudioFile(wordText)
			if err != nil {
				log.Printf("Warning: Failed to generate audio for '%s': %v", wordText, err)
			} else {
				if err := s.listRepo.UpdateWordAudio(word.ID, audioFilename); err != nil {
					log.Printf("Warning: Failed to update audio filename for word %d: %v", word.ID, err)
				} else {
					log.Printf("Generated audio for '%s': %s", wordText, audioFilename)
				}
			}
		}

		processed++

		// Report progress after each word
		if progressCallback != nil {
			progressCallback(total, processed, failed)
		}
	}

	if processed == failed {
		return errors.New("failed to add any words")
	}

	log.Printf("Bulk added %d words to list %d (%d failed)", processed-failed, listID, failed)
	return nil
}

// GetListWords retrieves all words for a list
func (s *ListService) GetListWords(listID, userID int64) ([]models.Word, error) {
	// Get list to verify access
	list, err := s.GetList(listID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to this list
	hasAccess, err := s.hasAccessToList(userID, list)
	if err != nil {
		return nil, fmt.Errorf("failed to verify access: %w", err)
	}
	if !hasAccess {
		return nil, ErrNotFamilyMember
	}

	words, err := s.listRepo.GetListWords(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get words: %w", err)
	}

	return words, nil
}

// UpdateWord updates a word's text and difficulty
func (s *ListService) UpdateWord(wordID, userID int64, wordText string, difficulty int, definition string) error {
	// Get word to get list ID
	word, err := s.listRepo.GetWordByID(wordID)
	if err != nil {
		return fmt.Errorf("failed to get word: %w", err)
	}
	if word == nil {
		return ErrWordNotFound
	}

	// Get list to verify family access
	list, err := s.GetList(word.SpellingListID)
	if err != nil {
		return err
	}

	// Public lists cannot be modified
	if list.IsPublic {
		return errors.New("cannot modify public lists")
	}

	// Verify user has access to the list's family
	if list.FamilyID == nil {
		return ErrNotFamilyMember
	}
	isMember, err := s.familyRepo.IsFamilyMember(userID, *list.FamilyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Validate word
	wordText = strings.TrimSpace(wordText)
	if wordText == "" {
		return errors.New("word text is required")
	}

	// Validate difficulty
	if difficulty < 1 || difficulty > 5 {
		difficulty = 1
	}

	// Trim definition
	definition = strings.TrimSpace(definition)

	// Update word
	if err := s.listRepo.UpdateWord(wordID, wordText, difficulty, definition); err != nil {
		return fmt.Errorf("failed to update word: %w", err)
	}

	// Generate audio for the word
	audioFilename, err := s.ttsService.GenerateAudioFile(wordText)
	if err != nil {
		return fmt.Errorf("failed to generate word audio: %w", err)
	}
	if err := s.listRepo.UpdateWordAudio(wordID, audioFilename); err != nil {
		return fmt.Errorf("failed to save word audio filename: %w", err)
	}

	// Generate audio for definition if provided
	if definition != "" {
		definitionPrefix := fmt.Sprintf("definition_%s", wordText)
		definitionAudioFilename, err := s.ttsService.GenerateAudioFileWithPrefix(definition, definitionPrefix)
		if err != nil {
			return fmt.Errorf("failed to generate definition audio: %w", err)
		}
		if err := s.listRepo.UpdateWordDefinitionAudio(wordID, definitionAudioFilename); err != nil {
			return fmt.Errorf("failed to save definition audio filename: %w", err)
		}
	} else {
		// Clear definition audio if definition was removed
		if err := s.listRepo.UpdateWordDefinitionAudio(wordID, ""); err != nil {
			return fmt.Errorf("failed to clear definition audio filename: %w", err)
		}
	}

	return nil
}

// DeleteWord deletes a word from a list
func (s *ListService) DeleteWord(wordID, userID int64) error {
	// Get word to get list ID and audio filenames
	word, err := s.listRepo.GetWordByID(wordID)
	if err != nil {
		return fmt.Errorf("failed to get word: %w", err)
	}
	if word == nil {
		return ErrWordNotFound
	}

	// Get list to verify family access
	list, err := s.GetList(word.SpellingListID)
	if err != nil {
		return err
	}

	// Public lists cannot be modified
	if list.IsPublic {
		return errors.New("cannot modify public lists")
	}

	// Verify user has access to the list's family
	if list.FamilyID == nil {
		return ErrNotFamilyMember
	}
	isMember, err := s.familyRepo.IsFamilyMember(userID, *list.FamilyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Store audio filenames before deleting the word
	wordAudioFilename := word.AudioFilename
	definitionAudioFilename := word.DefinitionAudioFilename

	// Delete word from database
	if err := s.listRepo.DeleteWord(wordID); err != nil {
		return fmt.Errorf("failed to delete word: %w", err)
	}

	// Clean up audio files if they're not used by other words
	if s.ttsService != nil {
		// Check and delete word audio file if not used elsewhere
		if wordAudioFilename != "" {
			isUsed, err := s.listRepo.IsAudioFileUsedByOtherWords(wordAudioFilename, wordID)
			if err != nil {
				log.Printf("Warning: Failed to check if audio file is used: %v", err)
			} else if !isUsed {
				if err := s.ttsService.DeleteAudioFile(wordAudioFilename); err != nil {
					log.Printf("Warning: Failed to delete audio file '%s': %v", wordAudioFilename, err)
				} else {
					log.Printf("Deleted unused audio file: %s", wordAudioFilename)
				}
			}
		}

		// Check and delete definition audio file if not used elsewhere
		if definitionAudioFilename != "" {
			isUsed, err := s.listRepo.IsDefinitionAudioFileUsedByOtherWords(definitionAudioFilename, wordID)
			if err != nil {
				log.Printf("Warning: Failed to check if definition audio file is used: %v", err)
			} else if !isUsed {
				if err := s.ttsService.DeleteAudioFile(definitionAudioFilename); err != nil {
					log.Printf("Warning: Failed to delete definition audio file '%s': %v", definitionAudioFilename, err)
				} else {
					log.Printf("Deleted unused definition audio file: %s", definitionAudioFilename)
				}
			}
		}
	}

	return nil
}

// AssignListToKid assigns a spelling list to a kid
func (s *ListService) AssignListToKid(listID, kidID, userID int64) error {
	// Get list to verify access
	list, err := s.GetList(listID)
	if err != nil {
		return err
	}

	// Check if user has access to this list (public lists can be assigned)
	hasAccess, err := s.hasAccessToList(userID, list)
	if err != nil {
		return fmt.Errorf("failed to verify access: %w", err)
	}
	if !hasAccess {
		return ErrNotFamilyMember
	}

	// Assign list
	if err := s.listRepo.AssignListToKid(listID, kidID, userID); err != nil {
		return fmt.Errorf("failed to assign list: %w", err)
	}

	return nil
}

// UnassignListFromKid removes a list assignment
func (s *ListService) UnassignListFromKid(listID, kidID, userID int64) error {
	// Get list to verify access
	list, err := s.GetList(listID)
	if err != nil {
		return err
	}

	// Check if user has access to this list
	hasAccess, err := s.hasAccessToList(userID, list)
	if err != nil {
		return fmt.Errorf("failed to verify access: %w", err)
	}
	if !hasAccess {
		return ErrNotFamilyMember
	}

	// Unassign list
	if err := s.listRepo.UnassignListFromKid(listID, kidID); err != nil {
		return fmt.Errorf("failed to unassign list: %w", err)
	}

	return nil
}

// GetKidAssignedLists retrieves all lists assigned to a kid
func (s *ListService) GetKidAssignedLists(kidID int64) ([]models.SpellingList, error) {
	lists, err := s.listRepo.GetKidAssignedLists(kidID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned lists: %w", err)
	}
	return lists, nil
}

// GetListAssignedKids retrieves all kids assigned to a list
func (s *ListService) GetListAssignedKids(listID, userID int64) ([]models.Kid, error) {
	// Get list to verify access
	list, err := s.GetList(listID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to this list
	hasAccess, err := s.hasAccessToList(userID, list)
	if err != nil {
		return nil, fmt.Errorf("failed to verify access: %w", err)
	}
	if !hasAccess {
		return nil, ErrNotFamilyMember
	}

	kids, err := s.listRepo.GetListAssignedKids(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned kids: %w", err)
	}

	return kids, nil
}

// GenerateMissingAudio checks all words and generates any missing audio files
func (s *ListService) GenerateMissingAudio() error {
	if s.ttsService == nil {
		return nil // TTS service not configured, skip
	}

	log.Println("Checking for missing audio files...")

	// Get all words
	words, err := s.listRepo.GetAllWords()
	if err != nil {
		return fmt.Errorf("failed to get all words: %w", err)
	}

	wordAudioGenerated := 0
	definitionAudioGenerated := 0

	for _, word := range words {
		// Check and generate word audio if missing
		if word.AudioFilename == "" {
			audioFilename, err := s.ttsService.GenerateAudioFile(word.WordText)
			if err != nil {
				log.Printf("Warning: Failed to generate audio for word '%s' (ID: %d): %v", word.WordText, word.ID, err)
			} else {
				if err := s.listRepo.UpdateWordAudio(word.ID, audioFilename); err != nil {
					log.Printf("Warning: Failed to save audio filename for word %d: %v", word.ID, err)
				} else {
					wordAudioGenerated++
					log.Printf("Generated audio for '%s': %s", word.WordText, audioFilename)
				}
			}
		}

		// Check and generate definition audio if missing
		if word.Definition != "" && word.DefinitionAudioFilename == "" {
			definitionPrefix := fmt.Sprintf("definition_%s", word.WordText)
			definitionAudioFilename, err := s.ttsService.GenerateAudioFileWithPrefix(word.Definition, definitionPrefix)
			if err != nil {
				log.Printf("Warning: Failed to generate definition audio for '%s' (ID: %d): %v", word.WordText, word.ID, err)
			} else {
				if err := s.listRepo.UpdateWordDefinitionAudio(word.ID, definitionAudioFilename); err != nil {
					log.Printf("Warning: Failed to save definition audio filename for word %d: %v", word.ID, err)
				} else {
					definitionAudioGenerated++
					log.Printf("Generated definition audio for '%s': %s", word.WordText, definitionAudioFilename)
				}
			}
		}
	}

	if wordAudioGenerated > 0 || definitionAudioGenerated > 0 {
		log.Printf("Audio generation complete: %d word audio files, %d definition audio files generated", wordAudioGenerated, definitionAudioGenerated)
	} else {
		log.Println("All audio files already exist")
	}

	return nil
}

// CleanupOrphanedAudioFiles removes audio files that are not referenced in the database
func (s *ListService) CleanupOrphanedAudioFiles() error {
	if s.ttsService == nil {
		return nil // TTS service not configured, skip
	}

	log.Println("Checking for orphaned audio files...")

	// Get all audio files from filesystem
	filesOnDisk, err := s.ttsService.GetAllAudioFiles()
	if err != nil {
		return fmt.Errorf("failed to get audio files from disk: %w", err)
	}

	// Get all audio filenames referenced in database
	referencedFiles, err := s.listRepo.GetAllAudioFilenames()
	if err != nil {
		return fmt.Errorf("failed to get referenced audio filenames: %w", err)
	}

	// Create a map for quick lookup of referenced files
	referenced := make(map[string]bool)
	for _, filename := range referencedFiles {
		referenced[filename] = true
	}

	// Delete files that are not referenced
	deletedCount := 0
	for _, filename := range filesOnDisk {
		if !referenced[filename] {
			if err := s.ttsService.DeleteAudioFile(filename); err != nil {
				log.Printf("Warning: Failed to delete orphaned audio file '%s': %v", filename, err)
			} else {
				deletedCount++
				log.Printf("Deleted orphaned audio file: %s", filename)
			}
		}
	}

	if deletedCount > 0 {
		log.Printf("Audio cleanup complete: %d orphaned files deleted", deletedCount)
	} else {
		log.Println("No orphaned audio files found")
	}

	return nil
}
