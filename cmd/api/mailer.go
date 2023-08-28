package main

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"
)

//go:embed templates
var emailTemplateFS embed.FS

func (app *application) SendEmail(from, to, subject, tmpl string, data interface{}) error {
	templateToRender := fmt.Sprintf("templates/%s.html.tmpl", tmpl)
	
	t, err := template.New("email-html").ParseFS(emailTemplateFS, templateToRender)
	if err != nil {
		app.errorLog.Println(err)
		return err
	}
	
	var tpl bytes.Buffer
	//inserir data no body da template
	if err = t.ExecuteTemplate(&tpl, "body", data); err != nil {
		app.errorLog.Println(err)
		return err
	}
	
	formattedMessage := tpl.String()
	
	templateToRender = fmt.Sprintf("templates/%s.plain.tmpl", tmpl)
	t, err = template.New("email-plain").ParseFS(emailTemplateFS, templateToRender)
	if err != nil {
		app.errorLog.Println(err)
		return err
	}

	
	if err = t.ExecuteTemplate(&tpl, "body", data); err != nil {
		app.errorLog.Println(err)
		return err
	}
	
	//body marcado na template
	if err = t.ExecuteTemplate(&tpl, "body", data); err != nil {
		app.errorLog.Println("quarto template")
		return err
	}
	
	//email em texto
	plainMessage := tpl.String()

	//send email
	mailServer := mail.NewSMTPClient()
	mailServer.Host = app.config.smtp.host
	mailServer.Port = app.config.smtp.port
	mailServer.Username = app.config.smtp.username
	mailServer.Password = app.config.smtp.password
	mailServer.Encryption = mail.EncryptionTLS
	mailServer.KeepAlive = false // manter conexao
	mailServer.ConnectTimeout = 10 * time.Second
	mailServer.SendTimeout = 10 * time.Second
	
	smtpClient, err := mailServer.Connect()
	if err != nil {
		app.errorLog.Println(err)
		return err
	}
	
	email := mail.NewMSG()
	email.SetFrom(from).AddTo(to). SetSubject(subject)
	
	email.SetBody(mail.TextHTML, formattedMessage)//email em html
	email.AddAlternative(mail.TextPlain, plainMessage)// email em texto
	
	err = email.Send(smtpClient)
	if err != nil {
		app.errorLog.Println(err)
		return err
	}

	return nil
}