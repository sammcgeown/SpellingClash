# SpellingClash Kubernetes Deployment with External PostgreSQL

This manifest deploys SpellingClash to Kubernetes using an external PostgreSQL database.

## Prerequisites

1. **External PostgreSQL Database** - Set up and accessible from your Kubernetes cluster
2. **Kubernetes Cluster** with:
   - Longhorn (or another storage class for audio files)
   - Traefik or another Ingress controller
   - cert-manager for TLS certificates (optional)
3. **Container Registry Access** - GitHub Container Registry credentials

## Configuration Steps

### 1. Update Database Connection

Edit the `spellingclash-db-secret` in `spellingclash-postgres.yaml`:

```yaml
stringData:
  DATABASE_URL: "postgres://username:password@hostname:5432/database?sslmode=require"
```

**Format**: `postgres://USER:PASSWORD@HOST:PORT/DATABASE?sslmode=require`

**Example**: `postgres://spellingclash:myP@ssw0rd@postgres.example.com:5432/spellingclash?sslmode=require`

### 2. Update Application URL

Edit the `spellingclash-config` ConfigMap:

```yaml
data:
  APP_BASE_URL: "https://your-domain.com"
  OAUTH_REDIRECT_BASE_URL: "https://your-domain.com"
```

And update the Ingress hostname:

```yaml
spec:
  rules:
    - host: your-domain.com
  tls:
    - hosts:
        - your-domain.com
```

### 3. Configure AWS SES (Optional)

If using email notifications, update `spellingclash-aws-secret`:

```yaml
stringData:
  AWS_ACCESS_KEY_ID: "AKIA..."
  AWS_SECRET_ACCESS_KEY: "wJalr..."
  SES_FROM_EMAIL: "noreply@your-domain.com"
```

Add to ConfigMap:
```yaml
data:
  AWS_REGION: "us-east-1"  # or your region
  SES_FROM_NAME: "SpellingClash"
```

### 4. Configure OAuth (Optional)

Update `spellingclash-oauth-secret`:

```yaml
stringData:
  GOOGLE_CLIENT_ID: "your-client-id"
  GOOGLE_CLIENT_SECRET: "your-client-secret"
  FACEBOOK_CLIENT_ID: "your-app-id"
  FACEBOOK_CLIENT_SECRET: "your-app-secret"
```

### 5. Create Container Registry Secret

```bash
kubectl create secret docker-registry ghcr-credentials \
  --namespace=spellingclash \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USERNAME \
  --docker-password=YOUR_GITHUB_PAT
```

## Deployment

### Apply the manifest:

```bash
kubectl apply -f kubernetes/spellingclash-postgres.yaml
```

### Verify deployment:

```bash
# Check pods
kubectl get pods -n spellingclash

# Check logs
kubectl logs -n spellingclash -l app=spellingclash --tail=100 -f

# Check service
kubectl get svc -n spellingclash

# Check ingress
kubectl get ingress -n spellingclash
```

## Database Setup

### Option 1: Managed PostgreSQL

Use a managed PostgreSQL service:
- **AWS RDS**: Amazon RDS for PostgreSQL
- **Google Cloud SQL**: Cloud SQL for PostgreSQL
- **Azure Database**: Azure Database for PostgreSQL
- **DigitalOcean**: Managed PostgreSQL
- **Neon**: Serverless PostgreSQL

### Option 2: Self-Hosted PostgreSQL

If running PostgreSQL in the same Kubernetes cluster, see `spellingclash-with-postgres.yaml` for a complete example with StatefulSet.

### Initialize the Database

The application will automatically:
1. Run migrations on startup
2. Create all required tables
3. Seed default public spelling lists
4. Generate audio files (may take several minutes on first run)

## Scaling

The manifest includes HorizontalPodAutoscaler configured for:
- **Min replicas**: 2
- **Max replicas**: 10
- **CPU trigger**: 70% utilization
- **Memory trigger**: 80% utilization

Adjust as needed:

```yaml
spec:
  minReplicas: 2
  maxReplicas: 10
```

