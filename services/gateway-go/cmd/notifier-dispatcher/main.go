package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type scheduleRow struct {
	ID             int64
	CorrelationID  string
	UserID         int64
	EventID        int64
	UserEmail      sql.NullString
	Title          string
	Description    string
	Timezone       string
	StartAt        time.Time
	EndAt          time.Time
	TriggerAt      time.Time
	OffsetMinutes  int
	TargetChannels []byte
	TargetEmail    sql.NullString
	TargetPush     sql.NullString
	TargetWhatsApp sql.NullString
	RetryCount     int
	MaxRetries     int
}

type dispatchMode string

const (
	dispatchModeMock  dispatchMode = "mock"
	dispatchModeMixed dispatchMode = "mixed"
	dispatchModeReal  dispatchMode = "real"
)

type channelDispatcher struct {
	mode         dispatchMode
	failChannels map[string]bool
	push         *pushAdapter
	email        *emailAdapter
	whatsapp     *whatsAppAdapter
}

type pushAdapter struct {
	endpoint     string
	authBearer   string
	defaultToken string
	client       *http.Client
}

type emailAdapter struct {
	host        string
	port        int
	username    string
	password    string
	fromAddress string
}

type whatsAppAdapter struct {
	endpoint  string
	authToken string
	defaultTo string
	client    *http.Client
}

