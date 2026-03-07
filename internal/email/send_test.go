package email

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"testing"
)

// fakeSMTP 模拟不支持 STARTTLS 的 SMTP 服务器（端口 25 场景）
func fakeSMTP(t *testing.T) (host string, port int, closeFn func()) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	h, p, _ := net.SplitHostPort(addr)
	port, _ = strconv.Atoi(p)
	go func() {
		conn, _ := ln.Accept()
		if conn == nil {
			return
		}
		defer conn.Close()
		conn.Write([]byte("220 localhost ESMTP\r\n"))
		r := bufio.NewReader(conn)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "EHLO") || strings.HasPrefix(line, "HELO") {
				// 返回与连接地址一致的主机名，避免 wrong host name
				conn.Write([]byte("250 127.0.0.1\r\n"))
			} else if strings.HasPrefix(line, "AUTH PLAIN") {
				conn.Write([]byte("235 OK\r\n"))
			} else if strings.HasPrefix(line, "MAIL FROM") {
				conn.Write([]byte("250 OK\r\n"))
			} else if strings.HasPrefix(line, "RCPT TO") {
				conn.Write([]byte("250 OK\r\n"))
			} else if line == "DATA" {
				conn.Write([]byte("354 Go ahead\r\n"))
			} else if line == "." {
				conn.Write([]byte("250 OK\r\n"))
			} else if line == "QUIT" {
				conn.Write([]byte("221 Bye\r\n"))
				return
			}
		}
	}()
	return h, port, func() { ln.Close() }
}

func TestSend_AllowPlain_Port25(t *testing.T) {
	host, port, closeFn := fakeSMTP(t)
	defer closeFn()

	cfg := &Config{
		SMTPHost:   host,
		SMTPPort:   port, // 任意端口，allow_plain 时走明文路径
		From:       "test@local",
		Password:   "secret",
		To:         []string{"to@local"},
		AllowPlain: true,
	}

	err := cfg.Send("", "test", "body")
	if err != nil {
		t.Logf("Send err: %v", err)
		// 关键：不能是 "unencrypted connection"，说明 allow_plain 生效
		if strings.Contains(err.Error(), "unencrypted connection") {
			t.Fatalf("allow_plain 未生效，仍报 unencrypted connection")
		}
		return
	}
	t.Log("Send 成功")
}
