package mailer

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"

	"contrail/internal/config"
)

// Mailer handles transactional email delivery via SMTP.
type Mailer struct {
	cfg *config.Config
}

// NewMailer creates a new Mailer instance.
func NewMailer(cfg *config.Config) *Mailer {
	return &Mailer{cfg: cfg}
}

// SendVerificationCode sends a 6-digit verification code to the recipient.
func (m *Mailer) SendVerificationCode(recipientEmail, code string) error {
	subject := "Contrail API - Email Verification Code"
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Verification Code</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background-color: #f7f7f8; margin: 0; padding: 30px; color: #1c1d1f; }
    .card { max-width: 520px; margin: 0 auto; background: #ffffff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 36px; box-shadow: 0 4px 16px rgba(0,0,0,0.04); }
    .header { text-align: center; margin-bottom: 24px; }
    .logo-text { font-size: 22px; font-weight: 700; color: #111827; letter-spacing: -0.5px; }
    .badge { display: inline-block; background: #f3f4f6; color: #4b5563; font-size: 11px; font-weight: 600; padding: 3px 8px; border-radius: 9999px; vertical-align: middle; margin-left: 6px; }
    .desc { font-size: 14px; color: #4b5563; line-height: 1.6; margin-bottom: 24px; text-align: center; }
    .code-box { background: #f8fafc; border: 1px dashed #cbd5e1; border-radius: 8px; padding: 18px; text-align: center; margin: 24px 0; }
    .code { font-size: 32px; font-family: ui-monospace, Menlo, Consolas, monospace; font-weight: 700; letter-spacing: 6px; color: #2563eb; }
    .footer { margin-top: 32px; font-size: 12px; color: #9ca3af; text-align: center; line-height: 1.5; border-top: 1px solid #f3f4f6; padding-top: 20px; }
    .lang-block { margin-top: 12px; font-size: 13px; color: #6b7280; text-align: center; }
  </style>
</head>
<body>
  <div class="card">
    <div class="header">
      <span class="logo-text">Contrail</span><span class="badge">API Portal</span>
    </div>
    <div class="desc">
      Thank you for registering with Contrail. Please use the verification code below to verify your email address. This code is valid for 10 minutes.
    </div>
    <div class="code-box">
      <div class="code">%s</div>
    </div>
    <div class="lang-block">
      [中文] 您的注册验证码为上述 6 位数字，有效期 10 分钟。请勿泄露给他人。<br>
      [日本語] ご登録の確認コードは上記6桁の数字です。有効期間は10分間です。
    </div>
    <div class="footer">
      Contrail Near-Real-Time Flight Telemetry Service<br>
      Strictly Non-Commercial &amp; Academic Research Use Only<br>
      Domain: airline.api.veyino.org &bull; Contact: team@veyino.org
    </div>
  </div>
</body>
</html>`, code)

	return m.sendMail(recipientEmail, subject, htmlBody)
}

// SendRegistrationClosedNotice sends an official notice stating that the user registration channel is currently closed (no verification code).
func (m *Mailer) SendRegistrationClosedNotice(recipientEmail string) error {
	subject := "Contrail API - Registration Channel Closed / 注册通道关闭告知书"
	htmlBody := `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Registration Closed</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background-color: #f7f7f8; margin: 0; padding: 30px; color: #1c1d1f; }
    .card { max-width: 520px; margin: 0 auto; background: #ffffff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 36px; box-shadow: 0 4px 16px rgba(0,0,0,0.04); }
    .header { text-align: center; margin-bottom: 24px; }
    .logo-text { font-size: 22px; font-weight: 700; color: #111827; letter-spacing: -0.5px; }
    .badge { display: inline-block; background: #fef2f2; color: #dc2626; font-size: 11px; font-weight: 600; padding: 3px 8px; border-radius: 9999px; vertical-align: middle; margin-left: 6px; border: 1px solid #fee2e2; }
    .desc { font-size: 14px; color: #374151; line-height: 1.6; margin-bottom: 20px; text-align: left; }
    .notice-box { background: #fef2f2; border: 1px solid #fecaca; border-radius: 8px; padding: 16px; text-align: center; margin: 20px 0; color: #991b1b; font-size: 15px; font-weight: 600; }
    .footer { margin-top: 32px; font-size: 12px; color: #9ca3af; text-align: center; line-height: 1.5; border-top: 1px solid #f3f4f6; padding-top: 20px; }
    .lang-block { margin-top: 16px; font-size: 13px; color: #6b7280; text-align: left; line-height: 1.6; border-top: 1px dashed #e5e7eb; padding-top: 14px; }
  </style>
</head>
<body>
  <div class="card">
    <div class="header">
      <span class="logo-text">Contrail</span><span class="badge">Notice / 告知</span>
    </div>
    <div class="notice-box">
      Registration Channel Currently Closed / 注册通道已暂停开放
    </div>
    <div class="desc">
      We received a verification request from this email address. However, the user registration channel for Contrail API is currently closed by the system administrator. No verification code is issued.
    </div>
    <div class="lang-block">
      <strong>[中文说明]</strong><br>
      系统收到针对此邮箱的注册验证请求。目前 Contrail API 的新用户注册通道已由管理员暂停开放，本邮件为通道关闭告知书，不包含任何注册验证码。当通道重新开放后，您可再次发起注册申请。如有学术研究或数据对接需求，请联系团队邮箱。<br><br>
      <strong>[English Notice]</strong><br>
      New user registration is currently suspended. This email serves as an official notice and does not contain a verification code. Please try again when the registration window reopens.
    </div>
    <div class="footer">
      Contrail Near-Real-Time Flight Telemetry Service<br>
      Strictly Non-Commercial &amp; Academic Research Use Only<br>
      Domain: airline.api.veyino.org &bull; Contact: team@veyino.org
    </div>
  </div>
</body>
</html>`

	return m.sendMail(recipientEmail, subject, htmlBody)
}

// SendRateLimitAlert sends a rate-limit warning notification to a user.
func (m *Mailer) SendRateLimitAlert(recipientEmail, keyPrefix string, currentRPM int) error {
	subject := "Contrail API - High Request Frequency Warning"
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Rate Limit Alert</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background-color: #f7f7f8; margin: 0; padding: 30px; color: #1c1d1f; }
    .card { max-width: 540px; margin: 0 auto; background: #ffffff; border: 1px solid #fecaca; border-radius: 12px; padding: 36px; box-shadow: 0 4px 16px rgba(0,0,0,0.04); }
    .header { text-align: center; margin-bottom: 20px; }
    .logo-text { font-size: 22px; font-weight: 700; color: #dc2626; }
    .desc { font-size: 14px; color: #374151; line-height: 1.6; margin-bottom: 20px; }
    .meta-box { background: #fef2f2; border: 1px solid #fee2e2; border-radius: 8px; padding: 14px 18px; margin: 18px 0; font-size: 13px; color: #991b1b; }
    .footer { margin-top: 28px; font-size: 12px; color: #9ca3af; text-align: center; line-height: 1.5; border-top: 1px solid #f3f4f6; padding-top: 18px; }
  </style>
</head>
<body>
  <div class="card">
    <div class="header">
      <span class="logo-text">Contrail API Rate Limit Warning</span>
    </div>
    <div class="desc">
      We detected an unusually high request frequency from your API key (prefix: <code>%s</code>). The key exceeded the allowed request threshold (%d req/min limit).
    </div>
    <div class="meta-box">
      <strong>Notice / 提示:</strong><br>
      Contrail is an open, non-commercial research infrastructure with finite upstream bandwidth. Please reduce your client polling frequency and implement local client caching. Excessive repeated rate-limit violations may lead to automatic temporary throttling or key revocation.
    </div>
    <div class="footer">
      Contrail Flight API &bull; airline.api.veyino.org &bull; Support: team@veyino.org
    </div>
  </div>
</body>
</html>`, keyPrefix, currentRPM)

	return m.sendMail(recipientEmail, subject, htmlBody)
}

// SendStatusAlert sends an operational or test alert notification to the specified recipient.
func (m *Mailer) SendStatusAlert(recipientEmail, subject, alertContent string) error {
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>%s</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background-color: #f7f7f8; margin: 0; padding: 30px; color: #1c1d1f; }
    .card { max-width: 560px; margin: 0 auto; background: #ffffff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 36px; box-shadow: 0 4px 16px rgba(0,0,0,0.04); }
    .header { text-align: center; margin-bottom: 24px; }
    .logo-text { font-size: 22px; font-weight: 700; color: #111827; }
    .badge { display: inline-block; background: #dbeafe; color: #1e40af; font-size: 11px; font-weight: 600; padding: 3px 8px; border-radius: 9999px; vertical-align: middle; margin-left: 6px; }
    .desc { font-size: 14px; color: #374151; line-height: 1.6; margin-bottom: 20px; white-space: pre-wrap; }
    .footer { margin-top: 28px; font-size: 12px; color: #9ca3af; text-align: center; line-height: 1.5; border-top: 1px solid #f3f4f6; padding-top: 18px; }
  </style>
</head>
<body>
  <div class="card">
    <div class="header">
      <span class="logo-text">Contrail</span><span class="badge">System Notification</span>
    </div>
    <div class="desc">%s</div>
    <div class="footer">
      Contrail Flight API &bull; airline.api.veyino.org &bull; Support: team@veyino.org
    </div>
  </div>
</body>
</html>`, subject, alertContent)

	return m.sendMail(recipientEmail, subject, htmlBody)
}

// sendMail executes the SMTP transaction.
func (m *Mailer) sendMail(to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.SMTPHost, m.cfg.SMTPPort)
	host := m.cfg.SMTPHost

	headers := make(map[string]string)
	headers["From"] = m.cfg.SMTPFrom
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"
	headers["Date"] = time.Now().Format(time.RFC1123Z)

	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(htmlBody)

	// Custom dialer that supports STARTTLS
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP %s: %w", addr, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	// Use STARTTLS if available
	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	}
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	// Authenticate if credentials provided
	if m.cfg.SMTPUser != "" && m.cfg.SMTPPass != "" {
		auth := smtp.PlainAuth("", m.cfg.SMTPUser, m.cfg.SMTPPass, host)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	fromAddr := extractEmail(m.cfg.SMTPFrom)
	if err = client.Mail(fromAddr); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA command failed: %w", err)
	}
	if _, err = w.Write([]byte(message.String())); err != nil {
		w.Close()
		return fmt.Errorf("failed to write email body: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("failed to close email body: %w", err)
	}

	log.Printf("[Mailer] Successfully dispatched email to %s (subject: %s)", to, subject)
	return nil
}

func extractEmail(from string) string {
	start := strings.Index(from, "<")
	end := strings.Index(from, ">")
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(from[start+1 : end])
	}
	return strings.TrimSpace(from)
}