## Storage

### Audio Files

Audio files are stored in a PersistentVolume (5GB by default):
- Shared across all pods using ReadWriteMany
- Generated once and reused
- Survives pod restarts

To increase storage:

```yaml
spec:
  resources:
    requests:
      storage: 10Gi  # Increase as needed
```

### Database

With external PostgreSQL:
- No database PVC needed
- Database managed separately
- Can use database backups/snapshots

## Monitoring

### Check Application Status

```bash
# Watch pods
kubectl get pods -n spellingclash -w

# View logs
kubectl logs -n spellingclash deployment/spellingclash -f

# Describe pod for events
kubectl describe pod -n spellingclash -l app=spellingclash
```

### Check Database Connection

```bash
# Shell into pod
kubectl exec -it -n spellingclash deployment/spellingclash -- sh

# Test database connection (if psql is available)
psql $DATABASE_URL -c "SELECT version();"
```

## Troubleshooting

### Pods not starting

```bash
kubectl describe pod -n spellingclash -l app=spellingclash
kubectl logs -n spellingclash -l app=spellingclash --previous
```

### Database connection issues

1. Verify DATABASE_URL format
2. Check network connectivity from pod to database
3. Verify PostgreSQL allows connections from cluster
4. Check PostgreSQL logs

### Audio generation taking too long

First startup may take 10+ minutes to generate audio files. Check:

```bash
kubectl logs -n spellingclash -l app=spellingclash --tail=50 -f
```

Look for: "Generating audio files (this may take a while)..."

### Out of memory

Increase resource limits:

```yaml
resources:
  limits:
    memory: "2Gi"
```

## Backup & Restore

### Export Database

```bash
# Port-forward to access admin panel
kubectl port-forward -n spellingclash svc/spellingclash 8080:80

# Navigate to: http://localhost:8080/admin/database
# Login as admin and download backup
```

Or use the CLI tool:

```bash
kubectl exec -n spellingclash deployment/spellingclash -- /app/backup export -output /tmp/backup.json
kubectl cp spellingclash/POD_NAME:/tmp/backup.json ./backup.json
```

### Restore Database

```bash
kubectl cp ./backup.json spellingclash/POD_NAME:/tmp/backup.json
kubectl exec -n spellingclash deployment/spellingclash -- /app/backup import -input /tmp/backup.json
```

## Security Recommendations

1. **Use Secrets for all sensitive data** - Never put passwords in ConfigMaps
2. **Enable SSL/TLS** for database connections (`sslmode=require`)
3. **Rotate credentials regularly** - Use Kubernetes secret rotation tools
4. **Network Policies** - Restrict pod-to-pod and external access
5. **Resource Limits** - Prevent resource exhaustion
6. **RBAC** - Use service accounts with minimal permissions

## Performance Tuning

### For High Traffic

```yaml
spec:
  replicas: 5  # Start with more replicas
  resources:
    requests:
      memory: "1Gi"
      cpu: "500m"
    limits:
      memory: "2Gi"
      cpu: "2000m"
```

### Database Connection Pooling

Consider using PgBouncer between the app and PostgreSQL:
- Reduces connection overhead
- Better resource utilization
- Faster response times

## Migration from SQLite

If migrating from the SQLite deployment:

1. Export data from SQLite deployment:
   ```bash
   kubectl exec -n spellingclash OLD_POD -- /app/backup export -output /tmp/backup.json
   kubectl cp spellingclash/OLD_POD:/tmp/backup.json ./backup.json
   ```

2. Deploy PostgreSQL version

3. Import data:
   ```bash
   kubectl cp ./backup.json spellingclash/NEW_POD:/tmp/backup.json
   kubectl exec -n spellingclash NEW_POD -- /app/backup import -input /tmp/backup.json -clear
   ```

4. Copy audio files:
   ```bash
   kubectl cp spellingclash/OLD_POD:/app/static/audio ./audio
   kubectl cp ./audio spellingclash/NEW_POD:/app/static/audio/
   ```
