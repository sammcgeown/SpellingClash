# Admin System Documentation

## Overview

SpellingClash includes an admin system for managing public spelling lists. The system admin user is created automatically during database initialization and has access to the admin dashboard.

## Default Admin Credentials

**⚠️ IMPORTANT: Change these credentials in production!**

- **Email**: `admin@spellingclash.local`
- **Password**: `admin123`

## Accessing the Admin Dashboard

1. Log in at `/login` using the admin credentials
2. Navigate to `/admin/dashboard`

## Admin Features

### Public Lists Management

The admin dashboard displays all public spelling lists in the system. These lists are used as default lists that all users can access.

### Regenerate Public Lists

The admin can regenerate all public lists from the source data files. This will:

1. Delete all existing public lists
2. Re-import lists from the JSON data files in `/data`
3. Regenerate audio files for all words

**To regenerate lists:**
1. Go to `/admin/dashboard`
2. Click the "Regenerate Public Lists" button
3. Confirm the action

⚠️ **Warning**: This action will delete all existing public lists and their assignments. Use with caution!

## Security

### Admin Middleware

Admin routes are protected by the `RequireAdmin` middleware, which:
- Validates the user session
- Checks that the user has `is_admin = true` in the database
- Returns 403 Forbidden if the user is not an admin

### Changing the Admin Password

**Method 1: Through the Application (Recommended)**

Currently, there is no built-in UI for changing the admin password. You can use the standard password reset flow or add a profile page.

**Method 2: Direct Database Update**

Generate a bcrypt hash and update the database:

```bash
# Generate a new bcrypt hash (using Go or an online tool)
# Example in Go:
# hash, _ := bcrypt.GenerateFromPassword([]byte("newpassword"), bcrypt.DefaultCost)

# Update SQLite
sqlite3 db/app.db "UPDATE users SET password_hash='$2a$10$...' WHERE email='admin@spellingclash.local'"

# Update PostgreSQL
psql -d spellingclash -c "UPDATE users SET password_hash='$2a$10$...' WHERE email='admin@spellingclash.local'"

# Update MySQL
mysql -D spellingclash -e "UPDATE users SET password_hash='$2a$10$...' WHERE email='admin@spellingclash.local'"
```

### Creating Additional Admins

To grant admin access to an existing user:

```sql
UPDATE users SET is_admin = 1 WHERE email = 'user@example.com';
```

## Technical Details

### Database Schema

The admin system adds the `is_admin` boolean field to the `users` table:

```sql
is_admin BOOLEAN DEFAULT 0
```

### System Admin User

The system admin (ID = 1) is created during database migrations:

```sql
INSERT OR IGNORE INTO users (id, email, password_hash, name, is_admin, created_at, updated_at) 
VALUES (
    1, 
    'admin@spellingclash.local', 
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    'Admin',
    1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
);
```

This user is referenced by public lists (`created_by = 1`) to satisfy foreign key constraints.

### Routes

- `GET /admin/dashboard` - Admin dashboard showing public lists
- `POST /admin/regenerate-lists` - Regenerate all public lists (CSRF protected)

### Kubernetes Deployment

In Kubernetes environments, the database is persisted using a PersistentVolumeClaim, so the admin user will be available immediately after the first deployment. Ensure the default password is changed before exposing the application to users.

## Troubleshooting

### "FOREIGN KEY constraint failed" when seeding lists

This error occurs if the system admin user (ID = 1) doesn't exist. Ensure database migrations have run successfully:

```bash
# Check if admin user exists
sqlite3 db/app.db "SELECT id, email, is_admin FROM users WHERE id = 1;"
```

If the user doesn't exist, run migrations again or manually insert the admin user.

### Cannot access admin dashboard

1. Verify you're logged in with admin credentials
2. Check that the user has `is_admin = true`:
   ```bash
   sqlite3 db/app.db "SELECT email, is_admin FROM users WHERE email = 'admin@spellingclash.local';"
   ```
3. Check server logs for authentication errors
