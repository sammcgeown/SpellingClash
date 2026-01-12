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
- `DB_PATH` - SQLite database path (default: /app/db/spellingclash.db)
- `AUDIO_DIR` - Directory for audio files (default: /app/static/audio)

## Persistent Data

The container uses two volumes:

- `/app/db` - SQLite database
- `/app/static/audio` - Generated audio files

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
