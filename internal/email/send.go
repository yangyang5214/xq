package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/beer/xq/internal/logger"
)

// plainAuthWrapper 包装 PlainAuth，强制 ServerInfo.TLS=true 以绕过 PlainAuth 的 TLS 检查
// 仅当 allow_plain 且端口 25 时使用，密码将以明文传输
type plainAuthWrapper struct {
	smtp.Auth
}

func (a plainAuthWrapper) Start(server *smtp.ServerInfo) (string, []byte, error) {
	s := *server
	s.TLS = true
	return a.Auth.Start(&s)
}

// Send 使用 Config 向 to 发送主题为 subject、正文为 body 的邮件
// to 为逗号分隔的收件人；若为空则使用 Config.To
func (c *Config) Send(to string, subject, body string) error {
	logger.Log.Printf("[email] 开始发送: host=%s port=%d from=%s allow_plain=%v", c.SMTPHost, c.SMTPPort, c.From, c.AllowPlain)
	addrs := c.recipients(to)
	if len(addrs) == 0 {
		logger.Log.Printf("[email] 错误: 无收件人")
		return fmt.Errorf("无收件人")
	}
	logger.Log.Printf("[email] 收件人: %v", addrs)
	addr := fmt.Sprintf("%s:%d", c.SMTPHost, c.SMTPPort)
	tlsConfig := &tls.Config{ServerName: c.SMTPHost}

	var conn *smtp.Client
	var err error
	if c.SMTPPort == 465 {
		logger.Log.Printf("[email] 连接 %s (465 隐式 TLS)…", addr)
		// 465 端口：隐式 TLS，先建立 TLS 连接
		var tlsConn *tls.Conn
		tlsConn, err = tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			logger.Log.Printf("[email] TLS 连接失败: %v", err)
			return err
		}
		defer tlsConn.Close()
		conn, err = smtp.NewClient(tlsConn, c.SMTPHost)
		if err != nil {
			return err
		}
		defer conn.Close()
	} else if c.AllowPlain {
		logger.Log.Printf("[email] 连接 %s (明文 allow_plain)…", addr)
		// allow_plain：明文连接，使用 wrapper 绕过 PlainAuth 检查（常用于端口 25）
		var rawConn net.Conn
		rawConn, err = net.Dial("tcp", addr)
		if err != nil {
			logger.Log.Printf("[email] 连接失败: %v", err)
			return err
		}
		defer rawConn.Close()
		conn, err = smtp.NewClient(rawConn, c.SMTPHost)
		if err != nil {
			logger.Log.Printf("[email] NewClient 失败: %v", err)
			return err
		}
		defer conn.Close()
	} else {
		logger.Log.Printf("[email] 连接 %s (STARTTLS)…", addr)
		// 25/587 等：先明文连接，再升级为 TLS
		var rawConn net.Conn
		rawConn, err = net.Dial("tcp", addr)
		if err != nil {
			logger.Log.Printf("[email] 连接失败: %v", err)
			return err
		}
		defer rawConn.Close()
		conn, err = smtp.NewClient(rawConn, c.SMTPHost)
		if err != nil {
			logger.Log.Printf("[email] NewClient 失败: %v", err)
			return err
		}
		defer conn.Close()
		logger.Log.Printf("[email] StartTLS…")
		if err = conn.StartTLS(tlsConfig); err != nil {
			logger.Log.Printf("[email] StartTLS 失败: %v", err)
			return err
		}
	}
	logger.Log.Printf("[email] 认证…")

	var auth smtp.Auth
	if c.AllowPlain {
		auth = plainAuthWrapper{smtp.PlainAuth("", c.From, c.Password, c.SMTPHost)}
	} else {
		auth = smtp.PlainAuth("", c.From, c.Password, c.SMTPHost)
	}
	if err = conn.Auth(auth); err != nil {
		logger.Log.Printf("[email] 认证失败: %v", err)
		return err
	}
	logger.Log.Printf("[email] MAIL FROM…")
	if err = conn.Mail(c.From); err != nil {
		logger.Log.Printf("[email] MAIL FROM 失败: %v", err)
		return err
	}
	for _, a := range addrs {
		if err = conn.Rcpt(a); err != nil {
			logger.Log.Printf("[email] RCPT TO %s 失败: %v", a, err)
			return err
		}
	}
	logger.Log.Printf("[email] DATA…")
	w, err := conn.Data()
	if err != nil {
		logger.Log.Printf("[email] DATA 失败: %v", err)
		return err
	}
	toHeader := strings.Join(addrs, ", ")
	msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s", toHeader, subject, body)
	if _, err = w.Write([]byte(msg)); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		logger.Log.Printf("[email] 写入正文失败: %v", err)
		return err
	}
	logger.Log.Printf("[email] QUIT…")
	if err = conn.Quit(); err != nil {
		logger.Log.Printf("[email] QUIT 失败: %v", err)
		return err
	}
	logger.Log.Printf("[email] 发送成功")
	return nil
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
