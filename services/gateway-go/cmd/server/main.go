package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type dependencyStatus struct {
	Name   string `json:"name"`
	Target string `json:"target"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type appContext struct {
	db               *pgxpool.Pool
	jwtSecret        []byte
	accessTTLMinutes int
	refreshTTLHours  int
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type profileUpdateRequest struct {
	Name string `json:"name"`
}

type preferencesUpdateRequest struct {
	ReminderIntensity string `json:"reminder_intensity"`
	PushEnabled       bool   `json:"push_enabled"`
	EmailEnabled      bool   `json:"email_enabled"`
	WhatsappEnabled   bool   `json:"whatsapp_enabled"`
}

type authClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

type recurrenceInput struct {
	Freq     string `json:"freq"`
	Interval int    `json:"interval"`
	Until    string `json:"until"`
}

type eventRequest struct {
	Title                  string          `json:"title"`
	Description            string          `json:"description"`
	StartAt                string          `json:"start_at"`
	EndAt                  string          `json:"end_at"`
	Timezone               string          `json:"timezone"`
	IsAllDay               bool            `json:"is_all_day"`
	Recurrence             recurrenceInput `json:"recurrence"`
	ReminderOffsetsMinutes []int           `json:"reminder_offsets_minutes"`
}

type eventRow struct {
	ID             int64
	UserID         int64
	Title          string
	Description    string
	StartAt        time.Time
	EndAt          time.Time
	Timezone       string
	IsAllDay       bool
	RecurrenceJSON []byte
	RemindersJSON  []byte
	CompletedAt    sql.NullTime
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type taskRequest struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Category        string   `json:"category"`
	Priority        string   `json:"priority"`
	DueAt           string   `json:"due_at"`
	ChecklistItems  []string `json:"checklist_titles"`
	DependsOnTaskID *int64   `json:"depends_on_task_id"`
}

type taskChecklistCreateRequest struct {
	Title string `json:"title"`
}

type taskChecklistUpdateRequest struct {
	IsDone bool `json:"is_done"`
}

type taskRow struct {
	ID              int64
	UserID          int64
	Title           string
	Description     string
	Category        string
	Priority        string
	DueAt           sql.NullTime
	DependsOnTaskID sql.NullInt64
	CompletedAt     sql.NullTime
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type taskChecklistRow struct {
	ID        int64
	TaskID    int64
	Title     string
	IsDone    bool
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type gamificationSummary struct {
	XP              int    `json:"xp"`
	Level           int    `json:"level"`
	CurrentStreak   int    `json:"current_streak"`
	LongestStreak   int    `json:"longest_streak"`
	LastActivityDay string `json:"last_activity_day,omitempty"`
}

type achievementRow struct {
	Key         string
	Title       string
	Description string
	UnlockedAt  time.Time
}

type completionResponse struct {
	Event      fiber.Map           `json:"event"`
	AwardedXP  int                 `json:"awarded_xp"`
	Summary    gamificationSummary `json:"summary"`
	Unlocked   []fiber.Map         `json:"unlocked"`
	WasAlready bool                `json:"was_already_completed"`
}

func main() {
	ctx := context.Background()
	db, err := connectDB(ctx)
	if err != nil {
		panic(err)
	}

	if err := runMigrations(ctx, db); err != nil {
		panic(err)
	}

	appCtx := &appContext{
		db:               db,
		jwtSecret:        []byte(envOr("JWT_SECRET", "change-me-in-production")),
		accessTTLMinutes: envOrInt("ACCESS_TOKEN_TTL_MINUTES", 30),
		refreshTTLHours:  envOrInt("REFRESH_TOKEN_TTL_HOURS", 720),
	}

	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
	}))

	app.Get("/api/v1/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "pong",
			"service": "gateway-go",
		})
	})

	app.Post("/api/v1/auth/register", appCtx.handleRegister)
	app.Post("/api/v1/auth/login", appCtx.handleLogin)
	app.Post("/api/v1/auth/refresh", appCtx.handleRefresh)
	app.Post("/api/v1/auth/forgot-password", appCtx.handleForgotPassword)
	app.Post("/api/v1/auth/reset-password", appCtx.handleResetPassword)

	secured := app.Group("/api/v1", appCtx.authMiddleware)
	secured.Get("/profile", appCtx.handleGetProfile)
	secured.Put("/profile", appCtx.handleUpdateProfile)
	secured.Get("/preferences", appCtx.handleGetPreferences)
	secured.Put("/preferences", appCtx.handleUpdatePreferences)
	secured.Post("/events", appCtx.handleCreateEvent)
	secured.Get("/events", appCtx.handleListEvents)
	secured.Get("/events/:id", appCtx.handleGetEvent)
	secured.Put("/events/:id", appCtx.handleUpdateEvent)
	secured.Delete("/events/:id", appCtx.handleDeleteEvent)
	secured.Post("/events/:id/complete", appCtx.handleCompleteEvent)
	secured.Post("/tasks", appCtx.handleCreateTask)
	secured.Get("/tasks", appCtx.handleListTasks)
	secured.Get("/tasks/:id", appCtx.handleGetTask)
	secured.Put("/tasks/:id", appCtx.handleUpdateTask)
	secured.Delete("/tasks/:id", appCtx.handleDeleteTask)
	secured.Post("/tasks/:id/checklist", appCtx.handleAddTaskChecklistItem)
	secured.Patch("/tasks/:id/checklist/:itemId", appCtx.handleUpdateTaskChecklistItem)
	secured.Delete("/tasks/:id/checklist/:itemId", appCtx.handleDeleteTaskChecklistItem)
	secured.Get("/gamification/summary", appCtx.handleGetGamificationSummary)
	secured.Get("/gamification/achievements", appCtx.handleGetAchievements)

	app.Get("/healthz", func(c *fiber.Ctx) error {
		checks := []dependencyStatus{
			checkTCP("postgres", envOr("POSTGRES_HOST", "localhost"), envOr("POSTGRES_PORT", "5432")),
			checkTCP("redis", envOr("REDIS_HOST", "localhost"), envOr("REDIS_PORT", "6379")),
			checkTCP("rabbitmq", envOr("RABBITMQ_HOST", "localhost"), envOr("RABBITMQ_PORT", "5672")),
			checkTCP("minio", envOr("MINIO_HOST", "localhost"), envOr("MINIO_PORT", "9000")),
			checkTCP("ai-service", "ai-service", envOr("AI_SERVICE_PORT", "8000")),
		}

		overall := "ok"
		for _, dep := range checks {
			if dep.Status != "ok" {
				overall = "degraded"
				break
			}
		}

		code := fiber.StatusOK
		if overall != "ok" {
			code = fiber.StatusServiceUnavailable
		}

		return c.Status(code).JSON(fiber.Map{
			"service":      "gateway-go",
			"status":       overall,
			"dependencies": checks,
		})
	})

	port := envOr("PORT", "8080")
	if err := app.Listen(":" + port); err != nil {
		panic(err)
	}
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

func runMigrations(ctx context.Context, db *pgxpool.Pool) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
            id BIGSERIAL PRIMARY KEY,
            email TEXT NOT NULL UNIQUE,
            password_hash TEXT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,
		`CREATE TABLE IF NOT EXISTS user_profiles (
            user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
            name TEXT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,
		`CREATE TABLE IF NOT EXISTS user_preferences (
            user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
            reminder_intensity TEXT NOT NULL DEFAULT 'medium',
            push_enabled BOOLEAN NOT NULL DEFAULT TRUE,
            email_enabled BOOLEAN NOT NULL DEFAULT TRUE,
            whatsapp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,
		`CREATE TABLE IF NOT EXISTS auth_sessions (
            id BIGSERIAL PRIMARY KEY,
            user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            refresh_token_hash TEXT NOT NULL UNIQUE,
            expires_at TIMESTAMPTZ NOT NULL,
            revoked_at TIMESTAMPTZ,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,
		`CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_status ON password_reset_tokens (user_id, expires_at) WHERE used_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS events (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			start_at TIMESTAMPTZ NOT NULL,
			end_at TIMESTAMPTZ NOT NULL,
			timezone TEXT NOT NULL DEFAULT 'UTC',
			is_all_day BOOLEAN NOT NULL DEFAULT FALSE,
			recurrence JSONB NOT NULL DEFAULT '{}'::jsonb,
			reminder_offsets JSONB NOT NULL DEFAULT '[60]'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_user_start ON events (user_id, start_at) WHERE deleted_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT 'Geral',
			priority TEXT NOT NULL DEFAULT 'medium',
			due_at TIMESTAMPTZ,
			depends_on_task_id BIGINT REFERENCES tasks(id) ON DELETE SET NULL,
			completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ
		)`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS depends_on_task_id BIGINT REFERENCES tasks(id) ON DELETE SET NULL`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_user_priority_due ON tasks (user_id, priority, due_at) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_user_dependency ON tasks (user_id, depends_on_task_id) WHERE deleted_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS task_checklist_items (
			id BIGSERIAL PRIMARY KEY,
			task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			is_done BOOLEAN NOT NULL DEFAULT FALSE,
			position INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_task_checklist_task_position ON task_checklist_items (task_id, position, id)`,
		`CREATE TABLE IF NOT EXISTS reminder_outbox (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			event_id BIGINT REFERENCES events(id) ON DELETE SET NULL,
			task_id BIGINT REFERENCES tasks(id) ON DELETE SET NULL,
			event_type TEXT NOT NULL,
			payload JSONB NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			processed_at TIMESTAMPTZ
		)`,
		`ALTER TABLE reminder_outbox ADD COLUMN IF NOT EXISTS task_id BIGINT REFERENCES tasks(id) ON DELETE SET NULL`,
		`CREATE INDEX IF NOT EXISTS idx_reminder_outbox_status_created ON reminder_outbox (status, created_at)`,
		`CREATE TABLE IF NOT EXISTS notification_schedule (
			id BIGSERIAL PRIMARY KEY,
			correlation_id TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'scheduled',
			topic TEXT NOT NULL,
			action TEXT NOT NULL,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			event_id BIGINT REFERENCES events(id) ON DELETE CASCADE,
			task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE,
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
		`ALTER TABLE notification_schedule ADD COLUMN IF NOT EXISTS task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE`,
		`ALTER TABLE notification_schedule ALTER COLUMN event_id DROP NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_notification_schedule_status_trigger ON notification_schedule (status, trigger_at)`,
		`CREATE INDEX IF NOT EXISTS idx_notification_schedule_task_status_trigger ON notification_schedule (task_id, status, trigger_at)`,
		`ALTER TABLE events ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ`,
		`CREATE TABLE IF NOT EXISTS user_gamification (
			user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			xp INTEGER NOT NULL DEFAULT 0,
			level INTEGER NOT NULL DEFAULT 1,
			current_streak INTEGER NOT NULL DEFAULT 0,
			longest_streak INTEGER NOT NULL DEFAULT 0,
			last_activity_day DATE,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS gamification_events (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			source TEXT NOT NULL,
			source_id BIGINT NOT NULL,
			xp_delta INTEGER NOT NULL,
			meta JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (user_id, source, source_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_achievements (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			achievement_key TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			unlocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (user_id, achievement_key)
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(ctx, q); err != nil {
			return err
		}
	}

	log.Println("database migrations applied")
	return nil
}

func (a *appContext) handleRegister(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := strings.TrimSpace(req.Password)

	if len(name) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name must be at least 2 characters"})
	}
	if !strings.Contains(email, "@") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid email"})
	}
	if len(password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "password must be at least 8 characters"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not hash password"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	var userID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		email, string(hash),
	).Scan(&userID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already in use"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create user"})
	}

	if _, err := tx.Exec(ctx, `INSERT INTO user_profiles (user_id, name) VALUES ($1, $2)`, userID, name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create profile"})
	}

	if _, err := tx.Exec(ctx, `INSERT INTO user_preferences (user_id) VALUES ($1)`, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create preferences"})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit transaction"})
	}

	accessToken, err := a.createAccessToken(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create access token"})
	}

	refreshToken, err := generateRandomToken(32)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create refresh token"})
	}

	if err := a.storeRefreshToken(c.UserContext(), userID, refreshToken); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not store refresh token"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user": fiber.Map{
			"id":    userID,
			"email": email,
			"name":  name,
		},
		"tokens": fiber.Map{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
	})
}

