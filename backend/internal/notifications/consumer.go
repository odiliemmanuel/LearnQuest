// Package notifications is an optional, in-process consumer for the email
// verification queue. In the reference deployment it runs as its own service
// (notification-service/), but hosts without background workers — e.g. a free
// Render web service — can run it inside the backend instead by setting
// RUN_EMAIL_CONSUMER=true.
//
// It speaks the same topology the outbox declares, so it is a drop-in
// replacement: exchange learnquest.events, topic
// identity.email_verification_requested, queue learnquest.notifications.email.
package notifications

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const brevoEndpoint = "https://api.brevo.com/v3/smtp/email"

const (
	eventExchange = "learnquest.events"
	eventTopic    = "identity.email_verification_requested"
	queueName     = "learnquest.notifications.email"
	retryQueue    = "learnquest.notifications.email.retry"
	dlxExchange   = "learnquest.notifications.dlx"
	dlqName       = "learnquest.notifications.email.dlq"
	maxRetries    = 5
)

type Consumer struct {
	RabbitMQURL string
	EmailFrom   string

	// AppURL is the public frontend origin, used to build the "verify" link in
	// the email. Empty disables the link.
	AppURL string

	// Provider is "smtp" (default) or "brevo". Brevo sends over HTTPS (port
	// 443), which hosts that block outbound SMTP ports — e.g. Render's free
	// tier — can still reach.
	Provider string

	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string

	BrevoAPIKey string

	client *http.Client
}

func (c *Consumer) httpClient() *http.Client {
	if c.client == nil {
		c.client = &http.Client{Timeout: 15 * time.Second}
	}
	return c.client
}

