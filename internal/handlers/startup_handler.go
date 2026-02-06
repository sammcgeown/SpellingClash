package handlers

import (
	"html/template"
	"net/http"
	"sync"
)

// StartupStatus tracks the initialization progress
type StartupStatus struct {
	mu       sync.RWMutex
	Ready    bool
	Current  string
	Progress int
	Steps    []StartupStep
}

type StartupStep struct {
	Name      string
	Completed bool
}

var startupStatus = &StartupStatus{
	Ready:    false,
	Current:  "Initializing...",
	Progress: 0,
	Steps: []StartupStep{
		{Name: "Database connection", Completed: false},
		{Name: "Running migrations", Completed: false},
		{Name: "Loading templates", Completed: false},
		{Name: "Initializing services", Completed: false},
		{Name: "Seeding default lists", Completed: false},
		{Name: "Generating audio files", Completed: false},
		{Name: "Server ready", Completed: false},
	},
}

// SetCurrentStep updates the current initialization step
func SetCurrentStep(step string) {
	startupStatus.mu.Lock()
	defer startupStatus.mu.Unlock()
	startupStatus.Current = step
}

// CompleteStep marks a step as completed and updates progress
func CompleteStep(stepName string) {
	startupStatus.mu.Lock()
	defer startupStatus.mu.Unlock()
	
	for i := range startupStatus.Steps {
		if startupStatus.Steps[i].Name == stepName {
			startupStatus.Steps[i].Completed = true
			break
		}
	}
	
	completed := 0
	for _, step := range startupStatus.Steps {
		if step.Completed {
			completed++
		}
	}
	startupStatus.Progress = (completed * 100) / len(startupStatus.Steps)
}

// MarkReady marks the server as fully initialized
func MarkReady() {
	startupStatus.mu.Lock()
	defer startupStatus.mu.Unlock()
	startupStatus.Ready = true
	startupStatus.Current = "Server ready"
	startupStatus.Progress = 100
}

// IsReady returns whether the server is fully initialized
func IsReady() bool {
	startupStatus.mu.RLock()
	defer startupStatus.mu.RUnlock()
	return startupStatus.Ready
}

// ShowStartupStatus displays the startup status page
func ShowStartupStatus(w http.ResponseWriter, r *http.Request) {
	startupStatus.mu.RLock()
	defer startupStatus.mu.RUnlock()
	
	if startupStatus.Ready {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<meta http-equiv="refresh" content="2">
	<title>SpellingClash - Starting Up</title>
	<style>
		* {
			margin: 0;
			padding: 0;
			box-sizing: border-box;
		}
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			min-height: 100vh;
			display: flex;
			align-items: center;
			justify-content: center;
			padding: 20px;
		}
		.container {
			background: white;
			border-radius: 20px;
			padding: 40px;
			box-shadow: 0 20px 60px rgba(0,0,0,0.3);
			max-width: 500px;
			width: 100%;
		}
		h1 {
			color: #333;
			margin-bottom: 10px;
			font-size: 32px;
			text-align: center;
		}
		.subtitle {
			color: #666;
			text-align: center;
			margin-bottom: 30px;
			font-size: 16px;
		}
		.progress-bar {
			width: 100%;
			height: 12px;
			background: #e0e0e0;
			border-radius: 6px;
			overflow: hidden;
			margin-bottom: 20px;
		}
		.progress-fill {
			height: 100%;
			background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
			transition: width 0.3s ease;
		}
		.progress-text {
			text-align: center;
			color: #667eea;
			font-weight: 600;
			margin-bottom: 30px;
			font-size: 18px;
		}
		.steps {
			list-style: none;
		}
		.step {
			padding: 12px 0;
			border-bottom: 1px solid #f0f0f0;
			display: flex;
			align-items: center;
			gap: 12px;
		}
		.step:last-child {
			border-bottom: none;
		}
		.step-icon {
			width: 24px;
			height: 24px;
			border-radius: 50%;
			display: flex;
			align-items: center;
			justify-content: center;
			flex-shrink: 0;
		}
		.step-icon.completed {
			background: #10b981;
			color: white;
		}
		.step-icon.pending {
			background: #e0e0e0;
			color: #999;
		}
		.step-name {
			color: #333;
			font-size: 15px;
		}
		.step.completed .step-name {
			color: #10b981;
		}
		.current-status {
			text-align: center;
			color: #667eea;
			font-style: italic;
			margin-top: 20px;
			padding-top: 20px;
			border-top: 2px solid #f0f0f0;
		}
		.spinner {
			display: inline-block;
			width: 14px;
			height: 14px;
			border: 2px solid #e0e0e0;
			border-top-color: #667eea;
			border-radius: 50%;
			animation: spin 0.8s linear infinite;
		}
		@keyframes spin {
			to { transform: rotate(360deg); }
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>🎯 SpellingClash</h1>
		<p class="subtitle">Server Initialization</p>
		
		<div class="progress-bar">
			<div class="progress-fill" style="width: {{.Progress}}%"></div>
		</div>
		
		<div class="progress-text">{{.Progress}}% Complete</div>
		
		<ul class="steps">
			{{range .Steps}}
			<li class="step {{if .Completed}}completed{{end}}">
				<div class="step-icon {{if .Completed}}completed{{else}}pending{{end}}">
					{{if .Completed}}✓{{else}}○{{end}}
				</div>
				<div class="step-name">{{.Name}}</div>
			</li>
			{{end}}
		</ul>
		
		<div class="current-status">
			<span class="spinner"></span> {{.Current}}
		</div>
	</div>
</body>
</html>`
	
	t, err := template.New("startup").Parse(tmpl)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, startupStatus)
}
