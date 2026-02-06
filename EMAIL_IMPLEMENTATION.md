# Email Notifications - Implementation Summary

## What Was Added

### 1. Email Service (`internal/service/email_service.go`)
- Amazon SES integration using AWS SDK v2
- Professional HTML email templates with responsive design
- Two email types implemented:
  - **Password Reset**: Sends secure reset link with 1-hour expiration
  - **Welcome Email**: Can be enabled for new user registrations
- Graceful degradation: Works without configuration (emails disabled)

### 2. Password Reset System

#### Database Schema
- New table: `password_reset_tokens`
- Fields: token, user_id, expires_at, created_at, used
- Migrations for SQLite, PostgreSQL, and MySQL
- Automatic cleanup of expired tokens (hourly)

#### Models (`internal/models/user.go`)
- `PasswordResetToken` struct with expiration checking

#### Repository Methods (`internal/repository/user_repo.go`)
- `CreatePasswordResetToken()`: Store new reset token
- `GetPasswordResetToken()`: Retrieve token with metadata
- `MarkPasswordResetTokenAsUsed()`: Prevent token reuse
- `DeleteExpiredPasswordResetTokens()`: Cleanup expired tokens
- `DeleteUserPasswordResetTokens()`: Remove all tokens for a user
- `UpdatePassword()`: Update user's password hash

#### Service Methods (`internal/service/auth_service.go`)
- `RequestPasswordReset()`: Generate token and send email
- `ValidatePasswordResetToken()`: Check token validity
- `ResetPassword()`: Complete password reset with new password
- `CleanupExpiredPasswordResetTokens()`: Background cleanup
- `generateSecureToken()`: Cryptographically secure token generation (32 bytes)

#### Handlers (`internal/handlers/auth_handler.go`)
- `ShowForgotPassword()`: Display forgot password form
- `ForgotPassword()`: Process forgot password request
- `ShowResetPassword()`: Display reset password form with token validation
- `ResetPassword()`: Process new password submission

### 3. Templates

#### `forgot_password.tmpl`
- Email input form
- Success message display
- Link back to login

#### `reset_password.tmpl`
- New password and confirmation fields
- Client-side password matching validation
- Token validation with error messages

#### Updated `login.tmpl`
- Added "Forgot Password?" link
- Success message support for post-reset notifications

### 4. Configuration

#### New Environment Variables
```bash
AWS_REGION=us-east-1                    # AWS region for SES
SES_FROM_EMAIL=noreply@yourdomain.com   # Verified sender email
SES_FROM_NAME=WordClash                 # Display name
APP_BASE_URL=https://yourdomain.com     # Base URL for links
```

#### Files Created
- `.env.example`: Complete environment variable reference
- `EMAIL_SETUP.md`: Comprehensive setup guide

### 5. Routes Added

```go
GET  /auth/forgot-password      - Show forgot password form
POST /auth/forgot-password      - Process forgot password request (rate limited)
GET  /auth/reset-password       - Show reset password form (with token validation)
POST /auth/reset-password       - Process password reset (rate limited)
```

### 6. Dependencies Added
```go
github.com/aws/aws-sdk-go-v2/config
github.com/aws/aws-sdk-go-v2/service/sesv2
```

## Security Features

1. **Secure Token Generation**
   - Cryptographically random tokens (32 bytes = 64 hex characters)
   - Stored in database with expiration timestamps

2. **Token Expiration**
   - Reset tokens expire after 1 hour
   - Automatic cleanup of expired tokens

3. **Single-Use Tokens**
   - Tokens marked as "used" after successful reset
   - Cannot be reused even if not expired

4. **Email Enumeration Prevention**
   - Always shows success message, regardless of email existence
   - Prevents attackers from discovering valid email addresses

5. **Rate Limiting**
   - Password reset requests are rate-limited
   - Prevents email bombing attacks

6. **HTTPS Recommendation**
   - Documentation emphasizes HTTPS for production
   - Prevents token interception

## How It Works

### Password Reset Flow

1. **User Requests Reset**
   - Navigates to `/login`, clicks "Forgot Password?"
   - Enters email at `/auth/forgot-password`
   
2. **Token Generation**
   - System generates cryptographically secure 64-character token
   - Deletes any existing tokens for the user
   - Stores new token in database with 1-hour expiration