func (a *appContext) handleLogin(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := strings.TrimSpace(req.Password)

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	var userID int64
	var passwordHash, name string
	err := a.db.QueryRow(ctx,
		`SELECT u.id, u.password_hash, p.name
         FROM users u
         JOIN user_profiles p ON p.user_id = u.id
         WHERE u.email = $1`, email,
	).Scan(&userID, &passwordHash, &name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load user"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	accessToken, err := a.createAccessToken(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create access token"})
	}

	refreshToken, err := generateRandomToken(32)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create refresh token"})
	}

	if err := a.storeRefreshToken(c.UserContext(), userID, refreshToken); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not store refresh token"})
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":    userID,
			"email": email,
			"name":  name,
		},
		"tokens": fiber.Map{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
	})
}

func (a *appContext) handleRefresh(c *fiber.Ctx) error {
	var req refreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	rawToken := strings.TrimSpace(req.RefreshToken)
	if rawToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "refresh_token is required"})
	}

	hashed := hashToken(rawToken)
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	var userID int64
	err = tx.QueryRow(ctx,
		`SELECT user_id
         FROM auth_sessions
         WHERE refresh_token_hash = $1
           AND revoked_at IS NULL
           AND expires_at > NOW()`,
		hashed,
	).Scan(&userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid refresh token"})
	}

	if _, err := tx.Exec(ctx,
		`UPDATE auth_sessions SET revoked_at = NOW() WHERE refresh_token_hash = $1`,
		hashed,
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not revoke refresh token"})
	}

	newRefreshToken, err := generateRandomToken(32)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create refresh token"})
	}
	newHashed := hashToken(newRefreshToken)

	if _, err := tx.Exec(ctx,
		`INSERT INTO auth_sessions (user_id, refresh_token_hash, expires_at)
         VALUES ($1, $2, $3)`,
		userID,
		newHashed,
		time.Now().Add(time.Duration(a.refreshTTLHours)*time.Hour),
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not store new refresh token"})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit transaction"})
	}

	accessToken, err := a.createAccessToken(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create access token"})
	}

	return c.JSON(fiber.Map{
		"tokens": fiber.Map{
			"access_token":  accessToken,
			"refresh_token": newRefreshToken,
		},
	})
}

