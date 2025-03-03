package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

func Send(to []string, subject, body string) error {
	smtpCfg, err := config.SMTP(config.GetInstance())
	if err != nil {
		return err
	}

	c, err := smtp.DialStartTLS(fmt.Sprintf("%s:%d", smtpCfg.Host, smtpCfg.Port), &tls.Config{RootCAs: smtpCfg.Certs})
	defer c.Quit()

	if err != nil {
		return err
	}

	auth := sasl.NewPlainClient("ztmfapi", smtpCfg.User, smtpCfg.Pass)

	err = c.Auth(auth)

	if err != nil {
		return err
	}

	contacts, err := model.FindDataCallContacts(context.Background())
	if err != nil {
		return err
	}

	msg := strings.NewReader("")

	for _, contact := range contacts {
		msg.Reset("To: " + contact.Email + "\r\n" + "Subject: " + subject + "\r\n" + "\r\n" + body + "\r\n")
		err = c.SendMail(smtpCfg.From, []string{contact.Email}, msg)
		if err != nil {
			log.Println("error sending email: ", err)
		}
	}

	return nil
}
