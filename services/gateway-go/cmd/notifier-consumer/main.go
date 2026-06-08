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
	TaskID         int64    `json:"task_id"`
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

func main() {
	ctx := context.Background()

	db, err := connectDB(ctx)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer db.Close()

	if err := runMigrations(ctx, db); err != nil {
		log.Fatalf("consumer migrations failed: %v", err)
	}

	queueName := envOr("RABBITMQ_REMINDER_QUEUE", "reminder.triggered")
	rabbitURL := rabbitMQURL()

	conn, ch, msgs, err := connectConsumer(queueName, rabbitURL)
	if err != nil {
		log.Fatalf("consumer connect failed: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	log.Printf("notifier consumer started: queue=%s", queueName)
	for d := range msgs {
		if err := handleMessage(ctx, db, d.Body); err != nil {
			log.Printf("consume failed, requeue=true: %v", err)
			_ = d.Nack(false, true)
			continue
		}
		_ = d.Ack(false)
	}
}

func runMigrations(ctx context.Context, db *pgxpool.Pool) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS notification_schedule (
			id BIGSERIAL PRIMARY KEY,
			correlation_id TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'scheduled',
			topic TEXT NOT NULL,
			action TEXT NOT NULL,
			user_id BIGINT NOT NULL,
			event_id BIGINT,
			task_id BIGINT,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			timezone TEXT NOT NULL DEFAULT 'UTC',
			start_at TIMESTAMPTZ NOT NULL,
			end_at TIMESTAMPTZ NOT NULL,
			trigger_at TIMESTAMPTZ NOT NULL,
			offset_minutes INTEGER NOT NULL,
			target_channels JSONB NOT NULL DEFAULT '["push"]'::jsonb,
			last_error TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			canceled_at TIMESTAMPTZ
		)`,
		`ALTER TABLE notification_schedule ADD COLUMN IF NOT EXISTS task_id BIGINT`,
		`ALTER TABLE notification_schedule ALTER COLUMN event_id DROP NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_notification_schedule_status_trigger ON notification_schedule (status, trigger_at)`,
		`CREATE INDEX IF NOT EXISTS idx_notification_schedule_task_status_trigger ON notification_schedule (task_id, status, trigger_at)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(ctx, q); err != nil {
			return err
		}
	}

	return nil
}

func handleMessage(ctx context.Context, db *pgxpool.Pool, raw []byte) error {
	var msg reminderTriggeredMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return fmt.Errorf("invalid json payload: %w", err)
	}

	if msg.CorrelationID == "" || msg.UserID <= 0 || (msg.EventID <= 0 && msg.TaskID <= 0) {
		return fmt.Errorf("invalid required fields")
	}

	triggerAt, err := time.Parse(time.RFC3339, msg.TriggerAt)
	if err != nil {
		return fmt.Errorf("invalid trigger_at: %w", err)
	}
	startAt, err := time.Parse(time.RFC3339, msg.StartAt)
	if err != nil {
		return fmt.Errorf("invalid start_at: %w", err)
	}
	endAt, err := time.Parse(time.RFC3339, msg.EndAt)
	if err != nil {
		return fmt.Errorf("invalid end_at: %w", err)
	}

	channelsJSON, err := json.Marshal(msg.TargetChannels)
	if err != nil {
		return fmt.Errorf("invalid channels: %w", err)
	}

	ctxDB, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	if msg.Action == "delete" {
		if msg.TaskID > 0 {
			_, err = db.Exec(ctxDB, `
				UPDATE notification_schedule
				SET status = 'canceled',
				    canceled_at = NOW(),
				    updated_at = NOW()
				WHERE task_id = $1
				  AND status IN ('scheduled', 'queued')
			`, msg.TaskID)
			return err
		}

		_, err = db.Exec(ctxDB, `
			UPDATE notification_schedule
			SET status = 'canceled',
			    canceled_at = NOW(),
			    updated_at = NOW()
			WHERE event_id = $1
			  AND status IN ('scheduled', 'queued')
		`, msg.EventID)
		return err
	}

	_, err = db.Exec(ctxDB, `
		INSERT INTO notification_schedule (
			correlation_id, status, topic, action, user_id, event_id, task_id,
			title, description, timezone, start_at, end_at,
			trigger_at, offset_minutes, target_channels
		)
		VALUES (
			$1, 'scheduled', $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14::jsonb
		)
		ON CONFLICT (correlation_id)
		DO UPDATE SET
			status = 'scheduled',
			action = EXCLUDED.action,
			event_id = EXCLUDED.event_id,
			task_id = EXCLUDED.task_id,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			timezone = EXCLUDED.timezone,
			start_at = EXCLUDED.start_at,
			end_at = EXCLUDED.end_at,
			trigger_at = EXCLUDED.trigger_at,
			offset_minutes = EXCLUDED.offset_minutes,
			target_channels = EXCLUDED.target_channels,
			last_error = NULL,
			updated_at = NOW(),
			canceled_at = NULL
	`, msg.CorrelationID, msg.Topic, msg.Action, msg.UserID, nullableID(msg.EventID), nullableID(msg.TaskID), msg.Title, msg.Description, defaultTZ(msg.Timezone), startAt.UTC(), endAt.UTC(), triggerAt.UTC(), msg.OffsetMinutes, string(channelsJSON))
	if err != nil {
		return err
	}

	return nil
}

func connectConsumer(queueName, rabbitURL string) (*amqp.Connection, *amqp.Channel, <-chan amqp.Delivery, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, nil, err
	}

	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, nil, err
	}

	msgPrefetch := envOrInt("CONSUMER_PREFETCH", 50)
	if err := ch.Qos(msgPrefetch, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, nil, err
	}

	msgs, err := ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, nil, err
	}

	return conn, ch, msgs, nil
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

func nullableID(v int64) interface{} {
	if v <= 0 {
		return nil
	}
	return v
}