func (a *appContext) handleForgotPassword(c *fiber.Ctx) error {
	var req forgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !strings.Contains(email, "@") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid email"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	var userID int64
	err := a.db.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return c.JSON(fiber.Map{
				"message": "If the account exists, a password reset token has been issued.",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load user"})
	}

	rawToken, err := generateRandomToken(24)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create reset token"})
	}

	expiresAt := time.Now().Add(30 * time.Minute)
	if _, err := a.db.Exec(ctx,
		`UPDATE password_reset_tokens
         SET used_at = NOW()
         WHERE user_id = $1
           AND used_at IS NULL
           AND expires_at > NOW()`,
		userID,
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not revoke previous reset tokens"})
	}

	if _, err := a.db.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
         VALUES ($1, $2, $3)`,
		userID,
		hashToken(rawToken),
		expiresAt,
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not store reset token"})
	}

	return c.JSON(fiber.Map{
		"message":       "Password reset token issued.",
		"reset_token":   rawToken,
		"expires_at":    expiresAt.UTC().Format(time.RFC3339),
		"delivery_mode": "development",
	})
}

func (a *appContext) handleResetPassword(c *fiber.Ctx) error {
	var req resetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	rawToken := strings.TrimSpace(req.Token)
	password := strings.TrimSpace(req.Password)
	if rawToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token is required"})
	}
	if len(password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "password must be at least 8 characters"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not hash password"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	var userID int64
	err = tx.QueryRow(ctx,
		`SELECT user_id
         FROM password_reset_tokens
         WHERE token_hash = $1
           AND used_at IS NULL
           AND expires_at > NOW()
         FOR UPDATE`,
		hashToken(rawToken),
	).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired reset token"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load reset token"})
	}

	if _, err := tx.Exec(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`,
		string(hash),
		userID,
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not update password"})
	}

	if _, err := tx.Exec(ctx,
		`UPDATE password_reset_tokens
         SET used_at = NOW()
         WHERE user_id = $1
           AND used_at IS NULL`,
		userID,
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not consume reset token"})
	}

	if _, err := tx.Exec(ctx,
		`UPDATE auth_sessions
         SET revoked_at = NOW()
         WHERE user_id = $1
           AND revoked_at IS NULL`,
		userID,
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not revoke previous sessions"})
	}

	var email, name string
	err = tx.QueryRow(ctx,
		`SELECT u.email, p.name
         FROM users u
         JOIN user_profiles p ON p.user_id = u.id
         WHERE u.id = $1`,
		userID,
	).Scan(&email, &name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load updated user"})
	}

	refreshToken, err := generateRandomToken(32)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create refresh token"})
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO auth_sessions (user_id, refresh_token_hash, expires_at)
         VALUES ($1, $2, $3)`,
		userID,
		hashToken(refreshToken),
		time.Now().Add(time.Duration(a.refreshTTLHours)*time.Hour),
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not store refresh token"})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit transaction"})
	}

	accessToken, err := a.createAccessToken(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create access token"})
	}

	return c.JSON(fiber.Map{
		"message": "Password reset successfully.",
		"user": fiber.Map{
			"id":    userID,
			"email": email,
			"name":  name,
		},
		"tokens": fiber.Map{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
	})
}

func (a *appContext) handleGetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	var email, name string
	err := a.db.QueryRow(ctx,
		`SELECT u.email, p.name
         FROM users u
         JOIN user_profiles p ON p.user_id = u.id
         WHERE u.id = $1`,
		userID,
	).Scan(&email, &name)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	return c.JSON(fiber.Map{
		"id":    userID,
		"email": email,
		"name":  name,
	})
}

func (a *appContext) handleUpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	var req profileUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	name := strings.TrimSpace(req.Name)
	if len(name) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name must be at least 2 characters"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	res, err := a.db.Exec(ctx,
		`UPDATE user_profiles SET name = $1, updated_at = NOW() WHERE user_id = $2`,
		name,
		userID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not update profile"})
	}
	if res.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "profile not found"})
	}

	return c.JSON(fiber.Map{"name": name})
}

func (a *appContext) handleGetPreferences(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	var prefs preferencesUpdateRequest
	err := a.db.QueryRow(ctx,
		`SELECT reminder_intensity, push_enabled, email_enabled, whatsapp_enabled
         FROM user_preferences
         WHERE user_id = $1`,
		userID,
	).Scan(&prefs.ReminderIntensity, &prefs.PushEnabled, &prefs.EmailEnabled, &prefs.WhatsappEnabled)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "preferences not found"})
	}

	return c.JSON(prefs)
}

func (a *appContext) handleUpdatePreferences(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	var req preferencesUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	intensity := strings.ToLower(strings.TrimSpace(req.ReminderIntensity))
	if intensity != "low" && intensity != "medium" && intensity != "high" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "reminder_intensity must be low, medium or high"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	res, err := a.db.Exec(ctx,
		`UPDATE user_preferences
         SET reminder_intensity = $1,
             push_enabled = $2,
             email_enabled = $3,
             whatsapp_enabled = $4,
             updated_at = NOW()
         WHERE user_id = $5`,
		intensity,
		req.PushEnabled,
		req.EmailEnabled,
		req.WhatsappEnabled,
		userID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not update preferences"})
	}
	if res.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "preferences not found"})
	}

	return c.JSON(fiber.Map{
		"reminder_intensity": intensity,
		"push_enabled":       req.PushEnabled,
		"email_enabled":      req.EmailEnabled,
		"whatsapp_enabled":   req.WhatsappEnabled,
	})
}

func (a *appContext) handleCreateEvent(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	var req eventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	startAt, endAt, recurrenceJSON, reminderJSON, err := validateEventRequest(req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	tz := strings.TrimSpace(req.Timezone)
	if tz == "" {
		tz = "UTC"
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	var row eventRow
	err = tx.QueryRow(ctx, `
		INSERT INTO events (user_id, title, description, start_at, end_at, timezone, is_all_day, recurrence, reminder_offsets)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb)
		RETURNING id, user_id, title, description, start_at, end_at, timezone, is_all_day, recurrence, reminder_offsets, completed_at, created_at, updated_at
	`, userID, strings.TrimSpace(req.Title), strings.TrimSpace(req.Description), startAt, endAt, tz, req.IsAllDay, string(recurrenceJSON), string(reminderJSON)).
		Scan(&row.ID, &row.UserID, &row.Title, &row.Description, &row.StartAt, &row.EndAt, &row.Timezone, &row.IsAllDay, &row.RecurrenceJSON, &row.RemindersJSON, &row.CompletedAt, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create event"})
	}

	if err := a.enqueueReminderContract(ctx, tx, row.UserID, row.ID, "event.created", row); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not enqueue reminder contract"})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit event creation"})
	}

	return c.Status(fiber.StatusCreated).JSON(eventRowToResponse(row))
}

func (a *appContext) handleListEvents(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), 10)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	fromStr := strings.TrimSpace(c.Query("from"))
	toStr := strings.TrimSpace(c.Query("to"))

	var (
		fromFilter *time.Time
		toFilter   *time.Time
	)

	if fromStr != "" {
		fromAt, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid from date, use RFC3339"})
		}
		fromFilter = &fromAt
	}
	if toStr != "" {
		toAt, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid to date, use RFC3339"})
		}
		toFilter = &toAt
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	whereClause := ` WHERE user_id = $1 AND deleted_at IS NULL `
	args := []interface{}{userID}
	idx := 2

	if fromFilter != nil {
		whereClause += fmt.Sprintf(" AND end_at >= $%d", idx)
		args = append(args, *fromFilter)
		idx++
	}
	if toFilter != nil {
		whereClause += fmt.Sprintf(" AND start_at <= $%d", idx)
		args = append(args, *toFilter)
		idx++
	}

	var total int
	err := a.db.QueryRow(ctx, "SELECT COUNT(*) FROM events"+whereClause, args...).Scan(&total)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not count events"})
	}

	query := `
		SELECT id, user_id, title, description, start_at, end_at, timezone, is_all_day, recurrence, reminder_offsets, completed_at, created_at, updated_at
		FROM events
	` + whereClause

	query += fmt.Sprintf(" ORDER BY start_at ASC, id ASC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, limit, offset)

	rows, err := a.db.Query(ctx, query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not list events"})
	}
	defer rows.Close()

	items := make([]fiber.Map, 0)
	for rows.Next() {
		var row eventRow
		if err := rows.Scan(&row.ID, &row.UserID, &row.Title, &row.Description, &row.StartAt, &row.EndAt, &row.Timezone, &row.IsAllDay, &row.RecurrenceJSON, &row.RemindersJSON, &row.CompletedAt, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not parse event row"})
		}
		items = append(items, eventRowToResponse(row))
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return c.JSON(fiber.Map{
		"events": items,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (a *appContext) handleGetEvent(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	eventID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || eventID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid event id"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	var row eventRow
	err = a.db.QueryRow(ctx, `
		SELECT id, user_id, title, description, start_at, end_at, timezone, is_all_day, recurrence, reminder_offsets, completed_at, created_at, updated_at
		FROM events
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`, eventID, userID).
		Scan(&row.ID, &row.UserID, &row.Title, &row.Description, &row.StartAt, &row.EndAt, &row.Timezone, &row.IsAllDay, &row.RecurrenceJSON, &row.RemindersJSON, &row.CompletedAt, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
	}

	return c.JSON(eventRowToResponse(row))
}

func (a *appContext) handleUpdateEvent(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	eventID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || eventID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid event id"})
	}

	var req eventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	startAt, endAt, recurrenceJSON, reminderJSON, err := validateEventRequest(req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	tz := strings.TrimSpace(req.Timezone)
	if tz == "" {
		tz = "UTC"
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	var row eventRow
	err = tx.QueryRow(ctx, `
		UPDATE events
		SET title = $1,
		    description = $2,
		    start_at = $3,
		    end_at = $4,
		    timezone = $5,
		    is_all_day = $6,
		    recurrence = $7::jsonb,
		    reminder_offsets = $8::jsonb,
		    updated_at = NOW()
		WHERE id = $9 AND user_id = $10 AND deleted_at IS NULL
		RETURNING id, user_id, title, description, start_at, end_at, timezone, is_all_day, recurrence, reminder_offsets, completed_at, created_at, updated_at
	`, strings.TrimSpace(req.Title), strings.TrimSpace(req.Description), startAt, endAt, tz, req.IsAllDay, string(recurrenceJSON), string(reminderJSON), eventID, userID).
		Scan(&row.ID, &row.UserID, &row.Title, &row.Description, &row.StartAt, &row.EndAt, &row.Timezone, &row.IsAllDay, &row.RecurrenceJSON, &row.RemindersJSON, &row.CompletedAt, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
	}

	if err := a.enqueueReminderContract(ctx, tx, row.UserID, row.ID, "event.updated", row); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not enqueue reminder contract"})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit event update"})
	}

	return c.JSON(eventRowToResponse(row))
}

func (a *appContext) handleDeleteEvent(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	eventID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || eventID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid event id"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	var row eventRow
	err = tx.QueryRow(ctx, `
		UPDATE events
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		RETURNING id, user_id, title, description, start_at, end_at, timezone, is_all_day, recurrence, reminder_offsets, completed_at, created_at, updated_at
	`, eventID, userID).
		Scan(&row.ID, &row.UserID, &row.Title, &row.Description, &row.StartAt, &row.EndAt, &row.Timezone, &row.IsAllDay, &row.RecurrenceJSON, &row.RemindersJSON, &row.CompletedAt, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
	}

	if err := a.enqueueReminderContract(ctx, tx, row.UserID, row.ID, "event.deleted", row); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not enqueue reminder contract"})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit event deletion"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (a *appContext) handleCompleteEvent(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	eventID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || eventID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid event id"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 8*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	var row eventRow
	err = tx.QueryRow(ctx, `
		UPDATE events
		SET completed_at = COALESCE(completed_at, NOW()),
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		RETURNING id, user_id, title, description, start_at, end_at, timezone, is_all_day, recurrence, reminder_offsets, completed_at, created_at, updated_at
	`, eventID, userID).
		Scan(&row.ID, &row.UserID, &row.Title, &row.Description, &row.StartAt, &row.EndAt, &row.Timezone, &row.IsAllDay, &row.RecurrenceJSON, &row.RemindersJSON, &row.CompletedAt, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "event not found"})
	}

	awardedXP := 0
	unlocked := []fiber.Map{}
	if row.CompletedAt.Valid {
		summary, xp, unlockedRows, err := a.applyEventCompletionGamification(ctx, tx, userID, row)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not update gamification"})
		}
		awardedXP = xp
		unlocked = achievementRowsToResponse(unlockedRows)

		if err := tx.Commit(ctx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit completion"})
		}

		return c.JSON(completionResponse{
			Event:      eventRowToResponse(row),
			AwardedXP:  awardedXP,
			Summary:    summary,
			Unlocked:   unlocked,
			WasAlready: awardedXP == 0,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit completion"})
	}

	summary, err := a.loadGamificationSummary(ctx, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load gamification summary"})
	}

	return c.JSON(completionResponse{
		Event:      eventRowToResponse(row),
		AwardedXP:  0,
		Summary:    summary,
		Unlocked:   []fiber.Map{},
		WasAlready: true,
	})
}

func (a *appContext) handleCreateTask(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)

	var req taskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	title, description, category, priority, dueAt, checklistItems, dependsOnTaskID, err := validateTaskRequest(req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	if dependsOnTaskID != nil {
		var exists bool
		err = tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM tasks
				WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
			)
		`, *dependsOnTaskID, userID).Scan(&exists)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not validate task dependency"})
		}
		if !exists {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "depends_on_task_id does not reference a valid task"})
		}
	}

	var row taskRow
	err = tx.QueryRow(ctx, `
		INSERT INTO tasks (user_id, title, description, category, priority, due_at, depends_on_task_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, title, description, category, priority, due_at, depends_on_task_id, completed_at, created_at, updated_at
	`, userID, title, description, category, priority, dueAt, dependsOnTaskID).
		Scan(&row.ID, &row.UserID, &row.Title, &row.Description, &row.Category, &row.Priority, &row.DueAt, &row.DependsOnTaskID, &row.CompletedAt, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create task"})
	}

	for idx, itemTitle := range checklistItems {
		if _, err := tx.Exec(ctx, `
			INSERT INTO task_checklist_items (task_id, title, position)
			VALUES ($1, $2, $3)
		`, row.ID, itemTitle, idx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create checklist items"})
		}
	}

	if err := a.refreshTaskCompletion(ctx, tx, row.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not calculate task completion"})
	}

	if row.DueAt.Valid {
		if err := a.enqueueTaskReminderContract(ctx, tx, row, "task.created"); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not enqueue task reminder contract"})
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit task creation"})
	}

	response, err := a.loadTaskResponse(ctx, userID, row.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load task"})
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

func (a *appContext) handleListTasks(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), 20)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	var total int
	err := a.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM tasks
		WHERE user_id = $1 AND deleted_at IS NULL
	`, userID).Scan(&total)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not count tasks"})
	}

	rows, err := a.db.Query(ctx, `
		SELECT id, user_id, title, description, category, priority, due_at, depends_on_task_id, completed_at, created_at, updated_at
		FROM tasks
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY
			CASE priority
				WHEN 'urgent' THEN 4
				WHEN 'high' THEN 3
				WHEN 'medium' THEN 2
				ELSE 1
			END DESC,
			due_at ASC NULLS LAST,
			created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not list tasks"})
	}
	defer rows.Close()

	items := make([]fiber.Map, 0)
	for rows.Next() {
		var row taskRow
		if err := rows.Scan(&row.ID, &row.UserID, &row.Title, &row.Description, &row.Category, &row.Priority, &row.DueAt, &row.DependsOnTaskID, &row.CompletedAt, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not parse task row"})
		}
		taskResp, err := a.loadTaskResponse(ctx, userID, row.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load checklist"})
		}
		items = append(items, taskResp)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return c.JSON(fiber.Map{
		"tasks": items,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (a *appContext) handleGetTask(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	taskID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || taskID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid task id"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	response, err := a.loadTaskResponse(ctx, userID, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load task"})
	}

	return c.JSON(response)
}

