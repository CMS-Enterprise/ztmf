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

// Send looks up contacts and sends emails with the provided subject and body
// it is meant to run as a background go routine and therefore logs errors rather than returning them
func Send(subject, body string) {
	var (
		contacts []*model.DataCallContact
		tlsCfg   *tls.Config
	)

	cfg := config.GetInstance()

	if cfg.SMTP.Certs != nil {
		tlsCfg = &tls.Config{RootCAs: cfg.SMTP.Certs}
	}

	c, err := smtp.DialStartTLS(fmt.Sprintf("%s:%d", cfg.SMTP.Host, cfg.SMTP.Port), tlsCfg)
	defer c.Quit()

	if err != nil {
		log.Println("error dialing tls: ", err)
		return
	}

	auth := sasl.NewPlainClient("ztmfapi", cfg.SMTP.User, cfg.SMTP.Pass)

	err = c.Auth(auth)

	if err != nil {
		log.Println("error authenticating to smtp server: ", err)
		return
	}

	if cfg.SMTP.TestMode {
		contacts, err = model.FindTestDataCallContacts(context.Background())
	} else {
		contacts, err = model.FindDataCallContacts(context.Background())
	}

	if err != nil {
		log.Println("error finding contacts: ", err)
		return
	}

	msg := strings.NewReader("")

	for _, contact := range contacts {
		msg.Reset("To: " + contact.Email + "\r\n" + "Subject: " + subject + "\r\n" + "\r\n" + body + "\r\n")
		err = c.SendMail(cfg.SMTP.From, []string{contact.Email}, msg)
		if err != nil {
			log.Println("error sending email: ", err)
		}
	}
}
