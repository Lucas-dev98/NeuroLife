package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

type outboxRow struct {
	ID        int64
	UserID    int64
	EventID   int64
	EventType string
	Payload   []byte
}

type eventPayload struct {
	ID                     int64  `json:"id"`
	UserID                 int64  `json:"user_id"`
	Title                  string `json:"title"`
	Description            string `json:"description"`
	StartAt                string `json:"start_at"`
	EndAt                  string `json:"end_at"`
	Timezone               string `json:"timezone"`
	IsAllDay               bool   `json:"is_all_day"`
	ReminderOffsetsMinutes []int  `json:"reminder_offsets_minutes"`
}

type reminderTriggeredMessage struct {
	Version        string   `json:"version"`
	Topic          string   `json:"topic"`
	Action         string   `json:"action"`
	Source         string   `json:"source"`
	ProducedAt     string   `json:"produced_at"`
	OutboxID       int64    `json:"outbox_id"`
	EventType      string   `json:"event_type"`
	UserID         int64    `json:"user_id"`
	EventID        int64    `json:"event_id"`
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	Timezone       string   `json:"timezone"`
	StartAt        string   `json:"start_at"`
	EndAt          string   `json:"end_at"`
	IsAllDay       bool     `json:"is_all_day"`
	OffsetMinutes  int      `json:"offset_minutes"`
	TriggerAt      string   `json:"trigger_at"`
	TargetChannels []string `json:"target_channels"`
	CorrelationID  string   `json:"correlation_id"`
}

type publisher struct {
	url   string
	queue string
	conn  *amqp.Connection
	ch    *amqp.Channel
}

func newPublisher(url, queue string) *publisher {
	return &publisher{url: url, queue: queue}
}

func (p *publisher) ensure() error {
	if p.conn != nil && !p.conn.IsClosed() && p.ch != nil {
		return nil
	}

	if p.ch != nil {
		_ = p.ch.Close()
		p.ch = nil
	}
	if p.conn != nil && !p.conn.IsClosed() {
		_ = p.conn.Close()
		p.conn = nil
	}

	conn, err := amqp.Dial(p.url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}

	_, err = ch.QueueDeclare(
		p.queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}

	p.conn = conn
	p.ch = ch
	return nil
}

func (p *publisher) publish(ctx context.Context, body []byte) error {
	if err := p.ensure(); err != nil {
		return err
	}

	err := p.ch.PublishWithContext(
		ctx,
		"",
		p.queue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)
	if err != nil {
		_ = p.ch.Close()
		_ = p.conn.Close()
		p.ch = nil
		p.conn = nil
		return err
	}

	return nil
}

func (p *publisher) close() {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil && !p.conn.IsClosed() {
		_ = p.conn.Close()
	}
}

func main() {
	ctx := context.Background()

	db, err := connectDB(ctx)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer db.Close()

	pollSeconds := envOrInt("OUTBOX_POLL_SECONDS", 2)
	batchSize := envOrInt("OUTBOX_BATCH_SIZE", 50)
	queueName := envOr("RABBITMQ_REMINDER_QUEUE", "reminder.triggered")
	rabbitURL := rabbitMQURL()

	pub := newPublisher(rabbitURL, queueName)
	defer pub.close()

	log.Printf("outbox worker started: poll=%ds batch=%d queue=%s", pollSeconds, batchSize, queueName)

	ticker := time.NewTicker(time.Duration(pollSeconds) * time.Second)
	defer ticker.Stop()

	for {
		if err := processBatch(ctx, db, pub, batchSize); err != nil {
			log.Printf("process batch failed: %v", err)
		}
		<-ticker.C
	}
}