type deadLetterPayload struct {
	ScheduleID     int64    `json:"schedule_id"`
	CorrelationID  string   `json:"correlation_id"`
	UserID         int64    `json:"user_id"`
	EventID        int64    `json:"event_id"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Timezone       string   `json:"timezone"`
	StartAt        string   `json:"start_at"`
	EndAt          string   `json:"end_at"`
	TriggerAt      string   `json:"trigger_at"`
	OffsetMinutes  int      `json:"offset_minutes"`
	TargetChannels []string `json:"target_channels"`
	RetryCount     int      `json:"retry_count"`
	MaxRetries     int      `json:"max_retries"`
}

func main() {
	ctx := context.Background()

	db, err := connectDB(ctx)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer db.Close()

	if err := runMigrations(ctx, db); err != nil {
		log.Fatalf("dispatcher migrations failed: %v", err)
	}

	pollSeconds := envOrInt("DISPATCHER_POLL_SECONDS", 2)
	batchSize := envOrInt("DISPATCHER_BATCH_SIZE", 50)
	dispatcher := newChannelDispatcherFromEnv()

	log.Printf("notifier dispatcher started: poll=%ds batch=%d", pollSeconds, batchSize)
	log.Printf("dispatch mode: %s", dispatcher.mode)
	if len(dispatcher.failChannels) > 0 {
		keys := make([]string, 0, len(dispatcher.failChannels))
		for k := range dispatcher.failChannels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		log.Printf("forced failure channels enabled: %s", strings.Join(keys, ","))
	}

	ticker := time.NewTicker(time.Duration(pollSeconds) * time.Second)
	defer ticker.Stop()

	for {
		if err := processBatch(ctx, db, batchSize, dispatcher); err != nil {
			log.Printf("dispatcher batch failed: %v", err)
		}
		<-ticker.C
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
			event_id BIGINT NOT NULL,
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
		`ALTER TABLE notification_schedule ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE notification_schedule ADD COLUMN IF NOT EXISTS max_retries INTEGER NOT NULL DEFAULT 5`,
		`ALTER TABLE notification_schedule ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ`,
		`ALTER TABLE notification_schedule ADD COLUMN IF NOT EXISTS dispatched_at TIMESTAMPTZ`,
		`ALTER TABLE notification_schedule ADD COLUMN IF NOT EXISTS target_email TEXT`,
		`ALTER TABLE notification_schedule ADD COLUMN IF NOT EXISTS target_push TEXT`,
		`ALTER TABLE notification_schedule ADD COLUMN IF NOT EXISTS target_whatsapp TEXT`,
		`CREATE TABLE IF NOT EXISTS notification_dead_letter (
			id BIGSERIAL PRIMARY KEY,
			schedule_id BIGINT NOT NULL,
			correlation_id TEXT NOT NULL,
			payload JSONB NOT NULL,
			last_error TEXT NOT NULL,
			failed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notification_schedule_dispatch ON notification_schedule (status, trigger_at, next_attempt_at)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func processBatch(ctx context.Context, db *pgxpool.Pool, batchSize int, dispatcher *channelDispatcher) error {
	ctxBatch, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	tx, err := db.Begin(ctxBatch)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctxBatch)

	rows, err := tx.Query(ctxBatch, `
		SELECT s.id, s.correlation_id, s.user_id, s.event_id, u.email,
		       s.title, s.description, s.timezone,
		       start_at, end_at, trigger_at, offset_minutes, target_channels,
		       s.target_email, s.target_push, s.target_whatsapp,
		       retry_count, max_retries
		FROM notification_schedule s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE s.status = 'scheduled'
		  AND s.canceled_at IS NULL
		  AND s.trigger_at <= NOW()
		  AND (s.next_attempt_at IS NULL OR s.next_attempt_at <= NOW())
		ORDER BY s.trigger_at ASC, s.id ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, batchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	items := make([]scheduleRow, 0)
	for rows.Next() {
		var r scheduleRow
		if err := rows.Scan(
			&r.ID, &r.CorrelationID, &r.UserID, &r.EventID,
			&r.UserEmail, &r.Title, &r.Description, &r.Timezone,
			&r.StartAt, &r.EndAt, &r.TriggerAt, &r.OffsetMinutes,
			&r.TargetChannels, &r.TargetEmail, &r.TargetPush, &r.TargetWhatsApp,
			&r.RetryCount, &r.MaxRetries,
		); err != nil {
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

	for _, item := range items {
		if err := dispatchOne(ctxBatch, tx, item, dispatcher); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctxBatch); err != nil {
		return err
	}

	log.Printf("dispatcher processed rows: %d", len(items))
	return nil
}

func dispatchOne(ctx context.Context, tx pgx.Tx, row scheduleRow, dispatcher *channelDispatcher) error {
	channels := []string{}
	if err := json.Unmarshal(row.TargetChannels, &channels); err != nil {
		return markFailure(ctx, tx, row, fmt.Errorf("invalid target_channels json: %w", err))
	}
	if len(channels) == 0 {
		channels = []string{"push"}
	}

	for _, channel := range channels {
		if err := dispatcher.sendChannel(ctx, channel, row); err != nil {
			return markFailure(ctx, tx, row, err)
		}
	}

	_, err := tx.Exec(ctx, `
		UPDATE notification_schedule
		SET status = 'dispatched',
		    dispatched_at = NOW(),
		    next_attempt_at = NULL,
		    last_error = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`, row.ID)
	return err
}

func (d *channelDispatcher) sendChannel(ctx context.Context, channel string, row scheduleRow) error {
	ch := strings.ToLower(strings.TrimSpace(channel))
	if ch == "" {
		return fmt.Errorf("empty channel")
	}

	if d.failChannels[ch] {
		return fmt.Errorf("forced failure for channel %s", ch)
	}

	switch ch {
	case "push":
		if d.push == nil {
			return d.handleMissingConfig(ch, row)
		}
		return d.push.Send(ctx, row)
	case "email":
		if d.email == nil {
			return d.handleMissingConfig(ch, row)
		}
		return d.email.Send(row)
	case "whatsapp":
		if d.whatsapp == nil {
			return d.handleMissingConfig(ch, row)
		}
		return d.whatsapp.Send(ctx, row)
	default:
		return fmt.Errorf("unsupported channel: %s", ch)
	}
}

func (d *channelDispatcher) handleMissingConfig(channel string, row scheduleRow) error {
	if d.mode == dispatchModeMixed || d.mode == dispatchModeMock {
		log.Printf("dispatch fallback=mock channel=%s schedule_id=%d correlation=%s", channel, row.ID, row.CorrelationID)
		return nil
	}
	return fmt.Errorf("channel %s is not configured for real dispatch", channel)
}

func (a *pushAdapter) Send(ctx context.Context, row scheduleRow) error {
	token := firstNonEmpty(nullStringToString(row.TargetPush), a.defaultToken)
	if token == "" {
		return fmt.Errorf("push target not found for user_id=%d", row.UserID)
	}

	bodyMap := map[string]interface{}{
		"message": map[string]interface{}{
			"token": token,
			"notification": map[string]string{
				"title": row.Title,
				"body":  row.Description,
			},
			"data": map[string]string{
				"correlation_id": row.CorrelationID,
				"event_id":       fmt.Sprintf("%d", row.EventID),
				"trigger_at":     row.TriggerAt.UTC().Format(time.RFC3339),
			},
		},
	}
	raw, err := json.Marshal(bodyMap)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, strings.NewReader(string(raw)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.authBearer)

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("dispatch channel=push schedule_id=%d correlation=%s", row.ID, row.CorrelationID)
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return fmt.Errorf("push provider returned status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func (a *emailAdapter) Send(row scheduleRow) error {
	to := firstNonEmpty(nullStringToString(row.TargetEmail), nullStringToString(row.UserEmail))
	if to == "" {
		return fmt.Errorf("email target not found for user_id=%d", row.UserID)
	}

	addr := fmt.Sprintf("%s:%d", a.host, a.port)
	headers := []string{
		fmt.Sprintf("From: %s", a.fromAddress),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: NeuroLife - %s", sanitizeHeader(row.Title)),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
	}
	body := fmt.Sprintf(
		"%s\n\nInicio: %s\nTermino: %s\nHorario do lembrete: %s\n",
		row.Description,
		row.StartAt.UTC().Format(time.RFC3339),
		row.EndAt.UTC().Format(time.RFC3339),
		row.TriggerAt.UTC().Format(time.RFC3339),
	)
	message := strings.Join(headers, "\r\n") + body

	var auth smtp.Auth
	if a.username != "" {
		auth = smtp.PlainAuth("", a.username, a.password, a.host)
	}

	if err := smtp.SendMail(addr, auth, a.fromAddress, []string{to}, []byte(message)); err != nil {
		return err
	}

	log.Printf("dispatch channel=email schedule_id=%d correlation=%s to=%s", row.ID, row.CorrelationID, to)
	return nil
}

func (a *whatsAppAdapter) Send(ctx context.Context, row scheduleRow) error {
	to := firstNonEmpty(nullStringToString(row.TargetWhatsApp), a.defaultTo)
	if to == "" {
		return fmt.Errorf("whatsapp target not found for user_id=%d", row.UserID)
	}

	bodyMap := map[string]interface{}{
		"to":   to,
		"type": "text",
		"text": map[string]string{
			"body": fmt.Sprintf("%s\n%s", row.Title, row.Description),
		},
	}
	raw, err := json.Marshal(bodyMap)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, strings.NewReader(string(raw)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(a.authToken) != "" {
		req.Header.Set("Authorization", "Bearer "+a.authToken)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("dispatch channel=whatsapp schedule_id=%d correlation=%s to=%s", row.ID, row.CorrelationID, to)
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return fmt.Errorf("whatsapp provider returned status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func newChannelDispatcherFromEnv() *channelDispatcher {
	modeRaw := strings.ToLower(strings.TrimSpace(envOr("DISPATCH_CHANNEL_MODE", "mixed")))
	mode := dispatchMode(modeRaw)
	if mode != dispatchModeMock && mode != dispatchModeMixed && mode != dispatchModeReal {
		mode = dispatchModeMixed
	}

	d := &channelDispatcher{
		mode:         mode,
		failChannels: parseSet(envOr("DISPATCH_FAIL_CHANNELS", "")),
	}

	if mode == dispatchModeMock {
		return d
	}

	fcmEndpoint := strings.TrimSpace(envOr("FCM_ENDPOINT", ""))
	fcmBearer := strings.TrimSpace(envOr("FCM_AUTH_BEARER", ""))
	if fcmEndpoint != "" && fcmBearer != "" {
		d.push = &pushAdapter{
			endpoint:     fcmEndpoint,
			authBearer:   fcmBearer,
			defaultToken: strings.TrimSpace(envOr("FCM_DEFAULT_TOKEN", "")),
			client: &http.Client{
				Timeout: time.Duration(envOrInt("FCM_TIMEOUT_SECONDS", 8)) * time.Second,
			},
		}
	}

	smtpHost := strings.TrimSpace(envOr("SMTP_HOST", ""))
	smtpFrom := strings.TrimSpace(envOr("SMTP_FROM", ""))
	if smtpHost != "" && smtpFrom != "" {
		d.email = &emailAdapter{
			host:        smtpHost,
			port:        envOrInt("SMTP_PORT", 587),
			username:    strings.TrimSpace(envOr("SMTP_USERNAME", "")),
			password:    strings.TrimSpace(envOr("SMTP_PASSWORD", "")),
			fromAddress: smtpFrom,
		}
	}

	waEndpoint := strings.TrimSpace(envOr("WHATSAPP_API_URL", ""))
	if waEndpoint != "" {
		d.whatsapp = &whatsAppAdapter{
			endpoint:  waEndpoint,
			authToken: strings.TrimSpace(envOr("WHATSAPP_API_TOKEN", "")),
			defaultTo: strings.TrimSpace(envOr("WHATSAPP_DEFAULT_TO", "")),
			client: &http.Client{
				Timeout: time.Duration(envOrInt("WHATSAPP_TIMEOUT_SECONDS", 8)) * time.Second,
			},
		}
	}

	return d
}

func nullStringToString(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return strings.TrimSpace(v.String)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func sanitizeHeader(v string) string {
	clean := strings.ReplaceAll(v, "\r", "")
	clean = strings.ReplaceAll(clean, "\n", "")
	return strings.TrimSpace(clean)
}

func markFailure(ctx context.Context, tx pgx.Tx, row scheduleRow, dispatchErr error) error {
	nextRetry := row.RetryCount + 1
	lastErr := dispatchErr.Error()

	if nextRetry >= row.MaxRetries {
		payload, err := json.Marshal(deadLetterPayload{
			ScheduleID:     row.ID,
			CorrelationID:  row.CorrelationID,
			UserID:         row.UserID,
			EventID:        row.EventID,
			Title:          row.Title,
			Description:    row.Description,
			Timezone:       row.Timezone,
			StartAt:        row.StartAt.UTC().Format(time.RFC3339),
			EndAt:          row.EndAt.UTC().Format(time.RFC3339),
			TriggerAt:      row.TriggerAt.UTC().Format(time.RFC3339),
			OffsetMinutes:  row.OffsetMinutes,
			TargetChannels: mustUnmarshalChannels(row.TargetChannels),
			RetryCount:     nextRetry,
			MaxRetries:     row.MaxRetries,
		})
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO notification_dead_letter (schedule_id, correlation_id, payload, last_error)
			VALUES ($1, $2, $3::jsonb, $4)
		`, row.ID, row.CorrelationID, string(payload), lastErr); err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			UPDATE notification_schedule
			SET status = 'dead_letter',
			    retry_count = $2,
			    next_attempt_at = NULL,
			    last_error = $3,
			    updated_at = NOW()
			WHERE id = $1
		`, row.ID, nextRetry, lastErr)
		return err
	}

	backoff := time.Duration(1<<(nextRetry-1)) * time.Minute
	nextAttempt := time.Now().UTC().Add(backoff)
	_, err := tx.Exec(ctx, `
		UPDATE notification_schedule
		SET retry_count = $2,
		    next_attempt_at = $3,
		    last_error = $4,
		    updated_at = NOW()
		WHERE id = $1
	`, row.ID, nextRetry, nextAttempt, lastErr)
	return err
}

func mustUnmarshalChannels(raw []byte) []string {
	channels := []string{}
	_ = json.Unmarshal(raw, &channels)
	if len(channels) == 0 {
		return []string{"push"}
	}
	return channels
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

func parseSet(raw string) map[string]bool {
	m := map[string]bool{}
	for _, p := range strings.Split(raw, ",") {
		v := strings.ToLower(strings.TrimSpace(p))
		if v != "" {
			m[v] = true
		}
	}
	return m
}