func (a *appContext) handleUpdateTask(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	taskID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || taskID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid task id"})
	}

	var req taskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	title, description, category, priority, dueAt, _, dependsOnTaskID, err := validateTaskRequest(req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if dependsOnTaskID != nil && *dependsOnTaskID == taskID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "depends_on_task_id cannot reference the same task"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	var previousDueAt sql.NullTime
	err = tx.QueryRow(ctx, `
		SELECT due_at
		FROM tasks
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`, taskID, userID).Scan(&previousDueAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load current task"})
	}

	if dependsOnTaskID != nil {
		var exists bool
		err = tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM tasks
				WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
			)
		`, *dependsOnTaskID, userID).Scan(&exists)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not validate task dependency"})
		}
		if !exists {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "depends_on_task_id does not reference a valid task"})
		}
	}

	var row taskRow
	err = tx.QueryRow(ctx, `
		UPDATE tasks
		SET title = $1,
			description = $2,
			category = $3,
			priority = $4,
			due_at = $5,
			depends_on_task_id = $6,
			updated_at = NOW()
		WHERE id = $7 AND user_id = $8 AND deleted_at IS NULL
		RETURNING id, user_id, title, description, category, priority, due_at, depends_on_task_id, completed_at, created_at, updated_at
	`, title, description, category, priority, dueAt, dependsOnTaskID, taskID, userID).
		Scan(&row.ID, &row.UserID, &row.Title, &row.Description, &row.Category, &row.Priority, &row.DueAt, &row.DependsOnTaskID, &row.CompletedAt, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not update task"})
	}

	if row.DueAt.Valid {
		if err := a.enqueueTaskReminderContract(ctx, tx, row, "task.deleted"); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not enqueue task reminder contract"})
		}
		if err := a.enqueueTaskReminderContract(ctx, tx, row, "task.updated"); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not enqueue task reminder contract"})
		}
	} else if previousDueAt.Valid {
		if err := a.enqueueTaskReminderContract(ctx, tx, row, "task.deleted"); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not enqueue task reminder contract"})
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit task update"})
	}

	response, err := a.loadTaskResponse(ctx, userID, taskID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load task"})
	}

	return c.JSON(response)
}

func (a *appContext) handleDeleteTask(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	taskID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || taskID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid task id"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	var row taskRow
	err = tx.QueryRow(ctx, `
		UPDATE tasks
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		RETURNING id, user_id, title, description, category, priority, due_at, depends_on_task_id, completed_at, created_at, updated_at
	`, taskID, userID).
		Scan(&row.ID, &row.UserID, &row.Title, &row.Description, &row.Category, &row.Priority, &row.DueAt, &row.DependsOnTaskID, &row.CompletedAt, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not delete task"})
	}

	if err := a.enqueueTaskReminderContract(ctx, tx, row, "task.deleted"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not enqueue task reminder contract"})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit task deletion"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (a *appContext) handleAddTaskChecklistItem(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	taskID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || taskID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid task id"})
	}

	var req taskChecklistCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	title := strings.TrimSpace(req.Title)
	if len(title) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title must be at least 2 characters"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM tasks
			WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
		)
	`, taskID, userID).Scan(&exists)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not validate task"})
	}
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
	}

	var nextPosition int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(position), -1) + 1
		FROM task_checklist_items
		WHERE task_id = $1
	`, taskID).Scan(&nextPosition)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not calculate checklist position"})
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO task_checklist_items (task_id, title, position)
		VALUES ($1, $2, $3)
	`, taskID, title, nextPosition); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create checklist item"})
	}

	if err := a.refreshTaskCompletion(ctx, tx, taskID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not calculate task completion"})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit checklist creation"})
	}

	response, err := a.loadTaskResponse(ctx, userID, taskID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load task"})
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

func (a *appContext) handleUpdateTaskChecklistItem(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	taskID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || taskID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid task id"})
	}
	itemID, err := strconv.ParseInt(c.Params("itemId"), 10, 64)
	if err != nil || itemID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid checklist item id"})
	}

	var req taskChecklistUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	res, err := tx.Exec(ctx, `
		UPDATE task_checklist_items AS i
		SET is_done = $1,
			updated_at = NOW()
		FROM tasks t
		WHERE i.id = $2
			AND i.task_id = $3
			AND t.id = i.task_id
			AND t.user_id = $4
			AND t.deleted_at IS NULL
	`, req.IsDone, itemID, taskID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not update checklist item"})
	}
	if res.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "checklist item not found"})
	}

	if err := a.refreshTaskCompletion(ctx, tx, taskID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not calculate task completion"})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit checklist update"})
	}

	response, err := a.loadTaskResponse(ctx, userID, taskID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load task"})
	}

	return c.JSON(response)
}

