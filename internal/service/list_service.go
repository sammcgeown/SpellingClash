package service

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"wordclash/internal/models"
	"wordclash/internal/repository"
	"wordclash/internal/utils"
)

var (
	ErrListNotFound = errors.New("list not found")
	ErrWordNotFound = errors.New("word not found")
)

// ListService handles spelling list business logic
type ListService struct {
	listRepo   *repository.ListRepository
	familyRepo *repository.FamilyRepository
	ttsService *utils.TTSService
}

// NewListService creates a new list service
func NewListService(listRepo *repository.ListRepository, familyRepo *repository.FamilyRepository, ttsService *utils.TTSService) *ListService {
	return &ListService{
		listRepo:   listRepo,
		familyRepo: familyRepo,
		ttsService: ttsService,
	}
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
	// Seed Year 1 and 2 Words
	if err := s.seedYear1And2Words(); err != nil {
		return err
	}

	// Seed Year 3 and 4 Words
	if err := s.seedYear3And4Words(); err != nil {
		return err
	}

	// Seed Year 5 and 6 Words
	if err := s.seedYear5And6Words(); err != nil {
		return err
	}

	return nil
}

// seedYear1And2Words creates the Year 1 and 2 public list
func (s *ListService) seedYear1And2Words() error {
	listName := "Year 1 and 2 Words"

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
	list, err := s.listRepo.CreatePublicList(listName, "UK National Curriculum common exception words for Years 1 and 2")
	if err != nil {
		return fmt.Errorf("failed to create %s public list: %w", listName, err)
	}

	// Year 1 and 2 common exception words
	words := []string{
		// Year 1 words
		"the", "a", "do", "to", "today", "of", "said", "says", "are", "were",
		"was", "is", "his", "has", "I", "you", "your", "they", "be", "he",
		"me", "she", "we", "no", "go", "so", "by", "my", "here", "there",
		"where", "love", "come", "some", "one", "once", "ask", "friend", "school", "put",
		"push", "pull", "full", "house", "our",
		// Year 2 words
		"door", "floor", "poor", "because", "find", "kind", "mind", "behind", "child", "children",
		"wild", "climb", "most", "only", "both", "old", "cold", "gold", "hold", "told",
		"every", "everybody", "even", "great", "break", "steak", "pretty", "beautiful", "after", "fast",
		"last", "past", "father", "class", "grass", "pass", "plant", "path", "bath", "hour",
		"move", "prove", "improve", "sure", "sugar", "eye", "could", "should", "would", "who",
		"whole", "any", "many", "clothes", "busy", "people", "water", "again", "half", "money",
		"Mr", "Mrs", "parents", "Christmas",
	}

	log.Printf("Adding %d words to %s list...", len(words), listName)

	// Add each word with audio generation
	for i, wordText := range words {
		word, err := s.listRepo.AddWord(list.ID, wordText, 2, i+1, "") // Default difficulty: 2 (easier for younger students), no definition
		if err != nil {
			log.Printf("Warning: Failed to add word '%s': %v", wordText, err)
			continue
		}

		// Generate audio file
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

	log.Printf("Successfully created default public list '%s' with %d words", listName, len(words))
	return nil
}

// seedYear3And4Words creates the Year 3 and 4 public list
func (s *ListService) seedYear3And4Words() error {
	listName := "Year 3 and 4 Words"

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
	list, err := s.listRepo.CreatePublicList(listName, "UK National Curriculum statutory words for Years 3 and 4")
	if err != nil {
		return fmt.Errorf("failed to create %s public list: %w", listName, err)
	}

	// Year 3 and 4 statutory words
	words := []string{
		"accident", "accidentally", "actual", "actually", "address", "answer", "appear", "arrive",
		"believe", "bicycle", "breath", "breathe", "build", "busy", "business", "calendar",
		"caught", "centre", "century", "certain", "circle", "complete", "consider", "continue",
		"decide", "describe", "different", "difficult", "disappear", "early", "earth", "eight",
		"eighth", "enough", "exercise", "experience", "experiment", "extreme", "famous", "favourite",
		"February", "forward", "forwards", "fruit", "grammar", "group", "guard", "guide",
		"heard", "heart", "height", "history", "imagine", "increase", "important", "interest",
		"island", "knowledge", "learn", "length", "library", "material", "medicine", "mention",
		"minute", "natural", "naughty", "notice", "occasion", "occasionally", "often", "opposite",
		"ordinary", "particular", "peculiar", "perhaps", "popular", "position", "possess", "possession",
		"possible", "potatoes", "pressure", "probably", "promise", "purpose", "quarter", "question",
		"recent", "regular", "reign", "remember", "sentence", "separate", "special", "straight",
		"strange", "strength", "suppose", "surprise", "therefore", "though", "although", "thought",
		"through", "various", "weight", "woman", "women",
	}

	log.Printf("Adding %d words to %s list...", len(words), listName)

	// Add each word with audio generation
	for i, wordText := range words {
		word, err := s.listRepo.AddWord(list.ID, wordText, 3, i+1, "") // Default difficulty: 3 (medium), no definition
		if err != nil {
			log.Printf("Warning: Failed to add word '%s': %v", wordText, err)
			continue
		}

		// Generate audio file
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

	log.Printf("Successfully created default public list '%s' with %d words", listName, len(words))
	return nil
}

// seedYear5And6Words creates the Year 5 and 6 public list
func (s *ListService) seedYear5And6Words() error {
	listName := "Year 5 and 6 Words"

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
	list, err := s.listRepo.CreatePublicList(listName, "UK National Curriculum statutory words for Years 5 and 6")
	if err != nil {
		return fmt.Errorf("failed to create %s public list: %w", listName, err)
	}

	// Year 5 and 6 statutory words
	words := []string{
		"accommodate", "accompany", "according", "achieve", "aggressive", "amateur", "ancient", "apparent",
		"appreciate", "attached", "available", "average", "awkward", "bargain", "bruise", "category",
		"cemetery", "committee", "communicate", "community", "competition", "conscience", "conscious", "controversy",
		"convenience", "correspond", "criticise", "curiosity", "definite", "desperate", "determined", "develop",
		"dictionary", "disastrous", "embarrass", "environment", "equip", "equipment", "especially", "exaggerate",
		"excellent", "existence", "explanation", "familiar", "foreign", "forty", "frequently", "government",
		"guarantee", "harass", "hindrance", "identity", "immediately", "individual", "interfere", "interrupt",
		"language", "leisure", "lightning", "marvellous", "mischievous", "muscle", "necessary", "neighbour",
		"nuisance", "occupy", "occur", "opportunity", "parliament", "persuade", "physical", "prejudice",
		"privilege", "profession", "programme", "pronunciation", "queue", "recognise", "recommend", "relevant",
		"restaurant", "rhyme", "rhythm", "sacrifice", "secretary", "shoulder", "signature", "sincere",
		"sincerely", "soldier", "stomach", "sufficient", "suggest", "symbol", "system", "temperature",
		"thorough", "twelfth", "variety", "vegetable", "vehicle", "yacht",
	}

	log.Printf("Adding %d words to %s list...", len(words), listName)

	// Add each word with audio generation
	for i, wordText := range words {
		word, err := s.listRepo.AddWord(list.ID, wordText, 4, i+1, "") // Default difficulty: 4 (harder for older students), no definition
		if err != nil {
			log.Printf("Warning: Failed to add word '%s': %v", wordText, err)
			continue
		}

		// Generate audio file
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

	log.Printf("Successfully created default public list '%s' with %d words", listName, len(words))
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
		allLists = append(allLists, models.ListSummary{
			SpellingList:     publicList,
			AssignedKidCount: 0, // We don't count cross-family assignments for public lists in this view
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

	// Delete list
	if err := s.listRepo.DeleteList(listID); err != nil {
		return fmt.Errorf("failed to delete list: %w", err)
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
			definitionAudioFilename, err := s.ttsService.GenerateAudioFile(definition)
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
func (s *ListService) UpdateWord(wordID, userID int64, wordText string, difficulty int) error {
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

	// Update word
	if err := s.listRepo.UpdateWord(wordID, wordText, difficulty); err != nil {
		return fmt.Errorf("failed to update word: %w", err)
	}

	return nil
}

// DeleteWord deletes a word from a list
func (s *ListService) DeleteWord(wordID, userID int64) error {
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

	// Delete word
	if err := s.listRepo.DeleteWord(wordID); err != nil {
		return fmt.Errorf("failed to delete word: %w", err)
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
