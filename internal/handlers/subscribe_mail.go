package handlers

import (
	"context"
	"fmt"
	"log"

	"github.com/resend/resend-go/v3"

	"softstore/internal/config"
)

// sendConfirmEmail sends the double opt-in confirmation email containing
// a one-click confirm link.
func sendConfirmEmail(to, confirmToken, source string) error {
	link := fmt.Sprintf("%s/subscribe/confirm?token=%s", config.BaseURL(), confirmToken)

	client := resend.NewClient(config.ResendAPIKey())
	params := &resend.SendEmailRequest{
		From:    "hello@lauden.dev",
		To:      []string{to},
		ReplyTo: "hello@lauden.dev",
		Subject: confirmSubject(source),
		Html:    buildConfirmHTML(link, source),
		Text:    buildConfirmText(link, source),
	}

	sent, err := client.Emails.SendWithContext(context.Background(), params)
	if err != nil {
		return fmt.Errorf("resend send failed: %w", err)
	}
	log.Println("resend accepted, id:", sent.Id)
	return nil
}

func buildConfirmText(link, source string) string {
	return fmt.Sprintf(
		"%s:\n\n%s\n\nIf you didn't request this, ignore this email.\n\n— Tyler L. Laudenslager, lauden.dev\n",
		confirmTextLead(source),
		link,
	)
}

func buildConfirmHTML(link, source string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="margin:0;padding:0;background:#FDF8F0;font-family:-apple-system,BlinkMacSystemFont,'Inter',sans-serif;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="background:#FDF8F0;padding:32px 16px;">
		<tr>
			<td align="center">
				<table width="480" cellpadding="0" cellspacing="0" style="background:#FFFFFF;border-radius:16px;overflow:hidden;border:1px solid #EBE4D6;">
					<tr>
						<td style="padding:32px 32px 24px;">
							<h1 style="font-size:22px;color:#111827;margin:0 0 12px;">Confirm your subscription</h1>
							<p style="color:#6B7280;font-size:14px;margin:0 0 20px;">Click below to confirm you'd like %s.</p>
							<a href="%s" style="display:inline-block;background:#3B82F6;color:#FFFFFF;text-decoration:none;padding:12px 24px;border-radius:8px;font-size:14px;font-weight:600;">Confirm subscription</a>
							<p style="color:#9CA3AF;font-size:12px;margin:24px 0 0;">If you didn't request this, you can safely ignore this email.</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>`, confirmWhat(source), link)
}

// confirmSubject, confirmTextLead and confirmWhat vary the confirmation email
// by signup source. Anything other than "popfocus" gets the original shop text.
func confirmSubject(source string) string {
	if source == "popfocus" {
		return "Confirm: get notified when Pop Focus is released"
	}
	return "Confirm your subscription"
}

func confirmTextLead(source string) string {
	if source == "popfocus" {
		return "Confirm you'd like an email when Pop Focus is released"
	}
	return "Confirm your subscription to lauden.dev updates"
}

func confirmWhat(source string) string {
	if source == "popfocus" {
		return "an email when Pop Focus is released"
	}
	return "updates on new releases and software from lauden.dev"
}