func (a *appContext) handleDeleteTaskChecklistItem(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	taskID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || taskID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid task id"})
	}
	itemID, err := strconv.ParseInt(c.Params("itemId"), 10, 64)
	if err != nil || itemID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid checklist item id"})
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	tx, err := a.db.Begin(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start transaction"})
	}
	defer tx.Rollback(ctx)

	res, err := tx.Exec(ctx, `
		DELETE FROM task_checklist_items AS i
		USING tasks t
		WHERE i.id = $1
			AND i.task_id = $2
			AND t.id = i.task_id
			AND t.user_id = $3
			AND t.deleted_at IS NULL
	`, itemID, taskID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not delete checklist item"})
	}
	if res.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "checklist item not found"})
	}

	if err := a.refreshTaskCompletion(ctx, tx, taskID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not calculate task completion"})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not commit checklist deletion"})
	}

	response, err := a.loadTaskResponse(ctx, userID, taskID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load task"})
	}

	return c.JSON(response)
}

func validateTaskRequest(req taskRequest) (string, string, string, string, interface{}, []string, *int64, error) {
	title := strings.TrimSpace(req.Title)
	if len(title) < 3 {
		return "", "", "", "", nil, nil, nil, errors.New("title must be at least 3 characters")
	}

	description := strings.TrimSpace(req.Description)
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Geral"
	}

	priority := strings.ToLower(strings.TrimSpace(req.Priority))
	if priority == "" {
		priority = "medium"
	}
	if priority != "low" && priority != "medium" && priority != "high" && priority != "urgent" {
		return "", "", "", "", nil, nil, nil, errors.New("priority must be low, medium, high or urgent")
	}

	var dueAt interface{} = nil
	dueText := strings.TrimSpace(req.DueAt)
	if dueText != "" {
		parsed, err := time.Parse(time.RFC3339, dueText)
		if err != nil {
			return "", "", "", "", nil, nil, nil, errors.New("due_at must be RFC3339")
		}
		dueAt = parsed.UTC()
	}

	var dependsOnTaskID *int64
	if req.DependsOnTaskID != nil {
		if *req.DependsOnTaskID <= 0 {
			return "", "", "", "", nil, nil, nil, errors.New("depends_on_task_id must be a positive integer")
		}
		dependsOnTaskID = req.DependsOnTaskID
	}

	items := make([]string, 0, len(req.ChecklistItems))
	for _, item := range req.ChecklistItems {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		items = append(items, trimmed)
	}

	return title, description, category, priority, dueAt, items, dependsOnTaskID, nil
}

