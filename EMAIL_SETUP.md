# Email Notification Setup Guide

WordClash now supports email notifications using Amazon SES (Simple Email Service) for password resets and other future notifications.

## Features

- **Password Reset**: Users can request password reset links via email
- **Welcome Emails**: Optional welcome emails for new user registrations (implemented but not enabled by default)
- **Secure Tokens**: Cryptographically secure reset tokens that expire after 1 hour
- **Beautiful HTML Emails**: Professional responsive HTML email templates

## Prerequisites

1. **AWS Account**: You need an AWS account with SES access
2. **Verified Email**: Your sending email address must be verified in SES
3. **AWS Credentials**: Properly configured AWS credentials

## SES Setup

### Step 1: Create/Configure AWS Account

1. Sign up for AWS at https://aws.amazon.com if you don't have an account
2. Navigate to Amazon SES in the AWS Console

### Step 2: Verify Your Email Address

1. Go to SES > Verified Identities
2. Click "Create identity"
3. Select "Email address"
4. Enter your sending email (e.g., noreply@yourdomain.com)
5. Click "Create identity"
6. Check your email and click the verification link

### Step 3: Move Out of SES Sandbox (Production)

By default, SES is in sandbox mode which only allows sending to verified addresses.

**For production:**
1. Go to SES > Account dashboard
2. Click "Request production access"
3. Fill out the form explaining your use case
4. AWS typically approves within 24 hours

**For testing:**
- Verify the recipient email addresses you want to test with
- Follow the same process as Step 2 for each test recipient

### Step 4: Configure AWS Credentials

#### Option A: AWS Credentials File (Recommended for Local Development)

Create `~/.aws/credentials`:
```ini
[default]
aws_access_key_id = YOUR_ACCESS_KEY_ID
aws_secret_access_key = YOUR_SECRET_ACCESS_KEY
```

Create `~/.aws/config`:
```ini
[default]
region = us-east-1
```

#### Option B: Environment Variables (Recommended for Production)

```bash
export AWS_ACCESS_KEY_ID=YOUR_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY=YOUR_SECRET_ACCESS_KEY
export AWS_REGION=us-east-1
```

#### Option C: IAM Role (Recommended for EC2/ECS)

If running on AWS infrastructure, attach an IAM role with SES permissions to your instance/task.

Required IAM permissions:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ses:SendEmail",
        "ses:SendRawEmail"
      ],
      "Resource": "*"
    }
  ]
}
```

## Application Configuration

### Environment Variables

Add these to your `.env` or environment:

```bash
# Email Settings (Amazon SES)
AWS_REGION=us-east-1                    # AWS region where SES is configured
SES_FROM_EMAIL=noreply@yourdomain.com   # Verified sender email address
SES_FROM_NAME=WordClash                 # Display name for emails
APP_BASE_URL=https://yourdomain.com     # Base URL for reset links
```

**Important Notes:**
- `SES_FROM_EMAIL` must be verified in SES
- `APP_BASE_URL` should be your production domain (or http://localhost:8080 for local testing)
- If `SES_FROM_EMAIL` is not set, the email service will be disabled and the app will work without emails

### Testing Locally

For local development:

```bash
# .env file
AWS_REGION=us-east-1
SES_FROM_EMAIL=your-verified-email@example.com
SES_FROM_NAME=WordClash Dev
APP_BASE_URL=http://localhost:8080
```

### Docker Configuration

When using Docker, pass environment variables:

```bash
docker run -e AWS_REGION=us-east-1 \
  -e SES_FROM_EMAIL=noreply@yourdomain.com \
  -e SES_FROM_NAME=WordClash \
  -e APP_BASE_URL=https://yourdomain.com \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  your-image
```

Or use Docker Compose:

```yaml
environment:
  AWS_REGION: us-east-1
  SES_FROM_EMAIL: noreply@yourdomain.com
  SES_FROM_NAME: WordClash
  APP_BASE_URL: https://yourdomain.com
  AWS_ACCESS_KEY_ID: your_key
  AWS_SECRET_ACCESS_KEY: your_secret
