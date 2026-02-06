# Database Backup and Restore

SpellingClash includes comprehensive database backup and restore functionality for data protection and database migration.

## Features

- **JSON Export Format**: Universal format that works across all database types (SQLite, PostgreSQL, MySQL)
- **Complete Backup**: Exports all data including users, families, kids, lists, words, and practice sessions
- **CLI Tool**: Command-line interface for automated backups
- **Web Interface**: Admin dashboard for easy backup/restore operations
- **Safe Restore**: Optional database clearing before import

## CLI Usage

### Export Database

```bash
./bin/backup export backup.json
```

Creates a complete backup of the database to `backup.json`.

### Import Database

```bash
# Import without clearing (adds to existing data)
./bin/backup import backup.json

# Import with clearing (replaces all data)
./bin/backup import backup.json --clear
```

**Warning**: The `--clear` flag will delete ALL existing data before importing.

## Web Interface Usage

1. Log in as an admin user
2. Navigate to Admin Dashboard
3. Click "Database" in the navigation menu
4. Use the Database Management page to:
   - **Export**: Download a backup file
   - **Import**: Upload a backup file
   - **View Statistics**: See database record counts

### Export via Web

1. Click "Download Backup" button
2. Save the JSON file to your local machine
3. Store safely for backup or migration purposes

### Import via Web

1. Click "Choose File" and select a backup JSON file
2. Optionally check "Clear existing data before import" to replace all data
3. Click "Import Database"
4. Confirm the operation

**Warning**: Clearing data will permanently delete all existing records.

## Backup File Format

The backup file is a JSON document containing:

```json
{
  "version": "1.0",
  "exported_at": "2024-02-06T14:30:00Z",
  "users": [...],
  "families": [...],
  "kids": [...],
  "lists": [...],
  "words": [...],
  "practices": [...]
}
```

## Database Migration

To migrate from one database type to another (e.g., SQLite to PostgreSQL):

1. **Export from source database**:
   ```bash
   ./bin/backup export source_backup.json
   ```

2. **Update configuration** to point to the target database:
   - Modify `DB_TYPE` in `.env` file
   - Update database connection settings

3. **Run migrations** on the target database:
   ```bash
   ./bin/spellingclash
   ```
   (Migrations run automatically on startup)

4. **Import to target database**:
   ```bash
   ./bin/backup import source_backup.json
   ```

## Best Practices

1. **Regular Backups**: Schedule regular exports using cron or similar
2. **Secure Storage**: Store backup files in a secure location
3. **Test Restores**: Periodically test backup restoration
4. **Version Control**: Keep multiple backup versions
5. **Pre-Migration Testing**: Test migrations on a development instance first

## Automation Example

Add to crontab for daily backups at 2 AM:

```bash
0 2 * * * cd /path/to/spellingclash && ./bin/backup export backups/backup-$(date +\%Y\%m\%d).json
```

## Troubleshooting

### Import Fails

- Verify JSON file format
- Check database permissions
- Ensure target database schema is up to date
- Review logs for specific error messages

### Missing Data After Import

- Verify backup file contains expected data
- Check if import completed successfully
- Review import logs for errors

### Performance Issues

- Large imports may take time
- Consider clearing database for faster imports
- Monitor database connection timeouts
