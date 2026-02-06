# Version Management

The application version is displayed in the admin dashboard header.

## Setting the Version

### Local Development
When running locally without setting a version, it defaults to "dev":
```bash
go run ./cmd/server
```

### Building with a Version
Set the version during build using ldflags:
```bash
go build -ldflags="-X main.Version=1.0.0" -o bin/spellingclash ./cmd/server
```

### Docker Build
When building the Docker image, pass the VERSION build argument:
```bash
# Using a semantic version
docker build --build-arg VERSION=1.0.0 -t spellingclash:1.0.0 .

# Using a git commit hash
docker build --build-arg VERSION=$(git rev-parse --short HEAD) -t spellingclash:latest .

# Using a git tag
docker build --build-arg VERSION=$(git describe --tags --always) -t spellingclash:latest .
```

### Kubernetes/Production
When deploying, ensure your build pipeline sets the VERSION:
```bash
VERSION=$(git describe --tags --always)
docker build --build-arg VERSION=$VERSION -t your-registry/spellingclash:$VERSION .
docker push your-registry/spellingclash:$VERSION
```

## Viewing the Version
The version is displayed in the admin dashboard header, right below "SpellingClash Admin".
