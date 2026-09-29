package notification

import (
	"context"
	"net"
	"net/textproto"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildEmailHeaders(t *testing.T) {
	h := buildEmailHeaders("noreply@example.com", []string{"a@b.com", "c@d.com"}, Message{
		Subject: "Hello",
		Body:    "Hi",
	})
	assert.Contains(t, h, "From: noreply@example.com")
	assert.Contains(t, h, "To: a@b.com, c@d.com")
	assert.Contains(t, h, "Subject: Hello")
	assert.Contains(t, h, "MIME-Version: 1.0")
	assert.Contains(t, h, "Content-Type: text/plain; charset=UTF-8")
}

func TestEmailSendMissingConfig(t *testing.T) {
	e := NewEmail(EmailConfig{})
	err := e.Send(context.Background(), Message{To: "a@b.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host not configured")
}

func TestEmailSendEmptyRecipient(t *testing.T) {
	e := NewEmail(EmailConfig{Host: "smtp.example.com", Port: 587})
	err := e.Send(context.Background(), Message{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recipient is empty")
}

// fakeSMTPServer 极简 SMTP 服务器，用于验证发送流程
type fakeSMTPServer struct {
	ln       net.Listener
	received []string
}

func newFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &fakeSMTPServer{ln: ln}
	go s.serve()
	return s
}

func (s *fakeSMTPServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *fakeSMTPServer) handle(conn net.Conn) {
	defer conn.Close()
	tp := textproto.NewConn(conn)
	write := func(line string) { _ = tp.PrintfLine("%s", line) }
	write("220 fake ESMTP")
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		switch {
		case strings.HasPrefix(line, "EHLO") || strings.HasPrefix(line, "HELO"):
			write("250-fake")
			write("250 OK")
		case strings.HasPrefix(line, "MAIL FROM"):
			write("250 OK")
		case strings.HasPrefix(line, "RCPT TO"):
			write("250 OK")
		case line == "DATA":
			write("354 go ahead")
			var buf strings.Builder
			for {
				dl, err := tp.ReadLine()
				if err != nil {
					return
				}
				if dl == "." {
					break
				}
				buf.WriteString(dl + "\n")
			}
			s.received = append(s.received, buf.String())
			write("250 OK")
		case line == "QUIT":
			write("221 bye")
			return
		default:
			write("250 OK")
		}
	}
}

func TestEmailSendSMTPIntegration(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer srv.ln.Close()

	e := NewEmail(EmailConfig{
		Host: "127.0.0.1",
		Port: srv.ln.Addr().(*net.TCPAddr).Port,
		From: "noreply@example.com",
	})
	err := e.Send(context.Background(), Message{
		To:      "a@b.com",
		Subject: "Hello",
		Body:    "Hi there",
	})
	assert.NoError(t, err)
}
