package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// Send 使用 Config 向 to 发送主题为 subject、正文为 body 的邮件
// to 为逗号分隔的收件人；若为空则使用 Config.To
func (c *Config) Send(to string, subject, body string) error {
	addrs := c.recipients(to)
	if len(addrs) == 0 {
		return fmt.Errorf("无收件人")
	}
	addr := fmt.Sprintf("%s:%d", c.SMTPHost, c.SMTPPort)
	conn, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	if c.SMTPPort == 587 {
		tlsConfig := &tls.Config{ServerName: c.SMTPHost}
		if err = conn.StartTLS(tlsConfig); err != nil {
			return err
		}
	}

	auth := smtp.PlainAuth("", c.From, c.Password, c.SMTPHost)
	if err = conn.Auth(auth); err != nil {
		return err
	}
	if err = conn.Mail(c.From); err != nil {
		return err
	}
	for _, a := range addrs {
		if err = conn.Rcpt(a); err != nil {
			return err
		}
	}
	w, err := conn.Data()
	if err != nil {
		return err
	}
	toHeader := strings.Join(addrs, ", ")
	msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s", toHeader, subject, body)
	if _, err = w.Write([]byte(msg)); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return conn.Quit()
}

func (c *Config) recipients(to string) []string {
	if to != "" {
		var list []string
		for _, a := range strings.Split(to, ",") {
			a = strings.TrimSpace(a)
			if a != "" {
				list = append(list, a)
			}
		}
		return list
	}
	return c.To
}