type verificationEmail struct {
	EventID   string    `json:"eventId"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type smtpError struct {
	Code int
	Err  error
}

func (e *smtpError) Error() string { return fmt.Sprintf("smtp server rejected request: %v", e.Err) }
func (e *smtpError) Unwrap() error { return e.Err }

// apiError is a non-2xx HTTP response from an email API (Brevo). Retryable
// distinguishes transient failures (429 / 5xx) from permanent ones (bad key,
// rejected sender), which go straight to the dead-letter queue.
type apiError struct {
	StatusCode int
	Body       string
	Retryable  bool
}

func (e *apiError) Error() string {
	return fmt.Sprintf("email api status %d: %s", e.StatusCode, e.Body)
}

// Start runs the consumer in a background goroutine, reconnecting forever until
// ctx is cancelled. It never returns an error: a dead broker must not take the
// API down with it.
func (c *Consumer) Start(ctx context.Context) {
	go func() {
		for ctx.Err() == nil {
			if err := c.consume(ctx); err != nil && ctx.Err() == nil {
				log.Printf("notifications: consumer stopped: %v; retrying in 5s", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
			}
		}
	}()
}

func (c *Consumer) consume(ctx context.Context) error {
	conn, err := amqp.Dial(c.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("connect rabbitmq: %w", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()
	if err := declareTopology(ch); err != nil {
		return err
	}
	if err := ch.Qos(10, 0, false); err != nil {
		return fmt.Errorf("set qos: %w", err)
	}
	deliveries, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume queue: %w", err)
	}
	log.Printf("notifications: consuming %s", queueName)

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("delivery channel closed")
			}
			c.handleDelivery(ctx, ch, delivery)
		}
	}
}

func declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(eventExchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare event exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(dlxExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlx exchange: %w", err)
	}
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare email queue: %w", err)
	}
	if err := ch.QueueBind(queueName, eventTopic, eventExchange, false, nil); err != nil {
		return fmt.Errorf("bind email queue: %w", err)
	}
	if _, err := ch.QueueDeclare(retryQueue, true, false, false, false, amqp.Table{
		"x-message-ttl":             int32(30_000),
		"x-dead-letter-exchange":    eventExchange,
		"x-dead-letter-routing-key": eventTopic,
	}); err != nil {
		return fmt.Errorf("declare retry queue: %w", err)
	}
	if _, err := ch.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dead-letter queue: %w", err)
	}
	if err := ch.QueueBind(dlqName, eventTopic, dlxExchange, false, nil); err != nil {
		return fmt.Errorf("bind dead-letter queue: %w", err)
	}
	return nil
}

func (c *Consumer) handleDelivery(ctx context.Context, ch *amqp.Channel, delivery amqp.Delivery) {
	var message verificationEmail
	if err := json.Unmarshal(delivery.Body, &message); err != nil || message.EventID == "" || message.Email == "" || message.Code == "" {
		log.Printf("notifications: invalid event %q: %v", delivery.MessageId, err)
		if err := publishDLQ(ch, delivery); err != nil {
			log.Printf("notifications: dead-letter failed: %v", err)
			_ = delivery.Nack(false, true)
			return
		}
		_ = delivery.Ack(false)
		return
	}
	if time.Now().After(message.ExpiresAt) {
		log.Printf("notifications: skipping expired verification event %s", message.EventID)
		_ = delivery.Ack(false)
		return
	}
	if err := c.sendVerificationEmail(ctx, message); err != nil {
		retries := retryCount(delivery.Headers)
		log.Printf("notifications: send failed event=%s retries=%d: %v", message.EventID, retries, err)
		var scheduleErr error
		if !retryable(err) || retries >= maxRetries {
			scheduleErr = publishDLQ(ch, delivery)
		} else {
			scheduleErr = publishRetry(ch, delivery, retries+1)
		}
		if scheduleErr != nil {
			log.Printf("notifications: could not schedule delivery retry: %v", scheduleErr)
			_ = delivery.Nack(false, true)
			return
		}
		_ = delivery.Ack(false)
		return
	}
	log.Printf("notifications: verification email accepted event=%s recipient=%s", message.EventID, message.Email)
	_ = delivery.Ack(false)
}

func publishRetry(ch *amqp.Channel, delivery amqp.Delivery, retry int) error {
	headers := copyHeaders(delivery.Headers)
	headers["x-retry-count"] = int32(retry)
	if err := ch.PublishWithContext(context.Background(), "", retryQueue, false, false, amqp.Publishing{
		ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: delivery.MessageId,
		Headers: headers, Body: delivery.Body,
	}); err != nil {
		return fmt.Errorf("publish retry: %w", err)
	}
	return nil
}

func publishDLQ(ch *amqp.Channel, delivery amqp.Delivery) error {
	if err := ch.PublishWithContext(context.Background(), dlxExchange, eventTopic, false, false, amqp.Publishing{
		ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: delivery.MessageId,
		Headers: delivery.Headers, Body: delivery.Body,
	}); err != nil {
		return fmt.Errorf("publish dead-letter: %w", err)
	}
	return nil
}

func (c *Consumer) sendVerificationEmail(ctx context.Context, message verificationEmail) error {
	subject := "Verify your LearnQuest email"
	body := verificationHTML(message.Name, message.Code, c.verifyURL(message))
	if strings.EqualFold(c.Provider, "brevo") {
		return c.sendBrevo(ctx, message.Email, subject, body)
	}
	return c.sendSMTP(ctx, message.Email, subject, body)
}

// verifyURL deep-links to the frontend verification page with the email and
// code pre-filled, so the recipient can verify with one click. Returns "" when
// no AppURL is configured.
func (c *Consumer) verifyURL(message verificationEmail) string {
	if c.AppURL == "" {
		return ""
	}
	query := url.Values{}
	query.Set("email", message.Email)
	query.Set("code", message.Code)
	return c.AppURL + "/verify-email?" + query.Encode()
}

// sendBrevo delivers over the Brevo transactional email API (HTTPS), used on
// hosts that block outbound SMTP ports.
func (c *Consumer) sendBrevo(ctx context.Context, to, subject, htmlBody string) error {
	payload, err := json.Marshal(map[string]any{
		"sender":      map[string]string{"email": c.EmailFrom, "name": "LearnQuest"},
		"to":          []map[string]string{{"email": to}},
		"subject":     subject,
		"htmlContent": htmlBody,
	})
	if err != nil {
		return fmt.Errorf("encode brevo payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, brevoEndpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build brevo request: %w", err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("api-key", c.BrevoAPIKey)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("call brevo api: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	detail, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return &apiError{
		StatusCode: resp.StatusCode,
		Body:       strings.TrimSpace(string(detail)),
		Retryable:  resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500,
	}
}

func (c *Consumer) sendSMTP(ctx context.Context, to, subject, htmlBody string) error {
	addr := net.JoinHostPort(c.SMTPHost, c.SMTPPort)
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	if err := ctx.Err(); err != nil {
		return err
	}
	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("connect to smtp server: %w", err)
	}

	var conn net.Conn
	if c.SMTPPort == "465" {
		tlsConn := tls.Client(rawConn, &tls.Config{ServerName: c.SMTPHost})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			rawConn.Close()
			return fmt.Errorf("establish TLS with smtp server: %w", err)
		}
		conn = tlsConn
	} else {
		conn = rawConn
	}
	client, err := smtp.NewClient(conn, c.SMTPHost)
	if err != nil {
		conn.Close()
		return fmt.Errorf("start smtp session: %w", err)
	}
	defer client.Close()

	if c.SMTPPort != "465" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: c.SMTPHost}); err != nil {
				return fmt.Errorf("start starttls with smtp server: %w", err)
			}
		}
	}
	auth := smtp.PlainAuth("", c.SMTPUsername, c.SMTPPassword, c.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return &smtpError{Code: smtpReplyCode(err), Err: err}
	}
	if err := client.Mail(c.EmailFrom); err != nil {
		return &smtpError{Code: smtpReplyCode(err), Err: err}
	}
	if err := client.Rcpt(to); err != nil {
		return &smtpError{Code: smtpReplyCode(err), Err: err}
	}
	writer, err := client.Data()
	if err != nil {
		return &smtpError{Code: smtpReplyCode(err), Err: err}
	}
	body := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		c.EmailFrom, to, subject, htmlBody,
	)
	if _, err := writer.Write([]byte(body)); err != nil {
		writer.Close()
		return fmt.Errorf("write email body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return &smtpError{Code: smtpReplyCode(err), Err: err}
	}
	return client.Quit()
}

func smtpReplyCode(err error) int {
	if err == nil {
		return 0
	}
	parsed, ok := parseSMTPReply(err)
	if !ok {
		return 0
	}
	return parsed
}

func parseSMTPReply(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	msg := err.Error()
	var code int
	for _, prefix := range []string{"535 ", "550 ", "551 ", "553 ", "554 ", "452 ", "450 ", "421 ", "451 "} {
		if strings.HasPrefix(msg, prefix) {
			fmt.Sscanf(prefix, "%d", &code)
			return code, true
		}
	}
	return 0, false
}

func retryable(err error) bool {
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		return apiErr.Retryable
	}
	var smtpErr *smtpError
	if errors.As(err, &smtpErr) {
		if smtpErr.Code == 0 {
			return true
		}
		return smtpErr.Code >= 450 && smtpErr.Code < 500
	}
	return true
}

func verificationHTML(name, code, verifyURL string) string {
	button := ""
	if verifyURL != "" {
		button = fmt.Sprintf(
			`<a href="%s" style="display:block;margin:24px 0;padding:14px;background:#4f46e5;border-radius:12px;text-align:center;color:#ffffff;font-weight:700;text-decoration:none">Verify my email</a><p style="color:#94a3b8;font-size:12px;word-break:break-all">Or paste this link into your browser:<br>%s</p>`,
			html.EscapeString(verifyURL), html.EscapeString(verifyURL),
		)
	}
	return fmt.Sprintf(`<!doctype html><html><body style="margin:0;background:#f8fafc;font-family:Arial,sans-serif;color:#0f172a"><div style="max-width:520px;margin:40px auto;background:#ffffff;border-radius:18px;padding:36px"><div style="font-weight:700;color:#4f46e5;font-size:18px">LearnQuest</div><h1 style="font-size:24px;margin:24px 0 8px">Verify your email</h1><p>Hello %s, use this code to finish creating your account.</p><div style="margin:28px 0;padding:18px;background:#eef2ff;border-radius:12px;text-align:center;font-size:30px;letter-spacing:8px;font-weight:700;color:#3730a3">%s</div>%s<p style="color:#64748b;font-size:14px">This code expires in 10 minutes. If you did not create a LearnQuest account, you can ignore this email.</p></div></body></html>`, html.EscapeString(name), code, button)
}

func retryCount(headers amqp.Table) int {
	v, ok := headers["x-retry-count"]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int32:
		return int(n)
	case int64:
		return int(n)
	case int:
		return n
	case string:
		parsed, _ := strconv.Atoi(n)
		return parsed
	default:
		return 0
	}
}

func copyHeaders(source amqp.Table) amqp.Table {
	out := amqp.Table{}
	for key, value := range source {
		out[key] = value
	}
	return out
}