func (a *appContext) refreshTaskCompletion(ctx context.Context, tx pgx.Tx, taskID int64) error {
	var total int
	var done int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE is_done)
		FROM task_checklist_items
		WHERE task_id = $1
	`, taskID).Scan(&total, &done); err != nil {
		return err
	}

	if total > 0 && done == total {
		_, err := tx.Exec(ctx, `
			UPDATE tasks
			SET completed_at = COALESCE(completed_at, NOW()), updated_at = NOW()
			WHERE id = $1
		`, taskID)
		return err
	}

	_, err := tx.Exec(ctx, `
		UPDATE tasks
		SET completed_at = NULL, updated_at = NOW()
		WHERE id = $1
	`, taskID)
	return err
}

func (a *appContext) loadTaskResponse(ctx context.Context, userID int64, taskID int64) (fiber.Map, error) {
	var row taskRow
	err := a.db.QueryRow(ctx, `
		SELECT id, user_id, title, description, category, priority, due_at, depends_on_task_id, completed_at, created_at, updated_at
		FROM tasks
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`, taskID, userID).
		Scan(&row.ID, &row.UserID, &row.Title, &row.Description, &row.Category, &row.Priority, &row.DueAt, &row.DependsOnTaskID, &row.CompletedAt, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return nil, errors.New("task not found")
		}
		return nil, err
	}

	checklistRows, err := a.loadTaskChecklist(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	isBlocked, err := a.isTaskBlocked(ctx, userID, row)
	if err != nil {
		return nil, err
	}

	nextReminders, err := a.loadTaskNextReminders(ctx, row.ID)
	if err != nil {
		return nil, err
	}

	return taskRowToResponse(row, checklistRows, isBlocked, nextReminders), nil
}

func (a *appContext) isTaskBlocked(ctx context.Context, userID int64, row taskRow) (bool, error) {
	if !row.DependsOnTaskID.Valid {
		return false, nil
	}

	var predecessorCompletedAt sql.NullTime
	err := a.db.QueryRow(ctx, `
		SELECT completed_at
		FROM tasks
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`, row.DependsOnTaskID.Int64, userID).Scan(&predecessorCompletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return false, nil
		}
		return false, err
	}

	return !predecessorCompletedAt.Valid, nil
}

func (a *appContext) loadTaskChecklist(ctx context.Context, taskID int64) ([]taskChecklistRow, error) {
	rows, err := a.db.Query(ctx, `
		SELECT id, task_id, title, is_done, position, created_at, updated_at
		FROM task_checklist_items
		WHERE task_id = $1
		ORDER BY position ASC, id ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]taskChecklistRow, 0)
	for rows.Next() {
		var row taskChecklistRow
		if err := rows.Scan(&row.ID, &row.TaskID, &row.Title, &row.IsDone, &row.Position, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, row)
	}

	return items, nil
}

