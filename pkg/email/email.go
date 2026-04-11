package email

import (
	"fmt"
	"history-api/assets"
	"history-api/internal/models"
	"history-api/pkg/config"
	"history-api/pkg/constants"
	"strings"

	"github.com/wneessen/go-mail"
)

func SendMail(toEmail, subject, templatePath string, data map[string]string) error {
	userSmtp, err := config.GetConfig("SMTP_USER")
	if err != nil {
		return err
	}

	passSmtp, err := config.GetConfig("SMTP_PASS")
	if err != nil {
		return err
	}

	htmlTemplate, err := assets.GetFileContent(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read email template: %s", err)
	}

	finalHTML := htmlTemplate
	for k, v := range data {
		finalHTML = strings.ReplaceAll(finalHTML, "{{"+k+"}}", v)
	}

	message := mail.NewMsg()
	if err := message.From(userSmtp); err != nil {
		return fmt.Errorf("failed to set From: %s", err)
	}
	if err := message.To(toEmail); err != nil {
		return fmt.Errorf("failed to set To: %s", err)
	}

	message.Subject(subject)
	message.SetBodyString(mail.TypeTextHTML, finalHTML)

	client, err := mail.NewClient(
		"smtp.gmail.com",
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithTLSPortPolicy(mail.TLSMandatory),
		mail.WithUsername(userSmtp),
		mail.WithPassword(passSmtp),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %s", err)
	}

	if err := client.DialAndSend(message); err != nil {
		return fmt.Errorf("failed to send mail: %s", err)
	}

	return nil
}

func SendMailOTP(dto *models.TokenEntity) error {
	var subject string
	var templatePath string

	switch dto.TokenType {
	case constants.TokenPasswordReset:
		subject = "Your Password Reset Code"
		templatePath = "resources/password_reset.html"
	case constants.TokenEmailVerify:
		subject = "Verify your email address"
		templatePath = "resources/email_verify.html"
	default:
		return fmt.Errorf("invalid token type: %v", dto.TokenType)
	}

	return SendMail(dto.Email, subject, templatePath, map[string]string{
		"OTP_CODE": dto.Token,
	})
}

func SendHistorianReviewMail(dto *models.UserVerificationStorageEntity) error {
	var subject string
	var templatePath string
	feUrl := config.GetConfigWithDefault("FRONTEND_URL", "http://localhost:3000")
	switch dto.Status {
	case constants.StatusApproved:
		subject = "Your Historian Application is Approved"
		templatePath = "resources/historian_approved.html"

	case constants.StatusRejected:
		subject = "Your Historian Application is Rejected"
		templatePath = "resources/historian_rejected.html"

	default:
		return fmt.Errorf("invalid status: %v", dto.Status)
	}

	return SendMail(dto.Email, subject, templatePath, map[string]string{
		"NAME":    dto.Email,
		"REASON":  dto.ReviewNote,
		"APP_URL": feUrl,
	})
}
