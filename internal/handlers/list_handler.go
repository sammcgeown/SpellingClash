package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
	"wordclash/internal/service"
)

// BulkImportProgress tracks the progress of a bulk import operation
type BulkImportProgress struct {
	Total     int
	Processed int
	Failed    int
	Completed bool
	Error     string
	mu        sync.RWMutex
}

// ListHandler handles spelling list HTTP requests
type ListHandler struct {
	listService    *service.ListService
	familyService  *service.FamilyService
	middleware     *Middleware
	templates      *template.Template
	importProgress map[string]*BulkImportProgress
	progressMu     sync.RWMutex
}

// NewListHandler creates a new list handler
func NewListHandler(listService *service.ListService, familyService *service.FamilyService, middleware *Middleware, templates *template.Template) *ListHandler {
	return &ListHandler{
		listService:    listService,
		familyService:  familyService,
		middleware:     middleware,
		templates:      templates,
		importProgress: make(map[string]*BulkImportProgress),
	}
}

// ShowLists displays the lists management page
func (h *ListHandler) ShowLists(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get all user's lists with assignment counts
	lists, err := h.listService.GetAllUserListsWithAssignments(user.ID)
	if err != nil {
		log.Printf("Error getting user lists: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get user's families for creating new lists
	families, err := h.familyService.GetUserFamilies(user.ID)
	if err != nil {
		log.Printf("Error getting user families: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get CSRF token
	csrfToken := h.getCSRFToken(r)

	data := map[string]interface{}{
		"Title":     "Manage Lists - WordClash",
		"User":      user,
		"Lists":     lists,
		"Families":  families,
		"CSRFToken": csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "lists.tmpl", data); err != nil {
		log.Printf("Error rendering lists template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// CreateList handles list creation
func (h *ListHandler) CreateList(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	familyIDStr := r.FormValue("family_id")

	familyID, err := strconv.ParseInt(familyIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid family ID", http.StatusBadRequest)
		return
	}

	list, err := h.listService.CreateList(familyID, user.ID, name, description)
	if err != nil {
		log.Printf("Error creating list: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Redirect to the new list's detail page
	http.Redirect(w, r, "/parent/lists/"+strconv.FormatInt(list.ID, 10), http.StatusSeeOther)
}

// ViewList displays a specific list with its words
func (h *ListHandler) ViewList(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	listIDStr := r.PathValue("id")
	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	// Get list
	list, err := h.listService.GetList(listID)
	if err != nil {
		log.Printf("Error getting list: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get words
	words, err := h.listService.GetListWords(listID, user.ID)
	if err != nil {
		log.Printf("Error getting list words: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get assigned kids
	assignedKids, err := h.listService.GetListAssignedKids(listID, user.ID)
	if err != nil {
		log.Printf("Error getting assigned kids: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get all family kids for assignment
	familyKids, err := h.familyService.GetFamilyKids(list.FamilyID, user.ID)
	if err != nil {
		log.Printf("Error getting family kids: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get CSRF token
	csrfToken := h.getCSRFToken(r)

	data := map[string]interface{}{
		"Title":        list.Name + " - WordClash",
		"User":         user,
		"List":         list,
		"Words":        words,
		"AssignedKids": assignedKids,
		"FamilyKids":   familyKids,
		"CSRFToken":    csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "list_detail.tmpl", data); err != nil {
		log.Printf("Error rendering list detail template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// UpdateList handles list updates
func (h *ListHandler) UpdateList(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	listIDStr := r.PathValue("id")
	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")

	if err := h.listService.UpdateList(listID, user.ID, name, description); err != nil {
		log.Printf("Error updating list: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/parent/lists/"+listIDStr, http.StatusSeeOther)
}

// DeleteList handles list deletion
func (h *ListHandler) DeleteList(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	listIDStr := r.PathValue("id")
	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	if err := h.listService.DeleteList(listID, user.ID); err != nil {
		log.Printf("Error deleting list: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/parent/lists", http.StatusSeeOther)
}

// AddWord handles adding a word to a list
func (h *ListHandler) AddWord(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	listIDStr := r.PathValue("id")
	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	wordText := r.FormValue("word")
	difficultyStr := r.FormValue("difficulty")

	difficulty, err := strconv.Atoi(difficultyStr)
	if err != nil {
		difficulty = 1
	}

	_, err = h.listService.AddWord(listID, user.ID, wordText, difficulty)
	if err != nil {
		log.Printf("Error adding word: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/parent/lists/"+listIDStr, http.StatusSeeOther)
}

// BulkAddWords handles adding multiple words at once with progress tracking
func (h *ListHandler) BulkAddWords(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	listIDStr := r.PathValue("id")
	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	wordsText := r.FormValue("words")
	difficultyStr := r.FormValue("difficulty")

	difficulty, err := strconv.Atoi(difficultyStr)
	if err != nil {
		difficulty = 3
	}

	// Create a unique progress ID for this import
	progressID := fmt.Sprintf("%d-%d", user.ID, listID)

	// Initialize progress tracking
	progress := &BulkImportProgress{
		Total:     0,
		Processed: 0,
		Failed:    0,
		Completed: false,
	}

	h.progressMu.Lock()
	h.importProgress[progressID] = progress
	h.progressMu.Unlock()

	// Start bulk import in background
	go func() {
		defer func() {
			progress.mu.Lock()
			progress.Completed = true
			progress.mu.Unlock()
		}()

		progressCallback := func(total, processed, failed int) {
			progress.mu.Lock()
			progress.Total = total
			progress.Processed = processed
			progress.Failed = failed
			progress.mu.Unlock()
		}

		if err := h.listService.BulkAddWordsWithProgress(listID, user.ID, wordsText, difficulty, progressCallback); err != nil {
			log.Printf("Error bulk adding words: %v", err)
			progress.mu.Lock()
			progress.Error = err.Error()
			progress.mu.Unlock()
		}
	}()

	// Return progress ID to client
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"progress_id": progressID})
}

// DeleteWord handles word deletion
func (h *ListHandler) DeleteWord(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	listIDStr := r.PathValue("listId")
	wordIDStr := r.PathValue("wordId")

	wordID, err := strconv.ParseInt(wordIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid word ID", http.StatusBadRequest)
		return
	}

	if err := h.listService.DeleteWord(wordID, user.ID); err != nil {
		log.Printf("Error deleting word: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/parent/lists/"+listIDStr, http.StatusSeeOther)
}

// AssignList handles assigning a list to a kid
func (h *ListHandler) AssignList(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	listIDStr := r.PathValue("listId")
	kidIDStr := r.PathValue("kidId")

	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	kidID, err := strconv.ParseInt(kidIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid kid ID", http.StatusBadRequest)
		return
	}

	if err := h.listService.AssignListToKid(listID, kidID, user.ID); err != nil {
		log.Printf("Error assigning list: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/parent/lists/"+listIDStr, http.StatusSeeOther)
}

// UnassignList handles unassigning a list from a kid
func (h *ListHandler) UnassignList(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	listIDStr := r.PathValue("listId")
	kidIDStr := r.PathValue("kidId")

	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	kidID, err := strconv.ParseInt(kidIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid kid ID", http.StatusBadRequest)
		return
	}

	if err := h.listService.UnassignListFromKid(listID, kidID, user.ID); err != nil {
		log.Printf("Error unassigning list: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/parent/lists/"+listIDStr, http.StatusSeeOther)
}

// GetBulkImportProgress returns the current progress of a bulk import
func (h *ListHandler) GetBulkImportProgress(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	listIDStr := r.PathValue("id")
	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	progressID := fmt.Sprintf("%d-%d", user.ID, listID)

	h.progressMu.RLock()
	progress, exists := h.importProgress[progressID]
	h.progressMu.RUnlock()

	if !exists {
		http.Error(w, "No import in progress", http.StatusNotFound)
		return
	}

	progress.mu.RLock()
	defer progress.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":     progress.Total,
		"processed": progress.Processed,
		"failed":    progress.Failed,
		"completed": progress.Completed,
		"error":     progress.Error,
	})

	// Clean up completed imports
	if progress.Completed {
		h.progressMu.Lock()
		delete(h.importProgress, progressID)
		h.progressMu.Unlock()
	}
}

// getCSRFToken is a helper to get CSRF token from session
func (h *ListHandler) getCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}
	token, _ := h.middleware.GetCSRFToken(cookie.Value)
	return token
}
