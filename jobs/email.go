package jobs

import (
	"fmt"
	"net/smtp"
	"os"
)

var (
	smtpHost = "smtp.gmail.com"
	smtpPort = "587"
	smtpUser = os.Getenv("SMTP_USER")
	smtpPass = os.Getenv("SMTP_PASS")
)

func executeSendEmail(payload map[string]interface{}) (int, []byte, error) {

	to, ok := payload["to"].(string)
	if !ok {
		return 0, nil, fmt.Errorf("missing 'to'")
	}

	subject, ok := payload["subject"].(string)
	if !ok {
		return 0, nil, fmt.Errorf("missing 'subject'")
	}

	body, ok := payload["body"].(string)
	if !ok {
		return 0, nil, fmt.Errorf("missing 'body'")
	}

	message := []byte(
		"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-version: 1.0;\r\n" +
			"Content-Type: text/plain; charset=\"UTF-8\";\r\n\r\n" +
			body + "\r\n",
	)

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	err := smtp.SendMail(
		smtpHost+":"+smtpPort,
		auth,
		smtpUser,
		[]string{to},
		message,
	)

	if err != nil {
		return 500, nil, err
	}

	return 200, []byte(`{"message":"email sent"}`), nil
}