```

## Password Reset Flow

### User Experience

1. User clicks "Forgot Password?" on login page
2. User enters their email address
3. User receives email with reset link (expires in 1 hour)
4. User clicks link and enters new password
5. User is redirected to login with success message

### For Administrators

**Database Table:**
Password reset tokens are stored in `password_reset_tokens` table with:
- Unique token (64-character hex string)
- User ID
- Expiration time (1 hour from creation)
- Used flag (prevents token reuse)

**Automatic Cleanup:**
- Expired tokens are cleaned up hourly by background process
- Used tokens are marked but not immediately deleted (for audit purposes)

## Testing Email Functionality

### Manual Testing

1. Start the application:
```bash
go run ./cmd/server
```

2. Navigate to http://localhost:8080/login
3. Click "Forgot Password?"
4. Enter a verified email address (in SES sandbox, only verified emails work)
5. Check your email for the reset link
6. Click the link and set a new password

### Checking Logs

The application logs email operations:

```
Email service enabled: from=noreply@yourdomain.com, region=us-east-1
Email sent successfully: to=user@example.com, subject=Reset Your WordClash Password
```

If emails are disabled:
```
Email service disabled: SES_FROM_EMAIL not configured
Skipping email send (service disabled): password reset to user@example.com
```

## Troubleshooting

### Email Not Received

1. **Check SES Sandbox Status**
   - In sandbox mode, you can only send to verified addresses
   - Verify recipient email in SES console or request production access

2. **Check Spam Folder**
   - SES emails may be flagged as spam initially
   - Configure SPF/DKIM records for your domain to improve deliverability

3. **Check Application Logs**
   - Look for error messages in server output
   - Verify AWS credentials are configured correctly

4. **Verify SES Sending Limits**
   - Check SES console for sending quota and rate limits
   - New accounts have low limits initially

### Common Errors

**Error: "Email service disabled"**
- Solution: Set `SES_FROM_EMAIL` environment variable

**Error: "Failed to send email: MessageRejected"**
- Solution: Verify the sender email in SES console

**Error: "Failed to load AWS config"**
- Solution: Check AWS credentials configuration

**Error: "Email address not verified"**
- Solution: Verify both sender and recipient (if in sandbox)

### Testing Without Email

The application works fine without email configuration:
- Password reset endpoints return success messages
- No emails are sent (logged instead)
- All other features work normally

## Security Considerations

1. **Token Security**
   - Tokens are cryptographically random (32 bytes = 64 hex chars)
   - Tokens expire after 1 hour
   - Tokens can only be used once

2. **Rate Limiting**
   - Password reset requests are rate-limited
   - Prevents email bombing attacks

3. **Information Disclosure Prevention**
   - Always shows success message, even if email doesn't exist
   - Prevents email enumeration attacks

4. **HTTPS Required**
   - Use HTTPS in production for `APP_BASE_URL`
   - Prevents token interception

## Cost Estimation

Amazon SES Pricing (as of 2024):
- First 62,000 emails per month: **FREE** (when sent from EC2)
- Additional emails: $0.10 per 1,000 emails
- Very cost-effective for small to medium applications

## Future Enhancements

The email service is designed to support additional use cases:

- Welcome emails on registration (already implemented, just uncomment in Register handler)
- Activity notifications
- Password change confirmations
- Account deletion confirmations
- Child progress reports for parents

To send welcome emails, update the Register handler in `auth_handler.go`:

```go
// Send welcome email (optional)
if h.emailService != nil && h.emailService.IsEnabled() {
    ctx := context.Background()
    _ = h.emailService.SendWelcomeEmail(ctx, user.Email, user.Name)
}
```

## Support

For issues related to:
- **SES Configuration**: AWS Support or SES documentation
- **Application Email Features**: Check application logs and verify configuration
- **Email Deliverability**: Configure SPF/DKIM/DMARC for your domain
