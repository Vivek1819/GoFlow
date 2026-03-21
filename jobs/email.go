package jobs

import (
	"context" // ✅ ADD
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

func executeSendEmail(ctx context.Context, payload map[string]interface{}) (int, []byte, error) {

	// 🔴 EARLY CANCEL CHECK
	if ctx.Err() == context.Canceled {
		return 0, nil, fmt.Errorf("email cancelled")
	}

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

	errChan := make(chan error, 1)

	// 🔥 RUN EMAIL IN GOROUTINE
	go func() {
		err := smtp.SendMail(
			smtpHost+":"+smtpPort,
			auth,
			smtpUser,
			[]string{to},
			message,
		)
		errChan <- err
	}()

	// 🔥 RACE: CANCEL vs SEND
	select {

	case <-ctx.Done():
		return 0, nil, fmt.Errorf("email cancelled")

	case err := <-errChan:
		if err != nil {
			return 500, nil, err
		}
	}

	return 200, []byte(`{"message":"email sent"}`), nil
}