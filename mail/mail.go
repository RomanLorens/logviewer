package mail

import (
	"context"
	"fmt"
	"net/smtp"

	l "github.com/RomanLorens/logviewer/logger"
)

var logger = l.L

//Mail mail
type Mail struct {
	server string
}

//NewEmail email
func NewEmail(server string) *Mail {
	return &Mail{server: server}
}

//Send sends email
func (m Mail) Send(ctx context.Context, to []string, subject string, msg string) error {
	logger.Info(ctx, "Sending email '%v'...", subject)
	_subject := fmt.Sprintf("Subject: %s\n", subject)
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	_msg := []byte(_subject + mime + msg)
	err := smtp.SendMail(m.server, nil, "donotreply@citi.com", to, _msg)
	if err != nil {
		return fmt.Errorf("Could not send email, %v", err)
	}
	return nil
}