func (a *appContext) loadTaskNextReminders(ctx context.Context, taskID int64) ([]fiber.Map, error) {
	rows, err := a.db.Query(ctx, `
		SELECT id, trigger_at, offset_minutes, target_channels, status
		FROM notification_schedule
		WHERE task_id = $1
		  AND canceled_at IS NULL
		  AND status IN ('scheduled', 'queued')
		ORDER BY trigger_at ASC
		LIMIT 3
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]fiber.Map, 0)
	for rows.Next() {
		var (
			id           int64
			triggerAt    time.Time
			offsetMinute int
			channelsJSON []byte
			status       string
		)
		if err := rows.Scan(&id, &triggerAt, &offsetMinute, &channelsJSON, &status); err != nil {
			return nil, err
		}

		channels := []string{}
		_ = json.Unmarshal(channelsJSON, &channels)

		items = append(items, fiber.Map{
			"id":             id,
			"trigger_at":     triggerAt.UTC().Format(time.RFC3339),
			"offset_minutes": offsetMinute,
			"channels":       channels,
			"status":         status,
		})
	}

	return items, nil
}

func taskRowToResponse(row taskRow, checklist []taskChecklistRow, isBlocked bool, nextReminders []fiber.Map) fiber.Map {
	items := make([]fiber.Map, 0, len(checklist))
	doneCount := 0
	for _, item := range checklist {
		if item.IsDone {
			doneCount++
		}
		items = append(items, fiber.Map{
			"id":         item.ID,
			"task_id":    item.TaskID,
			"title":      item.Title,
			"is_done":    item.IsDone,
			"position":   item.Position,
			"created_at": item.CreatedAt.UTC().Format(time.RFC3339),
			"updated_at": item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	progressPercent := 0
	if len(checklist) > 0 {
		progressPercent = int(float64(doneCount) / float64(len(checklist)) * 100.0)
	}

	return fiber.Map{
		"id":                 row.ID,
		"user_id":            row.UserID,
		"title":              row.Title,
		"description":        row.Description,
		"category":           row.Category,
		"priority":           row.Priority,
		"due_at":             nullableTimeRFC3339(row.DueAt),
		"depends_on_task_id": nullableInt64(row.DependsOnTaskID),
		"is_blocked":         isBlocked,
		"completed_at":       nullableTimeRFC3339(row.CompletedAt),
		"checklist":          items,
		"progress_percent":   progressPercent,
		"next_reminders":     nextReminders,
		"created_at":         row.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":         row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (a *appContext) handleGetGamificationSummary(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	if err := a.ensureGamificationProfile(ctx, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not initialize gamification"})
	}

	summary, err := a.loadGamificationSummary(ctx, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load gamification summary"})
	}

	return c.JSON(summary)
}

func (a *appContext) handleGetAchievements(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
	defer cancel()

	rows, err := a.db.Query(ctx, `
		SELECT achievement_key, title, description, unlocked_at
		FROM user_achievements
		WHERE user_id = $1
		ORDER BY unlocked_at DESC
	`, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load achievements"})
	}
	defer rows.Close()

	achievements := make([]fiber.Map, 0)
	for rows.Next() {
		var r achievementRow
		if err := rows.Scan(&r.Key, &r.Title, &r.Description, &r.UnlockedAt); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not parse achievements"})
		}
		achievements = append(achievements, fiber.Map{
			"key":         r.Key,
			"title":       r.Title,
			"description": r.Description,
			"unlocked_at": r.UnlockedAt.UTC().Format(time.RFC3339),
		})
	}

	return c.JSON(fiber.Map{"achievements": achievements})
}

func validateEventRequest(req eventRequest) (time.Time, time.Time, []byte, []byte, error) {
	title := strings.TrimSpace(req.Title)
	if len(title) < 3 {
		return time.Time{}, time.Time{}, nil, nil, errors.New("title must be at least 3 characters")
	}

	startAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartAt))
	if err != nil {
		return time.Time{}, time.Time{}, nil, nil, errors.New("start_at must be RFC3339")
	}

	endAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EndAt))
	if err != nil {
		return time.Time{}, time.Time{}, nil, nil, errors.New("end_at must be RFC3339")
	}

	if !endAt.After(startAt) {
		return time.Time{}, time.Time{}, nil, nil, errors.New("end_at must be after start_at")
	}

	recurrence := map[string]interface{}{}
	freq := strings.ToLower(strings.TrimSpace(req.Recurrence.Freq))
	if freq != "" {
		if freq != "daily" && freq != "weekly" && freq != "monthly" {
			return time.Time{}, time.Time{}, nil, nil, errors.New("recurrence.freq must be daily, weekly or monthly")
		}
		interval := req.Recurrence.Interval
		if interval <= 0 {
			interval = 1
		}
		recurrence["freq"] = freq
		recurrence["interval"] = interval

		if strings.TrimSpace(req.Recurrence.Until) != "" {
			untilAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.Recurrence.Until))
			if err != nil {
				return time.Time{}, time.Time{}, nil, nil, errors.New("recurrence.until must be RFC3339")
			}
			if untilAt.Before(startAt) {
				return time.Time{}, time.Time{}, nil, nil, errors.New("recurrence.until must be after start_at")
			}
			recurrence["until"] = untilAt.UTC().Format(time.RFC3339)
		}
	}

	recurrenceJSON, err := json.Marshal(recurrence)
	if err != nil {
		return time.Time{}, time.Time{}, nil, nil, errors.New("invalid recurrence")
	}

	offsets := req.ReminderOffsetsMinutes
	if len(offsets) == 0 {
		offsets = []int{60}
	}
	clean := make([]int, 0, len(offsets))
	seen := map[int]bool{}
	for _, m := range offsets {
		if m <= 0 || m > 60*24*30 {
			return time.Time{}, time.Time{}, nil, nil, errors.New("reminder_offsets_minutes must be between 1 and 43200")
		}
		if !seen[m] {
			clean = append(clean, m)
			seen[m] = true
		}
	}
	sort.Ints(clean)

	reminderJSON, err := json.Marshal(clean)
	if err != nil {
		return time.Time{}, time.Time{}, nil, nil, errors.New("invalid reminder_offsets_minutes")
	}

	return startAt.UTC(), endAt.UTC(), recurrenceJSON, reminderJSON, nil
}

func eventRowToResponse(row eventRow) fiber.Map {
	recurrence := fiber.Map{}
	_ = json.Unmarshal(row.RecurrenceJSON, &recurrence)

	offsets := []int{}
	_ = json.Unmarshal(row.RemindersJSON, &offsets)

	return fiber.Map{
		"id":                       row.ID,
		"user_id":                  row.UserID,
		"title":                    row.Title,
		"description":              row.Description,
		"start_at":                 row.StartAt.UTC().Format(time.RFC3339),
		"end_at":                   row.EndAt.UTC().Format(time.RFC3339),
		"timezone":                 row.Timezone,
		"is_all_day":               row.IsAllDay,
		"completed_at":             nullableTimeRFC3339(row.CompletedAt),
		"recurrence":               recurrence,
		"reminder_offsets_minutes": offsets,
		"created_at":               row.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":               row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (a *appContext) ensureGamificationProfile(ctx context.Context, userID int64) error {
	_, err := a.db.Exec(ctx, `
		INSERT INTO user_gamification (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	return err
}

func (a *appContext) loadGamificationSummary(ctx context.Context, userID int64) (gamificationSummary, error) {
	var (
		summary gamificationSummary
		lastDay sql.NullTime
	)
	err := a.db.QueryRow(ctx, `
		SELECT xp, level, current_streak, longest_streak, last_activity_day::timestamptz
		FROM user_gamification
		WHERE user_id = $1
	`, userID).Scan(&summary.XP, &summary.Level, &summary.CurrentStreak, &summary.LongestStreak, &lastDay)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return gamificationSummary{XP: 0, Level: 1, CurrentStreak: 0, LongestStreak: 0}, nil
		}
		return gamificationSummary{}, err
	}

	if lastDay.Valid {
		summary.LastActivityDay = lastDay.Time.UTC().Format("2006-01-02")
	}

	return summary, nil
}

func (a *appContext) applyEventCompletionGamification(ctx context.Context, tx pgx.Tx, userID int64, row eventRow) (gamificationSummary, int, []achievementRow, error) {
	if err := ensureGamificationProfileTx(ctx, tx, userID); err != nil {
		return gamificationSummary{}, 0, nil, err
	}

	completedAt := time.Now().UTC()
	if row.CompletedAt.Valid {
		completedAt = row.CompletedAt.Time.UTC()
	}

	awardedXP := 50
	if completedAt.Before(row.EndAt.UTC()) || completedAt.Equal(row.EndAt.UTC()) {
		awardedXP += 25
	}

	var eventID int64
	err := tx.QueryRow(ctx, `
		INSERT INTO gamification_events (user_id, source, source_id, xp_delta, meta)
		VALUES ($1, 'event.completed', $2, $3, $4::jsonb)
		ON CONFLICT (user_id, source, source_id) DO NOTHING
		RETURNING id
	`, userID, row.ID, awardedXP, `{"kind":"event_completion"}`).Scan(&eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			summary, sErr := loadGamificationSummaryTx(ctx, tx, userID)
			return summary, 0, []achievementRow{}, sErr
		}
		return gamificationSummary{}, 0, nil, err
	}

	var (
		xp            int
		level         int
		currentStreak int
		longestStreak int
		lastDay       sql.NullTime
	)
	err = tx.QueryRow(ctx, `
		SELECT xp, level, current_streak, longest_streak, last_activity_day::timestamptz
		FROM user_gamification
		WHERE user_id = $1
		FOR UPDATE
	`, userID).Scan(&xp, &level, &currentStreak, &longestStreak, &lastDay)
	if err != nil {
		return gamificationSummary{}, 0, nil, err
	}

	activityDay := completedAt.Truncate(24 * time.Hour)
	if lastDay.Valid {
		lastActivity := lastDay.Time.UTC().Truncate(24 * time.Hour)
		diffDays := int(activityDay.Sub(lastActivity).Hours() / 24)
		switch {
		case diffDays <= 0:
			// same day or older; keep current streak unchanged
		case diffDays == 1:
			currentStreak++
		default:
			currentStreak = 1
		}
	} else {
		currentStreak = 1
	}

	if currentStreak > longestStreak {
		longestStreak = currentStreak
	}

	xp += awardedXP
	newLevel := levelFromXP(xp)

	if _, err := tx.Exec(ctx, `
		UPDATE user_gamification
		SET xp = $2,
		    level = $3,
		    current_streak = $4,
		    longest_streak = $5,
		    last_activity_day = $6::date,
		    updated_at = NOW()
		WHERE user_id = $1
	`, userID, xp, newLevel, currentStreak, longestStreak, activityDay.Format("2006-01-02")); err != nil {
		return gamificationSummary{}, 0, nil, err
	}

	unlocked, err := unlockAchievements(ctx, tx, userID, xp, currentStreak)
	if err != nil {
		return gamificationSummary{}, 0, nil, err
	}

	summary := gamificationSummary{
		XP:              xp,
		Level:           newLevel,
		CurrentStreak:   currentStreak,
		LongestStreak:   longestStreak,
		LastActivityDay: activityDay.Format("2006-01-02"),
	}

	_ = eventID
	return summary, awardedXP, unlocked, nil
}