3. **Email Sent**
   - Professional HTML email sent via Amazon SES
   - Contains secure reset link: `https://domain.com/auth/reset-password?token=...`
   - Plain text fallback included

4. **User Resets Password**
   - Clicks link, arrives at `/auth/reset-password?token=...`
   - System validates token (exists, not used, not expired)
   - User enters new password (minimum 8 characters)
   - Client-side validation ensures passwords match

5. **Password Updated**
   - New password validated and hashed
   - Database updated with new hash
   - Token marked as used
   - User redirected to login with success message

6. **Cleanup**
   - Background goroutine runs hourly
   - Removes expired tokens from database

## Email Templates

### HTML Structure
- Responsive design (max-width: 600px)
- Professional styling with blue accent color (#4a90e2)
- Clear call-to-action buttons
- Fallback text version included
- Mobile-friendly

### Email Content
- Clear subject lines
- Personalized greeting with user's name
- Expiration warning (1 hour)
- Security notice for non-requesters
- Plain text link as backup

## Testing

### Without Email Configuration
```bash
# Server starts normally
go run ./cmd/server

# Logs show:
# Email service disabled: SES_FROM_EMAIL not configured

# Password reset flow:
# - Forms work normally
# - Success messages displayed
# - No emails sent (logged instead)
```

### With Email Configuration
```bash
# Set environment variables
export SES_FROM_EMAIL=noreply@yourdomain.com
export AWS_REGION=us-east-1

# Server starts with email enabled
go run ./cmd/server

# Logs show:
# Email service enabled: from=noreply@yourdomain.com, region=us-east-1

# Password reset flow:
# - Email sent to user
# - Logs confirmation: "Email sent successfully: to=user@example.com"
```

## Future Enhancements

The email service is designed to support additional notifications:

1. **Welcome Emails** (already implemented)
   - Uncomment in `Register` handler to enable
   
2. **Account Activity**
   - Login from new device
   - Password changed confirmation
   - Account deletion confirmation

3. **Child Progress Reports**
   - Weekly/monthly progress summaries
   - Struggling words alerts
   - Achievement notifications

4. **List Sharing**
   - Notifications when lists are shared
   - Collaboration invites

## Files Modified

### New Files
- `internal/service/email_service.go` - SES email service
- `internal/templates/auth/forgot_password.tmpl` - Forgot password form
- `internal/templates/auth/reset_password.tmpl` - Reset password form
- `migrations/sqlite/004_password_reset.sql` - SQLite migration
- `migrations/postgres/004_password_reset.sql` - PostgreSQL migration
- `migrations/mysql/004_password_reset.sql` - MySQL migration
- `EMAIL_SETUP.md` - Detailed setup documentation
- `.env.example` - Environment variable reference

### Modified Files
- `internal/config/config.go` - Added SES configuration fields
- `internal/models/user.go` - Added PasswordResetToken model
- `internal/repository/user_repo.go` - Added password reset methods
- `internal/service/auth_service.go` - Added password reset logic
- `internal/handlers/auth_handler.go` - Added password reset handlers
- `internal/templates/auth/login.tmpl` - Added forgot password link
- `cmd/server/main.go` - Initialized email service and routes
- `README.md` - Updated features and configuration docs
- `go.mod` - Added AWS SDK dependencies

## Cost Considerations

Amazon SES is very cost-effective:
- **First 62,000 emails/month**: FREE (when sent from EC2)
- **Additional emails**: $0.10 per 1,000 emails
- **Typical small app**: Essentially free

## Production Checklist

- [ ] AWS account created
- [ ] SES sender email verified
- [ ] SES production access requested (if needed)
- [ ] AWS credentials configured
- [ ] Environment variables set
- [ ] `APP_BASE_URL` set to HTTPS domain
- [ ] DNS records configured (SPF/DKIM) for deliverability
- [ ] Test password reset flow end-to-end
- [ ] Monitor SES sending statistics
- [ ] Set up SES bounce/complaint notifications (optional)

## Support

For issues:
- **AWS/SES Configuration**: See `EMAIL_SETUP.md`
- **Application Integration**: Check server logs
- **Email Deliverability**: Configure SPF/DKIM for your domain
