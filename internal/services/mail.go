package services

import (
	"fmt"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func SendInvitationEmail(toEmail, inviterName, loginURL string) error {
	from := mail.NewEmail("Katalog", "keremakaynak@gmail.com")
	subject := fmt.Sprintf("%s has invited you to join their team on Katalog!", inviterName)
	to := mail.NewEmail("New User", toEmail)

	htmlContent := fmt.Sprintf(`
        <div style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f4f4f4; text-align: center;">
			<div style="background-color: #ffffff; border-radius: 8px; padding: 30px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); display: inline-block; text-align: center;">
				<h1 style="color: #2c3e50; margin-bottom: 20px;">You've Been Invited!</h1>
				<p>Hello,</p>
				<p>You've been invited to join Katalog by <strong>%s</strong>. We're excited to have you on board!</p>
				<p>Katalog gives you data superpowers!</p>
				<a href="%s/login" style="display: inline-block; background-color: #3498db; color: #ffffff; text-decoration: none; padding: 12px 24px; border-radius: 4px; font-weight: bold; margin-top: 20px;">Get Started</a>
				<p>If you have any questions, feel free to reach out to our support team.</p>
				<p>Best regards,<br>Kerem from Katalog</p>
			</div>
			<div style="margin-top: 30px; text-align: center; font-size: 12px; color: #7f8c8d;">
				<p>© 2024 Katalog. All rights reserved.</p>
			</div>
		</div>
        `, inviterName, loginURL)

	plainTextContent := fmt.Sprintf("Hello, you've been invited to join Katalog by %s. Click the link below to get started: %s", inviterName, loginURL)

	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	_, err := client.Send(message)
	if err != nil {
		return err
	}
	return nil
}
