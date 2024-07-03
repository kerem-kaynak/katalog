package services

import (
	"fmt"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func SendInvitationEmail(toEmail, inviterName, loginURL string) error {
	from := mail.NewEmail("Katalog", "no-reply@katalog.so")
	subject := fmt.Sprintf("%s has invited you to join their team on Katalog!", inviterName)
	to := mail.NewEmail("New User", toEmail)

	htmlContent := fmt.Sprintf(`
        <div style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f4f4f4; text-align: center;">
			<div style="background-color: #ffffff; border-radius: 8px; padding: 30px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); display: inline-block; text-align: center;">
				<h1 style="color: #2c3e50; margin-bottom: 32px;">You've Been Invited!</h1>
				<a href="https://imgbb.com/"><img style="margin-bottom:32px; width:128px; height:128px" src="https://i.ibb.co/2s9Qq97/katalog-logo.png" alt="katalog-logo" border="0"></a>
				<p>Hello,</p>
				<p>You've been invited to join <strong>%s</strong>'s team on <a href="https://katalog.so">Katalog</a>. They're looking forward to having you on board!</p>
				<p>Katalog gives you and your team data superpowers!</p>
				<a href="%s/login" style="display: inline-block; background-color: #2563eb; color: #ffffff; text-decoration: none; padding: 12px 24px; border-radius: 4px; font-weight: bold; margin-top: 20px;">Get Started</a>
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
