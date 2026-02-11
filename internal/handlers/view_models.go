package handlers

import (
	"time"

	"spellingclash/internal/models"
	"spellingclash/internal/repository"
)

type LoginViewData struct {
	Title          string
	OAuthProviders []OAuthProviderView
	Error          string
	Email          string
	Success        string
}

type RegisterViewData struct {
	Title          string
	FamilyCode     string
	OAuthProviders []OAuthProviderView
	Error          string
	Email          string
	Name           string
}

type ForgotPasswordViewData struct {
	Title   string
	Success string
	Error   string
}

type ResetPasswordViewData struct {
	Title string
	Token string
	Error string
}

type AdminDashboardViewData struct {
	Title       string
	User        *models.User
	PublicLists []models.SpellingList
	CSRFToken   string
	Version     string
}

type AdminUserWithFamily struct {
	models.User
	FamilyCode string
}

type AdminParentsViewData struct {
	Title     string
	User      *models.User
	Users     []AdminUserWithFamily
	CSRFToken string
}

type AdminFamiliesViewData struct {
	Title         string
	User          *models.User
	Families      []models.Family
	Users         []models.User
	FamilyMembers map[string][]models.User
	CSRFToken     string
}

type AdminKidsViewData struct {
	Title     string
	User      *models.User
	Kids      []models.Kid
	CSRFToken string
}

type AdminDatabaseViewData struct {
	Title     string
	User      *models.User
	Stats     *DatabaseStats
	CSRFToken string
	Error     string
	Success   string
}

type ParentDashboardViewData struct {
	Title         string
	User          *models.User
	Families      []models.Family
	Kids          []models.Kid
	FamilyMembers []models.FamilyMember
	ParentUsers   []models.User
	CSRFToken     string
}

type ParentFamilyViewData struct {
	Title     string
	User      *models.User
	Families  []models.Family
	CSRFToken string
}

type ParentKidsViewData struct {
	Title     string
	User      *models.User
	Families  []models.Family
	Kids      []models.KidWithLists
	AllLists  []models.ListSummary
	CSRFToken string
}

type ParentListsViewData struct {
	Title     string
	User      *models.User
	Lists     []models.ListSummary
	Families  []models.Family
	CSRFToken string
}

type ListDetailViewData struct {
	Title        string
	User         *models.User
	List         *models.SpellingList
	Words        []models.Word
	AssignedKids []models.Kid
	FamilyKids   []models.Kid
	CSRFToken    string
}

type KidSelectViewData struct {
	Title    string
	HasError bool
}

type KidLoginViewData struct {
	Title    string
	Kid      *models.Kid
	HasError bool
}

type KidDashboardViewData struct {
	Title          string
	Kid            *models.Kid
	AssignedLists  []models.SpellingList
	TotalPoints    int
	TotalSessions  int
	RecentSessions []models.PracticeSession
}

type KidDetailsViewData struct {
	Title           string
	User            *models.User
	Kid             *models.Kid
	AssignedLists   []models.SpellingList
	AllLists        []models.ListSummary
	StrugglingWords []repository.StrugglingWord
	Stats           *models.KidStats
	CSRFToken       string
}

type StrugglingWordsViewData struct {
	Kid             *models.Kid
	StrugglingWords []repository.StrugglingWord
	Stats           *models.KidStats
}

type PracticeViewData struct {
	Title              string
	Kid                *models.Kid
	Word               *models.Word
	CurrentIndex       int
	TotalWords         int
	CorrectCount       int
	TotalPoints        int
	WordTiming         time.Time
	ProgressPercentage int
}

type PracticeResultsViewData struct {
	Title       string
	Kid         *models.Kid
	Session     *models.PracticeSession
	Attempts    []models.WordAttempt
	Accuracy    float64
	TotalPoints int
}

type MissingLetterViewData struct {
	Title     string
	Kid       *models.Kid
	GameState *models.MissingLetterGameState
}

type MissingLetterResultsViewData struct {
	Title   string
	Kid     *models.Kid
	Results *models.MissingLetterSession
}

type MissingLetterGameStateViewData struct {
	Kid       *models.Kid
	GameState *models.MissingLetterGameState
}

type HangmanViewData struct {
	Title     string
	Kid       *models.Kid
	GameState *models.HangmanGameState
}

type HangmanResultsViewData struct {
	Title   string
	Kid     *models.Kid
	Results *models.HangmanSession
}

type HangmanGameStateViewData struct {
	Kid       *models.Kid
	GameState *models.HangmanGameState
}