func processBatch(ctx context.Context, db *pgxpool.Pool, pub *publisher, batchSize int) error {
	ctxBatch, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	tx, err := db.Begin(ctxBatch)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctxBatch)

	rows, err := tx.Query(ctxBatch, `
		SELECT id, user_id, COALESCE(event_id, 0), event_type, payload
		FROM reminder_outbox
		WHERE status = 'pending'
		ORDER BY id ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, batchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	items := make([]outboxRow, 0)
	for rows.Next() {
		var r outboxRow
		if err := rows.Scan(&r.ID, &r.UserID, &r.EventID, &r.EventType, &r.Payload); err != nil {
			return err
		}
		items = append(items, r)
	}

	if len(items) == 0 {
		if err := tx.Commit(ctxBatch); err != nil {
			return err
		}
		return nil
	}

	processed := 0
	for _, item := range items {
		msgs, err := buildReminderMessages(item)
		if err != nil {
			return fmt.Errorf("build reminder payload for outbox %d: %w", item.ID, err)
		}

		for _, msg := range msgs {
			body, err := json.Marshal(msg)
			if err != nil {
				return err
			}
			if err := pub.publish(ctxBatch, body); err != nil {
				return err
			}
		}

		_, err = tx.Exec(ctxBatch, `
			UPDATE reminder_outbox
			SET status = 'processed',
			    processed_at = NOW()
			WHERE id = $1
		`, item.ID)
		if err != nil {
			return err
		}

		processed++
	}

	if err := tx.Commit(ctxBatch); err != nil {
		return err
	}

	if processed > 0 {
		log.Printf("processed outbox items: %d", processed)
	}

	return nil
}

func buildReminderMessages(row outboxRow) ([]reminderTriggeredMessage, error) {
	var event eventPayload
	if err := json.Unmarshal(row.Payload, &event); err != nil {
		return nil, err
	}

	startAt, err := time.Parse(time.RFC3339, event.StartAt)
	if err != nil {
		return nil, fmt.Errorf("invalid start_at: %w", err)
	}

	channels := []string{"push", "email", "whatsapp"}
	producedAt := time.Now().UTC()
	messages := make([]reminderTriggeredMessage, 0)

	action := "upsert"
	if row.EventType == "event.deleted" {
		action = "delete"
	}

	offsets := event.ReminderOffsetsMinutes
	if len(offsets) == 0 {
		offsets = []int{60}
	}

	if action == "delete" {
		messages = append(messages, reminderTriggeredMessage{
			Version:        "1.0",
			Topic:          "reminder.triggered",
			Action:         action,
			Source:         "gateway-go.outbox-worker",
			ProducedAt:     producedAt.Format(time.RFC3339),
			OutboxID:       row.ID,
			EventType:      row.EventType,
			UserID:         row.UserID,
			EventID:        event.ID,
			Title:          event.Title,
			Description:    event.Description,
			Timezone:       defaultTZ(event.Timezone),
			StartAt:        event.StartAt,
			EndAt:          event.EndAt,
			IsAllDay:       event.IsAllDay,
			OffsetMinutes:  0,
			TriggerAt:      producedAt.Format(time.RFC3339),
			TargetChannels: channels,
			CorrelationID:  fmt.Sprintf("outbox-%d-delete", row.ID),
		})
		return messages, nil
	}

	for _, offset := range offsets {
		triggerAt := startAt.Add(-time.Duration(offset) * time.Minute).UTC()
		messages = append(messages, reminderTriggeredMessage{
			Version:        "1.0",
			Topic:          "reminder.triggered",
			Action:         action,
			Source:         "gateway-go.outbox-worker",
			ProducedAt:     producedAt.Format(time.RFC3339),
			OutboxID:       row.ID,
			EventType:      row.EventType,
			UserID:         row.UserID,
			EventID:        event.ID,
			Title:          event.Title,
			Description:    event.Description,
			Timezone:       defaultTZ(event.Timezone),
			StartAt:        event.StartAt,
			EndAt:          event.EndAt,
			IsAllDay:       event.IsAllDay,
			OffsetMinutes:  offset,
			TriggerAt:      triggerAt.Format(time.RFC3339),
			TargetChannels: channels,
			CorrelationID:  fmt.Sprintf("outbox-%d-offset-%d", row.ID, offset),
		})
	}

	return messages, nil
}

func connectDB(ctx context.Context) (*pgxpool.Pool, error) {
	host := envOr("POSTGRES_HOST", "localhost")
	port := envOr("POSTGRES_PORT", "5432")
	user := envOr("POSTGRES_USER", "neurolife")
	password := envOr("POSTGRES_PASSWORD", "neurolife")
	dbName := envOr("POSTGRES_DB", "neurolife")
	sslMode := envOr("POSTGRES_SSLMODE", "disable")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbName, sslMode)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctxPing); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func rabbitMQURL() string {
	if direct := strings.TrimSpace(os.Getenv("RABBITMQ_URL")); direct != "" {
		return direct
	}
	host := envOr("RABBITMQ_HOST", "localhost")
	port := envOr("RABBITMQ_PORT", "5672")
	user := envOr("RABBITMQ_USER", envOr("RABBITMQ_DEFAULT_USER", "guest"))
	pass := envOr("RABBITMQ_PASS", envOr("RABBITMQ_DEFAULT_PASS", "guest"))
	return fmt.Sprintf("amqp://%s:%s@%s:%s/", user, pass, host, port)
}

func defaultTZ(tz string) string {
	if strings.TrimSpace(tz) == "" {
		return "UTC"
	}
	return tz
}

func envOr(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func envOrInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