func ensureGamificationProfileTx(ctx context.Context, tx pgx.Tx, userID int64) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_gamification (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	return err
}

func loadGamificationSummaryTx(ctx context.Context, tx pgx.Tx, userID int64) (gamificationSummary, error) {
	var (
		summary gamificationSummary
		lastDay sql.NullTime
	)
	err := tx.QueryRow(ctx, `
		SELECT xp, level, current_streak, longest_streak, last_activity_day::timestamptz
		FROM user_gamification
		WHERE user_id = $1
	`, userID).Scan(&summary.XP, &summary.Level, &summary.CurrentStreak, &summary.LongestStreak, &lastDay)
	if err != nil {
		return gamificationSummary{}, err
	}
	if lastDay.Valid {
		summary.LastActivityDay = lastDay.Time.UTC().Format("2006-01-02")
	}
	return summary, nil
}

func levelFromXP(xp int) int {
	if xp < 0 {
		xp = 0
	}
	return (xp / 500) + 1
}

func unlockAchievements(ctx context.Context, tx pgx.Tx, userID int64, xp int, currentStreak int) ([]achievementRow, error) {
	type candidate struct {
		key         string
		title       string
		description string
		met         bool
	}

	candidates := []candidate{
		{key: "first_completion", title: "Primeira Conclusao", description: "Concluiu o primeiro compromisso", met: true},
		{key: "xp_500", title: "Nivel Inicial", description: "Alcancou 500 XP", met: xp >= 500},
		{key: "streak_3", title: "Sequencia 3", description: "Manteve 3 dias seguidos", met: currentStreak >= 3},
		{key: "streak_7", title: "Sequencia 7", description: "Manteve 7 dias seguidos", met: currentStreak >= 7},
	}

	unlocked := make([]achievementRow, 0)
	for _, c := range candidates {
		if !c.met {
			continue
		}

		var row achievementRow
		err := tx.QueryRow(ctx, `
			INSERT INTO user_achievements (user_id, achievement_key, title, description)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id, achievement_key) DO NOTHING
			RETURNING achievement_key, title, description, unlocked_at
		`, userID, c.key, c.title, c.description).
			Scan(&row.Key, &row.Title, &row.Description, &row.UnlockedAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
				continue
			}
			return nil, err
		}
		unlocked = append(unlocked, row)
	}

	return unlocked, nil
}

func achievementRowsToResponse(rows []achievementRow) []fiber.Map {
	items := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		items = append(items, fiber.Map{
			"key":         r.Key,
			"title":       r.Title,
			"description": r.Description,
			"unlocked_at": r.UnlockedAt.UTC().Format(time.RFC3339),
		})
	}
	return items
}

func nullableTimeRFC3339(v sql.NullTime) interface{} {
	if !v.Valid {
		return nil
	}
	return v.Time.UTC().Format(time.RFC3339)
}

func nullableInt64(v sql.NullInt64) interface{} {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

func taskReminderOffsetsByPriority(priority string) []int {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "low":
		return []int{60}
	case "high":
		return []int{2880, 1440, 360, 60, 15}
	case "urgent":
		return []int{2880, 1440, 360, 60, 15, 5}
	default:
		return []int{1440, 360, 60}
	}
}

func taskRowToReminderPayload(row taskRow, eventType string) fiber.Map {
	startAt := time.Now().UTC()
	if row.DueAt.Valid {
		startAt = row.DueAt.Time.UTC()
	}

	return fiber.Map{
		"id":                       row.ID,
		"task_id":                  row.ID,
		"user_id":                  row.UserID,
		"title":                    row.Title,
		"description":              row.Description,
		"start_at":                 startAt.Format(time.RFC3339),
		"end_at":                   startAt.Add(30 * time.Minute).Format(time.RFC3339),
		"timezone":                 "UTC",
		"is_all_day":               false,
		"event_type":               eventType,
		"reminder_offsets_minutes": taskReminderOffsetsByPriority(row.Priority),
	}
}

func (a *appContext) enqueueReminderContract(ctx context.Context, tx pgx.Tx, userID, eventID int64, eventType string, row eventRow) error {
	payload := eventRowToResponse(row)
	payload["event_type"] = eventType
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO reminder_outbox (user_id, event_id, task_id, event_type, payload)
		VALUES ($1, $2, NULL, $3, $4::jsonb)
	`, userID, eventID, eventType, string(payloadJSON))
	if err != nil {
		return err
	}
	return nil
}

func (a *appContext) enqueueTaskReminderContract(ctx context.Context, tx pgx.Tx, row taskRow, eventType string) error {
	payload := taskRowToReminderPayload(row, eventType)
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO reminder_outbox (user_id, event_id, task_id, event_type, payload)
		VALUES ($1, NULL, $2, $3, $4::jsonb)
	`, row.UserID, row.ID, eventType, string(payloadJSON))
	if err != nil {
		return err
	}
	return nil
}

func (a *appContext) authMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid authorization header"})
	}

	tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	claims := &authClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return a.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid access token"})
	}

	c.Locals("user_id", claims.UserID)
	return c.Next()
}

func (a *appContext) createAccessToken(userID int64) (string, error) {
	jti, err := generateRandomToken(8)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := authClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(a.accessTTLMinutes) * time.Minute)),
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

func (a *appContext) storeRefreshToken(ctx context.Context, userID int64, token string) error {
	ctxDB, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := a.db.Exec(ctxDB,
		`INSERT INTO auth_sessions (user_id, refresh_token_hash, expires_at)
         VALUES ($1, $2, $3)`,
		userID,
		hashToken(token),
		time.Now().Add(time.Duration(a.refreshTTLHours)*time.Hour),
	)
	return err
}

func generateRandomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
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
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func parsePositiveInt(value string, fallback int) int {
	v := strings.TrimSpace(value)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func checkTCP(name, host, port string) dependencyStatus {
	target := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", target, 2*time.Second)
	if err != nil {
		return dependencyStatus{Name: name, Target: target, Status: "error", Error: err.Error()}
	}
	_ = conn.Close()
	return dependencyStatus{Name: name, Target: target, Status: "ok"}
}
