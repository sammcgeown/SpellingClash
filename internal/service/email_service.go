package service

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// EmailService handles sending emails via Amazon SES
type EmailService struct {
	client     *sesv2.Client
	fromEmail  string
	fromName   string
	appBaseURL string
	enabled    bool
	debug      bool
}

// NewEmailService creates a new email service
func NewEmailService(awsRegion, fromEmail, fromName, appBaseURL string, debug bool) (*EmailService, error) {
	// If fromEmail is empty, create a disabled service
	if fromEmail == "" {
		log.Println("Email service disabled: SES_FROM_EMAIL not configured")
		if debug {
			log.Println("[DEBUG] Email service will skip sending all emails")
		}
		return &EmailService{
			enabled: false,
			debug:   debug,
		}, nil
	}

	if debug {
		log.Printf("[DEBUG] Initializing email service with AWS SES")
		log.Printf("[DEBUG] AWS Region: %s", awsRegion)
		log.Printf("[DEBUG] From Email: %s", fromEmail)
		log.Printf("[DEBUG] From Name: %s", fromName)
		log.Printf("[DEBUG] App Base URL: %s", appBaseURL)
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		if debug {
			log.Printf("[DEBUG] Failed to load AWS config: %v", err)
		}
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	if debug {
		log.Println("[DEBUG] AWS config loaded successfully")
	}

	// Create SES client
	client := sesv2.NewFromConfig(cfg)

	log.Printf("Email service enabled: from=%s, region=%s", fromEmail, awsRegion)
	if debug {
		log.Println("[DEBUG] SES client created successfully")
	}

	return &EmailService{
		client:     client,
		fromEmail:  fromEmail,
		fromName:   fromName,
		appBaseURL: appBaseURL,
		enabled:    true,
		debug:      debug,
	}, nil
}

// IsEnabled returns whether the email service is enabled
func (s *EmailService) IsEnabled() bool {
	return s.enabled
}

// SendPasswordResetEmail sends a password reset email with a reset link
func (s *EmailService) SendPasswordResetEmail(ctx context.Context, toEmail, toName, resetToken string) error {
	if s.debug {
		log.Printf("[DEBUG] SendPasswordResetEmail called: to=%s, name=%s, token=%s", toEmail, toName, resetToken)
	}

	if !s.enabled {
		log.Printf("Skipping email send (service disabled): password reset to %s", toEmail)
		if s.debug {
			log.Printf("[DEBUG] Email service is disabled, no email will be sent")
		}
		return nil
	}

	resetLink := fmt.Sprintf("%s/auth/reset-password?token=%s", s.appBaseURL, resetToken)
	if s.debug {
		log.Printf("[DEBUG] Reset link generated: %s", resetLink)
	}

	subject := "Reset Your WordClash Password"
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.header { background-color: #4a90e2; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
		.content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 5px 5px; }
		.button { display: inline-block; padding: 12px 30px; background-color: #4a90e2; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
		.footer { text-align: center; margin-top: 20px; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>Password Reset Request</h1>
		</div>
		<div class="content">
			<p>Hi %s,</p>
			<p>We received a request to reset your password for your WordClash account.</p>
			<p>Click the button below to reset your password:</p>
			<p style="text-align: center;">
				<a href="%s" class="button">Reset Password</a>
			</p>
			<p>Or copy and paste this link into your browser:</p>
			<p style="word-break: break-all; font-size: 12px; color: #666;">%s</p>
			<p><strong>This link will expire in 1 hour.</strong></p>
			<p>If you didn't request a password reset, you can safely ignore this email.</p>
		</div>
		<div class="footer">
			<p>This is an automated email from WordClash. Please do not reply.</p>
		</div>
	</div>
</body>
</html>
`, toName, resetLink, resetLink)

	textBody := fmt.Sprintf(`Hi %s,

We received a request to reset your password for your WordClash account.

Click the link below to reset your password:
%s

This link will expire in 1 hour.

If you didn't request a password reset, you can safely ignore this email.

---
This is an automated email from WordClash. Please do not reply.
`, toName, resetLink)

	if s.debug {
		log.Printf("[DEBUG] Sending password reset email: subject=%s, to=%s", subject, toEmail)
		log.Printf("[DEBUG] HTML body length: %d bytes", len(htmlBody))
		log.Printf("[DEBUG] Text body length: %d bytes", len(textBody))
	}

	return s.sendEmail(ctx, toEmail, subject, htmlBody, textBody)
}

// SendWelcomeEmail sends a welcome email to new users
func (s *EmailService) SendWelcomeEmail(ctx context.Context, toEmail, toName string) error {
	if s.debug {
		log.Printf("[DEBUG] SendWelcomeEmail called: to=%s, name=%s", toEmail, toName)
	}

	if !s.enabled {
		log.Printf("Skipping email send (service disabled): welcome to %s", toEmail)
		if s.debug {
			log.Printf("[DEBUG] Email service is disabled, no email will be sent")
		}
		return nil
	}

	subject := "Welcome to WordClash!"
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.header { background-color: #4a90e2; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
		.content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 5px 5px; }
		.button { display: inline-block; padding: 12px 30px; background-color: #4a90e2; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
		.footer { text-align: center; margin-top: 20px; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>Welcome to WordClash!</h1>
		</div>
		<div class="content">
			<p>Hi %s,</p>
			<p>Thank you for creating your WordClash account! We're excited to help your children improve their spelling skills through fun and engaging games.</p>
			<p>Here's what you can do next:</p>
			<ul>
				<li>Add children to your family account</li>
				<li>Create custom spelling lists</li>
				<li>Track your children's progress</li>
				<li>Let your children practice with interactive games</li>
			</ul>
			<p style="text-align: center;">
				<a href="%s/login" class="button">Get Started</a>
			</p>
		</div>
		<div class="footer">
			<p>This is an automated email from WordClash. Please do not reply.</p>
		</div>
	</div>
</body>
</html>
`, toName, s.appBaseURL)

	textBody := fmt.Sprintf(`Hi %s,

Thank you for creating your WordClash account! We're excited to help your children improve their spelling skills through fun and engaging games.

Here's what you can do next:
- Add children to your family account
- Create custom spelling lists
- Track your children's progress
- Let your children practice with interactive games

Get started: %s/login

---
This is an automated email from WordClash. Please do not reply.
`, toName, s.appBaseURL)

	if s.debug {
		log.Printf("[DEBUG] Sending welcome email: subject=%s, to=%s", subject, toEmail)
		log.Printf("[DEBUG] HTML body length: %d bytes", len(htmlBody))
		log.Printf("[DEBUG] Text body length: %d bytes", len(textBody))
	}

	return s.sendEmail(ctx, toEmail, subject, htmlBody, textBody)
}

// sendEmail sends an email using Amazon SES
func (s *EmailService) sendEmail(ctx context.Context, toEmail, subject, htmlBody, textBody string) error {
	if s.debug {
		log.Printf("[DEBUG] sendEmail called: to=%s, subject=%s", toEmail, subject)
	}

	fromAddress := s.fromEmail
	if s.fromName != "" {
		fromAddress = fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail)
	}

	if s.debug {
		log.Printf("[DEBUG] From address: %s", fromAddress)
		log.Printf("[DEBUG] To address: %s", toEmail)
		log.Printf("[DEBUG] Subject: %s", subject)
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(fromAddress),
		Destination: &types.Destination{
			ToAddresses: []string{toEmail},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String(subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &types.Body{
					Html: &types.Content{
						Data:    aws.String(htmlBody),
						Charset: aws.String("UTF-8"),
					},
					Text: &types.Content{
						Data:    aws.String(textBody),
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
	}

	if s.debug {
		log.Printf("[DEBUG] Calling SES SendEmail API...")
	}

	result, err := s.client.SendEmail(ctx, input)
	if err != nil {
		if s.debug {
			log.Printf("[DEBUG] SES SendEmail failed: %v", err)
		}
		return fmt.Errorf("failed to send email to %s: %w", toEmail, err)
	}

	if s.debug {
		log.Printf("[DEBUG] SES SendEmail succeeded")
		if result.MessageId != nil {
			log.Printf("[DEBUG] Message ID: %s", *result.MessageId)
		}
	}

	log.Printf("Email sent successfully: to=%s, subject=%s", toEmail, subject)
	return nil
}

// SendInvitationEmail sends a custom invitation email (used by admin handler)
func (s *EmailService) SendInvitationEmail(ctx context.Context, toEmail, subject, htmlBody, textBody string) error {
	return s.sendEmail(ctx, toEmail, subject, htmlBody, textBody)
}
