package mail

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/mail"
	"strings"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/tlslog"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

// Send looks up contacts and sends emails with the provided subject and body
// it is meant to run as a background go routine and therefore logs errors rather than returning them
func Send(subject, body string, recipients []string) {
	var (
		cfg        = config.GetInstance()
		emailsSent int
	)

	// A nil RootCAs (no custom SMTP certs) still means the system roots, and
	// go-smtp fills in ServerName from the host, so always passing a config
	// changes nothing but the handshake log line.
	tlsCfg := &tls.Config{
		RootCAs:          cfg.SMTP.Certs,
		VerifyConnection: tlslog.Verifier("smtp"),
	}

	log.Println("dialing SMTP server...")
	c, err := smtp.DialStartTLS(fmt.Sprintf("%s:%d", cfg.SMTP.Host, cfg.SMTP.Port), tlsCfg)

	if err != nil {
		log.Println("error dialing tls: ", err)
		return
	}

	log.Println("authenticating to SMTP server...")

	auth := sasl.NewPlainClient("", cfg.SMTP.User, cfg.SMTP.Pass)
	err = c.Auth(auth)
	if err != nil {
		log.Println("error authenticating to smtp server: ", err)
		return
	}

	msg := strings.NewReader("")

	log.Println("sending emails...")

	for _, address := range recipients {
		address = strings.TrimSpace(address)

		_, err = mail.ParseAddress(address)
		if err != nil {
			log.Printf(`invalid email: "%s"`, address)
			continue
		}

		msg.Reset("To: " + address + "\r\n" + "From: " + cfg.SMTP.From + "\r\n" + "Subject: " + subject + "\r\n\r\n" + body + "\r\n")
		err = c.SendMail(cfg.SMTP.From, []string{address}, msg)
		if err != nil {
			log.Println("error sending email: ", err)
		} else {
			emailsSent++
		}
	}

	log.Printf("sent %d emails", emailsSent)

	c.Quit()
}
