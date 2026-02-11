---
description: 'Project-specific instructions for this Go web application'
applyTo: '**'
---

# Project-Specific Instructions

## Local Development

- Start the application locally with `go run ./cmd/server`

## Frontend & Templating

- Everything must use the same template, styling, and structure as the rest of the project. Match existing patterns before introducing new ones.
- Forms must use HTMX for dynamic behavior instead of custom JavaScript wherever possible.
- When writing HTML templates, ensure proper escaping to prevent XSS vulnerabilities. Use Go's `html/template` package, not `text/template`, for any user-facing output.
- Avoid in-line styles or scripts in templates. Keep styling in CSS files and behavior in Go handlers or HTMX attributes.

## Database

- Use prepared statements or ORM methods for all database queries to prevent SQL injection. Never interpolate user input directly into query strings.

## Docker

- When writing Dockerfiles or Docker-related configuration, follow best practices for security and efficiency (multi-stage builds, non-root users, minimal base images, pinned versions).

## Context Usage

- When writing Go code, ensure proper use of `context.Context` for request handling and cancellation. Pass contexts through the call chain rather than storing them in structs.
