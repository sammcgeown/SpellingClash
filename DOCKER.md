# Docker Deployment Guide

## Building with GoReleaser

### Prerequisites

- [GoReleaser](https://goreleaser.com/install/) installed
- Docker installed and running
- GitHub account (for pushing images to GHCR)

### Local Build & Test

Build a snapshot (without pushing):

```bash
goreleaser release --snapshot --clean
```

This creates binaries and Docker images locally.

### Building Docker Images Manually

Build for your current architecture:

```bash
docker build -t spellingclash:latest .
```

Build for multiple architectures:

```bash
docker buildx build --platform linux/amd64,linux/arm64 -t spellingclash:latest .
```

## Running with Docker

### Using Docker Run

```bash
docker run -d \
  --name spellingclash \
  -p 8080:8080 \
  -v spellingclash_db:/app/db \
  -v spellingclash_audio:/app/static/audio \
  ghcr.io/your-username/spellingclash:latest
```

### Using Docker Compose

1. Update `docker-compose.yml` with your image name
2. Start the application:

```bash
docker-compose up -d
```

View logs:

```bash
docker-compose logs -f
```

Stop the application:

```bash
docker-compose down
```

## Environment Variables

- `PORT` - Server port (default: 8080)
- `DATABASE_TYPE` - Database type: `sqlite`, `postgres`, or `mysql` (default: sqlite)
- `DB_PATH` - SQLite database file path (default: /app/db/spellingclash.db)
- `DATABASE_URL` - Connection URL for PostgreSQL or MySQL
- `AUDIO_DIR` - Directory for audio files (default: /app/static/audio)

## Persistent Data

The container uses two volumes:

- `/app/db` - SQLite database (when using SQLite)
- `/app/static/audio` - Generated audio files

## Using External Databases

### SQLite (Default)

The default configuration uses SQLite with a local file:

```bash
docker run -d \
  --name spellingclash \
  -p 8080:8080 \
  -v spellingclash_db:/app/db \
  -v spellingclash_audio:/app/static/audio \
  ghcr.io/your-username/spellingclash:latest
```

### PostgreSQL

To connect to an external PostgreSQL database:

```bash
docker run -d \
  --name spellingclash \
  -p 8080:8080 \
  -e DATABASE_TYPE=postgres \
  -e DATABASE_URL="postgres://username:password@hostname:5432/dbname?sslmode=disable" \
  -v spellingclash_audio:/app/static/audio \
  ghcr.io/your-username/spellingclash:latest
```

**PostgreSQL URL format:**
```
postgres://username:password@hostname:5432/database?sslmode=disable
```

**Docker Compose with PostgreSQL:**

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
      - AUDIO_DIR=/app/static/audio
    volumes:
      - spellingclash_audio:/app/static/audio
    restart: unless-stopped

volumes:
  postgres_data:
  spellingclash_audio:
```

### MySQL

To connect to an external MySQL database:

```bash
docker run -d \
  --name spellingclash \
  -p 8080:8080 \
  -e DATABASE_TYPE=mysql \
  -e DATABASE_URL="username:password@tcp(hostname:3306)/dbname?parseTime=true" \
  -v spellingclash_audio:/app/static/audio \
  ghcr.io/your-username/spellingclash:latest
```

**MySQL URL format:**
```
username:password@tcp(hostname:3306)/database?parseTime=true
```

**Docker Compose with MySQL:**

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
      - AUDIO_DIR=/app/static/audio
    volumes:
      - spellingclash_audio:/app/static/audio
    restart: unless-stopped

volumes:
  mysql_data:
  spellingclash_audio:
```

### Using External Database Hosts

To connect to databases running outside Docker (e.g., managed cloud databases):

**PostgreSQL (AWS RDS, Google Cloud SQL, etc.):**
```bash
docker run -d \
  --name spellingclash \
  -p 8080:8080 \
  -e DATABASE_TYPE=postgres \
  -e DATABASE_URL="postgres://user:pass@your-db.region.rds.amazonaws.com:5432/spellingclash?sslmode=require" \
  -v spellingclash_audio:/app/static/audio \
  ghcr.io/your-username/spellingclash:latest
```

**MySQL (AWS RDS, DigitalOcean, etc.):**
```bash
docker run -d \
  --name spellingclash \
  -p 8080:8080 \
  -e DATABASE_TYPE=mysql \
  -e DATABASE_URL="user:pass@tcp(your-db.region.rds.amazonaws.com:3306)/spellingclash?parseTime=true&tls=skip-verify" \
  -v spellingclash_audio:/app/static/audio \
  ghcr.io/your-username/spellingclash:latest
```

## Releasing

### Automated Release (GitHub Actions)

1. Tag a new version:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

2. GitHub Actions will automatically:
   - Build binaries for Linux (amd64 and arm64)
   - Build Docker images for both architectures
   - Push images to GitHub Container Registry
   - Create a GitHub Release with artifacts

### Manual Release

```bash
export GITHUB_TOKEN=your_github_token
export GITHUB_REPOSITORY_OWNER=your-username
goreleaser release --clean
```

## Accessing the Application

Once running, access SpellingClash at:

- Web UI: http://localhost:8080
- Health check: http://localhost:8080/

## Troubleshooting

### View logs

```bash
docker logs spellingclash
```

### Access container shell

```bash
docker exec -it spellingclash sh
```

### Rebuild database

```bash
docker exec -it spellingclash rm /app/db/spellingclash.db
docker restart spellingclash
```

### Check volumes

```bash
docker volume ls
docker volume inspect spellingclash_db
docker volume inspect spellingclash_audio
```
