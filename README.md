# SpellingClash

A web-based spelling practice application for kids, built with Go and HTMX.

## Features

- **Parent Dashboard**: Manage kids, spelling lists, and track progress
- **Kid Practice Mode**: Interactive spelling practice with audio pronunciation
- **Multiple Game Modes**: Standard practice, Hangman, and Missing Letter games
- **Family System**: Share kids and lists within a family group
- **Public Lists**: Pre-built spelling lists for different year groups
- **OAuth Login**: Sign in with Google, Facebook, or Apple
- **Invite-Only Registration**: Optional invite-only mode with email invitations
- **Email Notifications**: Password reset and account recovery via Amazon SES
- **Database Backup/Restore**: Export and import data for backup and migration
- **Multi-Database Support**: SQLite, PostgreSQL, and MySQL

## Quick Start

### Local Development

```bash
# Clone the repository
git clone https://github.com/your-username/spellingclash.git
cd spellingclash

# Run the server
go run ./cmd/server

# Access at http://localhost:8080
```

### Default Admin Credentials

⚠️ **Change these in production!**

- **Email**: `admin@spellingclash.local`
- **Password**: `admin123`

---

## Table of Contents

- [Configuration](#configuration)
- [Authentication](#authentication)
- [Invite-Only Registration](#invite-only-registration)
- [Admin System](#admin-system)
- [Database Backup](#database-backup)
- [Docker Deployment](#docker-deployment)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)

---

## Configuration

All configuration is done via environment variables:

### Core Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DATABASE_TYPE` | `sqlite` | Database type: `sqlite`, `postgres`, or `mysql` |
| `DB_PATH` | `./spellingclash.db` | SQLite database file path |
| `DATABASE_URL` | - | Connection URL for PostgreSQL or MySQL |
| `STATIC_PATH` | `./static` | Static files directory |
| `TEMPLATES_PATH` | `./internal/templates` | Templates directory |
| `MIGRATIONS_PATH` | `./migrations` | Migrations directory |

### OAuth Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `OAUTH_REDIRECT_BASE_URL` | - | Base URL for OAuth callbacks (e.g., `https://your-domain.com`) |
| `GOOGLE_CLIENT_ID` | - | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | - | Google OAuth client secret |
| `FACEBOOK_CLIENT_ID` | - | Facebook OAuth app ID |
| `FACEBOOK_CLIENT_SECRET` | - | Facebook OAuth app secret |
| `APPLE_CLIENT_ID` | - | Apple Sign In service ID |
| `APPLE_CLIENT_SECRET` | - | Apple Sign In client secret (JWT) |

### Email Settings (Amazon SES)

| Variable | Default | Description |
|----------|---------|-------------|
| `AWS_REGION` | `us-east-1` | AWS region for SES |
| `SES_FROM_EMAIL` | - | Verified sender email address (required for email features) |
| `SES_FROM_NAME` | `WordClash` | Display name for outgoing emails |
| `APP_BASE_URL` | `http://localhost:8080` | Base URL for password reset links |

**Note**: Email notifications (password reset) will be disabled if `SES_FROM_EMAIL` is not configured. See [EMAIL_SETUP.md](EMAIL_SETUP.md) for detailed setup instructions.

---

## Authentication

SpellingClash supports multiple authentication methods:

### Email/Password Authentication

Standard email and password registration/login at `/login` and `/register`. Includes password reset functionality via email.

### Password Reset

Users can reset their password by clicking "Forgot Password?" on the login page. A secure reset link will be emailed (requires SES configuration). Reset links expire after 1 hour.

Standard email and password registration/login at `/login` and `/register`.

### OAuth Authentication (Social Login)

Users can sign in with Google, Facebook, or Apple. OAuth buttons appear automatically on login and registration pages when provider credentials are configured.

#### Setting Up OAuth Providers

##### Google

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Navigate to **APIs & Services > Credentials**
4. Click **Create Credentials > OAuth client ID**
5. Select **Web application**
6. Add authorized redirect URI: `https://your-domain.com/auth/google/callback`
7. Copy the Client ID and Client Secret

```bash
export GOOGLE_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export GOOGLE_CLIENT_SECRET="your-client-secret"
```

##### Facebook

1. Go to [Facebook Developers](https://developers.facebook.com/)
2. Create a new app (Consumer type)
3. Add **Facebook Login** product
4. In Settings > Basic, get App ID and App Secret
5. Add OAuth redirect URI: `https://your-domain.com/auth/facebook/callback`

```bash
export FACEBOOK_CLIENT_ID="your-app-id"
export FACEBOOK_CLIENT_SECRET="your-app-secret"
```

##### Apple

1. Go to [Apple Developer](https://developer.apple.com/)
2. Create an App ID with Sign In with Apple capability
3. Create a Services ID for web authentication
4. Configure the return URL: `https://your-domain.com/auth/apple/callback`
5. Create a private key for Sign In with Apple
6. Generate the client secret JWT using the private key

```bash
export APPLE_CLIENT_ID="com.your-domain.spellingclash"
export APPLE_CLIENT_SECRET="your-jwt-client-secret"
```

#### OAuth Callback URLs

| Provider | Callback URL |
|----------|-------------|
| Google | `https://your-domain.com/auth/google/callback` |
| Facebook | `https://your-domain.com/auth/facebook/callback` |
| Apple | `https://your-domain.com/auth/apple/callback` |

#### OAuth User Flow

1. User clicks an OAuth provider button on login/register page
2. User is redirected to the provider's authorization page
3. After authorization, user is redirected back to the callback URL
4. SpellingClash creates a new account or links to existing account
5. User is logged in and redirected to the parent dashboard

#### Family Code with OAuth

When registering via OAuth with a `family_code` query parameter (e.g., `/register?family_code=ABC123`), the new user will automatically join the specified family.

---

## Invite-Only Registration

SpellingClash supports an optional invite-only registration mode, restricting new user signups to those with valid invitation codes.

### Overview

When invite-only mode is enabled:
- The "Create Account" link is hidden from the login page
- Registration requires a valid invitation code
- Users must access registration via an invitation link (e.g., `/register?invite=CODE`)
- Invitation codes expire after 7 days
- Each invitation can only be used once

### Admin Interface

Admins can manage invitations at `/admin/invitations`:

#### Toggle Invite-Only Mode

Click **Enable Invite-Only** or **Disable Invite-Only** to toggle registration mode. When disabled, registration is open to everyone (default behavior).

#### Send Invitations

1. Enter recipient's email address
2. Click **Send Invitation**
3. An email with a registration link will be sent (requires SES configuration)
4. The invitation code is generated automatically

#### View Invitations

The invitations table shows:
- **Email**: Recipient's email address
- **Code**: 32-character invitation code
- **Status**: Active (unused), Used (already registered), or Expired
- **Expires**: Expiration date (7 days from creation)
- **Created**: When the invitation was created

#### Delete Invitations

Click the 🗑️ button to remove unused or expired invitations.

### Invitation Flow

1. Admin sends invitation to `user@example.com`
2. User receives email with link: `https://your-domain.com/register?invite=abc123...`
3. User clicks link and completes registration
4. Invitation is marked as "Used" and cannot be reused
5. User can now log in normally

### Security Features

- **Cryptographically secure codes**: 32-character random hex strings
- **Expiration**: All invitations expire after 7 days
- **One-time use**: Invitations are invalidated after registration
- **Admin-only**: Only admins can send and manage invitations
- **Email validation**: Invitation codes are stored with the intended recipient's email

### Database Schema

Invitations are stored in the `invitations` table:
```sql
CREATE TABLE invitations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT UNIQUE NOT NULL,
    email TEXT NOT NULL,
    created_by_id INTEGER NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT 0,
    used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

The `settings` table stores the invite-only mode status:
```sql
CREATE TABLE settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT UNIQUE NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/invitations` | GET | Show invitations management page |
| `/admin/invitations/toggle` | POST | Toggle invite-only mode on/off |
| `/admin/invitations/send` | POST | Send new invitation email |
| `/admin/invitations/{id}` | POST | Delete invitation |
| `/register?invite=CODE` | GET | Register with invitation code |

---

## Admin System

### Overview

The admin system manages public spelling lists. The system admin user is created automatically during database initialization.

### Accessing the Admin Dashboard

1. Log in at `/login` using admin credentials
2. Navigate to `/admin/dashboard`

### Admin Features

#### Public Lists Management

The admin dashboard displays all public spelling lists. These are default lists available to all users.

#### Regenerate Public Lists

The admin can regenerate all public lists from source data files:

1. Go to `/admin/dashboard`
2. Click **Regenerate Public Lists**
3. Confirm the action

⚠️ **Warning**: This deletes all existing public lists and their assignments!

### Security

#### Admin Middleware

Admin routes are protected by `RequireAdmin` middleware which:
- Validates the user session
- Checks `is_admin = true` in the database
- Returns 403 Forbidden for non-admin users

#### Changing the Admin Password

Generate a bcrypt hash and update the database:

```bash
# SQLite
sqlite3 spellingclash.db "UPDATE users SET password_hash='NEW_HASH' WHERE email='admin@spellingclash.local'"

# PostgreSQL
psql -d spellingclash -c "UPDATE users SET password_hash='NEW_HASH' WHERE email='admin@spellingclash.local'"

# MySQL
mysql -D spellingclash -e "UPDATE users SET password_hash='NEW_HASH' WHERE email='admin@spellingclash.local'"
```

#### Creating Additional Admins

```sql
UPDATE users SET is_admin = 1 WHERE email = 'user@example.com';
```

---

## Database Backup

SpellingClash includes comprehensive backup and restore functionality for data protection and database migration.

### Quick Start

**Export database:**
```bash
./bin/backup export backup.json
```

**Import database:**
```bash
./bin/backup import backup.json
```

**Import with database clear:**
```bash
./bin/backup import backup.json --clear
```

### Web Interface

1. Log in as admin
2. Navigate to **Admin Dashboard → Database**
3. Use the web interface to:
   - Download backup files
   - Upload and restore backups
   - View database statistics

### Database Migration

To migrate between database types (e.g., SQLite → PostgreSQL):

1. Export from source: `./bin/backup export source.json`
2. Update `DB_TYPE` in `.env` to target database
3. Run server to create schema: `go run ./cmd/server`
4. Import to target: `./bin/backup import source.json`

### Backup Format

Backups are stored as JSON files containing all users, families, kids, lists, words, and practice sessions. The format is universal and works across SQLite, PostgreSQL, and MySQL.

**For detailed documentation**, see [DATABASE_BACKUP.md](DATABASE_BACKUP.md)

---

## Docker Deployment

### Building Docker Images

```bash
# Build for current architecture
docker build -t spellingclash:latest .

# Build for multiple architectures
docker buildx build --platform linux/amd64,linux/arm64 -t spellingclash:latest .
```

### Running with Docker

#### Basic Run (SQLite)

```bash
docker run -d \
  --name spellingclash \
  -p 8080:8080 \
  -v spellingclash_db:/app/db \
  -v spellingclash_audio:/app/static/audio \
  ghcr.io/your-username/spellingclash:latest
```

#### With OAuth Providers

```bash
docker run -d \
  --name spellingclash \
  -p 8080:8080 \
  -e OAUTH_REDIRECT_BASE_URL="https://your-domain.com" \
  -e GOOGLE_CLIENT_ID="your-client-id" \
  -e GOOGLE_CLIENT_SECRET="your-client-secret" \
  -v spellingclash_db:/app/db \
  -v spellingclash_audio:/app/static/audio \
  ghcr.io/your-username/spellingclash:latest
```

### Docker Compose

#### SQLite (Default)

```yaml
version: '3.8'

services:
  spellingclash:
    image: ghcr.io/your-username/spellingclash:latest
    container_name: spellingclash
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - OAUTH_REDIRECT_BASE_URL=https://your-domain.com
      - GOOGLE_CLIENT_ID=your-client-id
      - GOOGLE_CLIENT_SECRET=your-client-secret
    volumes:
      - spellingclash_db:/app/db
      - spellingclash_audio:/app/static/audio
    restart: unless-stopped

volumes:
  spellingclash_db:
  spellingclash_audio:
```

#### PostgreSQL

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: spellingclash
      POSTGRES_USER: spellingclash
      POSTGRES_PASSWORD: your_secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  spellingclash:
    image: ghcr.io/your-username/spellingclash:latest
    container_name: spellingclash
    depends_on:
      - postgres
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - DATABASE_TYPE=postgres
      - DATABASE_URL=postgres://spellingclash:your_secure_password@postgres:5432/spellingclash?sslmode=disable
      - OAUTH_REDIRECT_BASE_URL=https://your-domain.com
    volumes:
      - spellingclash_audio:/app/static/audio
    restart: unless-stopped

volumes:
  postgres_data:
  spellingclash_audio:
```

#### MySQL

```yaml
version: '3.8'

services:
  mysql:
    image: mysql:8
    environment:
      MYSQL_DATABASE: spellingclash
      MYSQL_USER: spellingclash
      MYSQL_PASSWORD: your_secure_password
      MYSQL_ROOT_PASSWORD: root_password
    volumes:
      - mysql_data:/var/lib/mysql
    restart: unless-stopped

  spellingclash:
    image: ghcr.io/your-username/spellingclash:latest
    container_name: spellingclash
    depends_on:
      - mysql
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - DATABASE_TYPE=mysql
      - DATABASE_URL=spellingclash:your_secure_password@tcp(mysql:3306)/spellingclash?parseTime=true
    volumes:
      - spellingclash_audio:/app/static/audio
    restart: unless-stopped

volumes:
  mysql_data:
  spellingclash_audio:
```

### Persistent Data

| Volume | Purpose |
|--------|---------|
| `/app/db` | SQLite database |
| `/app/static/audio` | Generated audio files |

### External Database Connections

**PostgreSQL (AWS RDS, Google Cloud SQL, etc.):**
```bash
docker run -d \
  -e DATABASE_TYPE=postgres \
  -e DATABASE_URL="postgres://user:pass@your-db.region.rds.amazonaws.com:5432/spellingclash?sslmode=require" \
  ghcr.io/your-username/spellingclash:latest
```

**MySQL (AWS RDS, DigitalOcean, etc.):**
```bash
docker run -d \
  -e DATABASE_TYPE=mysql \
  -e DATABASE_URL="user:pass@tcp(your-db.region.rds.amazonaws.com:3306)/spellingclash?parseTime=true&tls=skip-verify" \
  ghcr.io/your-username/spellingclash:latest
```

### Releasing

#### Automated (GitHub Actions)

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

GitHub Actions will build binaries, Docker images, and create a release.

---

## Testing

### Running Tests

```bash
# Quick test run
go test ./...

# With coverage report
./test.sh

# Specific package
go test ./internal/service

# Specific test
go test ./internal/utils -run TestHashPassword

# Skip integration tests
go test ./... -short
```

### Test Structure

| Package | Description |
|---------|-------------|
| `internal/service` | Business logic and data processing |
| `internal/utils` | Password handling, validation, sanitization |
| `internal/handlers` | Game logic and mechanics |
| `internal/models` | Data validation and calculations |
| `internal/database` | Dialect abstraction and integration |

### Test Coverage Goals

| Component | Target |
|-----------|--------|
| Services | 80% |
| Utilities | 90% |
| Handlers | 70% |
| Models | 85% |
| Database | 75% |

### Writing Tests

#### Table-Driven Test Example

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "TEST", false},
        {"empty string", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := MyFunction(tt.input)
            if tt.wantErr && err == nil {
                t.Error("Expected error, got nil")
            }
            if result != tt.expected {
                t.Errorf("Expected %s, got %s", tt.expected, result)
            }
        })
    }
}
```

### Best Practices

**Do:**
- Use table-driven tests for multiple scenarios
- Test edge cases and error conditions
- Use descriptive test names
- Clean up resources with `defer` or `t.Cleanup()`
- Use `testing.Short()` for integration tests

**Don't:**
- Test implementation details
- Write tests that depend on execution order
- Use real databases without cleanup
- Ignore test failures

---

## Troubleshooting

### Docker Issues

```bash
# View logs
docker logs spellingclash

# Access container shell
docker exec -it spellingclash sh

# Rebuild database
docker exec -it spellingclash rm /app/db/spellingclash.db
docker restart spellingclash

# Check volumes
docker volume ls
docker volume inspect spellingclash_db
```

### OAuth Issues

**Buttons not appearing:**
- Ensure OAuth environment variables are set
- Verify `OAUTH_REDIRECT_BASE_URL` is configured
- Check server logs for configuration errors

**Callback errors:**
- Verify callback URLs match exactly in provider console
- Ensure HTTPS is used in production
- Check that client ID and secret are correct

### Admin Issues

**Cannot access admin dashboard:**
1. Verify logged in with admin credentials
2. Check `is_admin = true` in database:
   ```bash
   sqlite3 spellingclash.db "SELECT email, is_admin FROM users WHERE email = 'admin@spellingclash.local';"
   ```
3. Check server logs for authentication errors

**"FOREIGN KEY constraint failed" when seeding lists:**
- Ensure admin user (ID = 1) exists
- Run migrations again if needed

### Test Issues

```bash
# Verbose output
go test -v ./...

# Check coverage
go test -coverprofile=coverage.out ./internal/service
go tool cover -html=coverage.out

# Run only integration tests
go test ./... -run Integration
```

---

## License

See [LICENSE](LICENSE) for details.
