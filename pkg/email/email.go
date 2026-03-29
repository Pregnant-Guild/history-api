package email

import (
	"fmt"
	"history-api/assets"
	"history-api/pkg/config"
	"history-api/pkg/constants"
	"strings"

	"github.com/wneessen/go-mail"
)

func SendMailOTP(toEmail, otpCode string, tokenType constants.TokenType) error {
	userSmtp, err := config.GetConfig("SMTP_USER")
	if err != nil {
		return err
	}

	passSmtp, err := config.GetConfig("SMTP_PASS")
	if err != nil {
		return err
	}

	var subject string
	var templatePath string

	switch tokenType {
	case constants.TokenPasswordReset:
		subject = "Your Password Reset Code"
		templatePath = "resources/password_reset.html"
	case constants.TokenEmailVerify:
		subject = "Verify your email address"
		templatePath = "resources/email_verify.html"
	default:
		return fmt.Errorf("invalid token type: %v", tokenType)
	}
	htmlTemplate, err := assets.GetFileContent(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read email template: %s", err)
	}

	message := mail.NewMsg()
	if err := message.From(userSmtp); err != nil {
		return fmt.Errorf("failed to set From email address: %s", err)
	}
	if err := message.To(toEmail); err != nil {
		return fmt.Errorf("failed to set To email address: %s", err)
	}

	finalHTML := strings.ReplaceAll(htmlTemplate, "{{OTP_CODE}}", otpCode)

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
		return fmt.Errorf("failed to create mail client: %s", err)
	}

	err = client.DialAndSend(message)
	if err != nil {
		return fmt.Errorf("failed to send mail: %s", err)
	}
	return nil
}
