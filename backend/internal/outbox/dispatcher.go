// Package outbox publishes committed domain events to RabbitMQ. Events remain
// in Postgres until RabbitMQ confirms publication, so a restart cannot lose an
// email-verification request.
package outbox

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/learnquest/backend/internal/models"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
)

const (
	exchangeName          = "learnquest.events"
	notificationQueueName = "learnquest.notifications.email"
	verificationTopic     = "identity.email_verification_requested"
)

type Dispatcher struct {
	DB          *gorm.DB
	RabbitMQURL string
}

func (d *Dispatcher) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			if err := d.Dispatch(ctx); err != nil {
				log.Printf("outbox: dispatch failed: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (d *Dispatcher) Dispatch(ctx context.Context) error {
	var pending []models.OutboxEvent
	if err := d.DB.WithContext(ctx).
		Where("published_at IS NULL AND available_at <= ?", time.Now()).
		Order("created_at asc").Limit(25).Find(&pending).Error; err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}

	conn, err := amqp.Dial(d.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("connect rabbitmq: %w", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open rabbitmq channel: %w", err)
	}
	defer ch.Close()
	if err := ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare event exchange: %w", err)
	}
	// The producer owns this durable binding too. That way a verification
	// request waits in RabbitMQ even when the notification service is offline.
	if _, err := ch.QueueDeclare(notificationQueueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare notification queue: %w", err)
	}
	if err := ch.QueueBind(notificationQueueName, verificationTopic, exchangeName, false, nil); err != nil {
		return fmt.Errorf("bind notification queue: %w", err)
	}
	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, len(pending)))

	for _, event := range pending {
		publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := ch.PublishWithContext(publishCtx, exchangeName, event.Topic, false, false, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.ID,
			Timestamp:    event.CreatedAt,
			Body:         []byte(event.Payload),
		})
		cancel()
		if err != nil {
			d.markFailure(event, err)
			return fmt.Errorf("publish %s: %w", event.ID, err)
		}
		select {
		case confirmation := <-confirms:
			if !confirmation.Ack {
				err := fmt.Errorf("rabbitmq did not confirm publication")
				d.markFailure(event, err)
				return fmt.Errorf("publish %s: %w", event.ID, err)
			}
		case <-ctx.Done():
			d.markFailure(event, ctx.Err())
			return ctx.Err()
		}
		now := time.Now()
		if err := d.DB.WithContext(ctx).Model(&models.OutboxEvent{}).
			Where("id = ? AND published_at IS NULL", event.ID).
			Updates(map[string]any{"published_at": now, "last_error": ""}).Error; err != nil {
			return fmt.Errorf("mark %s published: %w", event.ID, err)
		}
	}
	return nil
}

func (d *Dispatcher) markFailure(event models.OutboxEvent, publishErr error) {
	attempts := event.AttemptCount + 1
	delay := time.Second * time.Duration(1<<min(attempts, 6))
	if err := d.DB.Model(&models.OutboxEvent{}).Where("id = ?", event.ID).Updates(map[string]any{
		"attempt_count": attempts,
		"available_at":  time.Now().Add(delay),
		"last_error":    publishErr.Error(),
	}).Error; err != nil {
		log.Printf("outbox: could not record publish failure for %s: %v", event.ID, err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
