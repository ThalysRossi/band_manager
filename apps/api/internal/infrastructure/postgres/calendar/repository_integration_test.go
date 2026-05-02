package calendar

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	applicationcalendar "github.com/thalys/band-manager/apps/api/internal/application/calendar"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

func TestRepositoryCreateReadUpdateAndSoftDeleteEvent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	createdEvent, err := repository.CreateEvent(ctx, validCreateEventCommand(account))
	if err != nil {
		t.Fatalf("create calendar event: %v", err)
	}

	if createdEvent.Title != "Show em Recife" {
		t.Fatalf("expected created title, got %q", createdEvent.Title)
	}

	readEvent, err := repository.GetEvent(ctx, applicationcalendar.GetEventQuery{
		Account: account,
		EventID: createdEvent.ID,
	})
	if err != nil {
		t.Fatalf("get calendar event: %v", err)
	}
	if readEvent.StartsAtLocal.Format("2006-01-02T15:04:05") != "2026-05-10T20:00:00" {
		t.Fatalf("expected local start, got %s", readEvent.StartsAtLocal.Format("2006-01-02T15:04:05"))
	}

	updateCommand := validUpdateEventCommand(account, createdEvent.ID)
	updatedEvent, err := repository.UpdateEvent(ctx, updateCommand)
	if err != nil {
		t.Fatalf("update calendar event: %v", err)
	}
	if updatedEvent.Title != "Ensaio geral" {
		t.Fatalf("expected updated title, got %q", updatedEvent.Title)
	}

	err = repository.SoftDeleteEvent(ctx, applicationcalendar.SoftDeleteEventCommand{
		Account:        account,
		EventID:        createdEvent.ID,
		IdempotencyKey: "idem_calendar_delete",
		RequestID:      "request_calendar_delete",
		DeletedAt:      testTimestamp().Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatalf("soft delete calendar event: %v", err)
	}

	_, err = repository.GetEvent(ctx, applicationcalendar.GetEventQuery{
		Account: account,
		EventID: createdEvent.ID,
	})
	if !errors.Is(err, applicationcalendar.ErrCalendarEventNotFound) {
		t.Fatalf("expected not found after soft delete, got %v", err)
	}

	assertTableCount(t, pool, "audit_logs", "band_id = $1 AND entity_id = $2", []interface{}{account.BandID, createdEvent.ID}, 3)
}

func TestRepositoryListEventsExpandsRecurringEventsAndExcludesDeletedEvents(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	event, err := repository.CreateEvent(ctx, applicationcalendar.CreateEventCommand{
		Account:       account,
		Type:          applicationcalendar.EventTypeRehearsal,
		Title:         "Ensaio semanal",
		StartsAtLocal: localTime(t, "2026-05-05T19:00:00"),
		EndsAtLocal:   localTime(t, "2026-05-05T21:00:00"),
		Recurrence: applicationcalendar.Recurrence{
			Frequency: applicationcalendar.RecurrenceFrequencyWeekly,
			Interval:  1,
			Count:     3,
		},
		IdempotencyKey: "idem_calendar_recurring",
		RequestID:      "request_calendar_recurring",
		CreatedAt:      testTimestamp(),
	})
	if err != nil {
		t.Fatalf("create recurring event: %v", err)
	}

	eventList, err := applicationcalendar.ListEvents(ctx, repository, applicationcalendar.ListEventsInput{
		Account: account,
		From:    "2026-05-01",
		To:      "2026-05-31",
	})
	if err != nil {
		t.Fatalf("list calendar events: %v", err)
	}
	if len(eventList.Events) != 3 {
		t.Fatalf("expected three recurring occurrences, got %d", len(eventList.Events))
	}
	if eventList.Events[2].OccurrenceID != event.ID+":2026-05-19T19:00:00" {
		t.Fatalf("expected third weekly occurrence, got %q", eventList.Events[2].OccurrenceID)
	}

	err = repository.SoftDeleteEvent(ctx, applicationcalendar.SoftDeleteEventCommand{
		Account:        account,
		EventID:        event.ID,
		IdempotencyKey: "idem_calendar_recurring_delete",
		RequestID:      "request_calendar_recurring_delete",
		DeletedAt:      testTimestamp().Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatalf("soft delete recurring event: %v", err)
	}

	eventList, err = applicationcalendar.ListEvents(ctx, repository, applicationcalendar.ListEventsInput{
		Account: account,
		From:    "2026-05-01",
		To:      "2026-05-31",
	})
	if err != nil {
		t.Fatalf("list calendar events after delete: %v", err)
	}
	if len(eventList.Events) != 0 {
		t.Fatalf("expected no occurrences after delete, got %d", len(eventList.Events))
	}
}

func newIntegrationDatabase(t *testing.T) (*pgxpool.Pool, applicationcalendar.AccountContext) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		t.Skip("DATABASE_URL is not set; skipping Postgres integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	setupPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Skipf("Postgres is unavailable: %v", err)
	}
	if err := setupPool.Ping(ctx); err != nil {
		setupPool.Close()
		t.Skipf("Postgres is unavailable: %v", err)
	}

	schemaName := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "_")
	quotedSchemaName := pgx.Identifier{schemaName}.Sanitize()
	_, err = setupPool.Exec(ctx, "CREATE SCHEMA "+quotedSchemaName)
	if err != nil {
		setupPool.Close()
		t.Fatalf("create test schema %q: %v", schemaName, err)
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		setupPool.Close()
		t.Fatalf("parse database url: %v", err)
	}
	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = map[string]string{}
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schemaName
	poolConfig.MaxConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		setupPool.Close()
		t.Fatalf("create schema-scoped pool: %v", err)
	}

	applyMigrations(ctx, t, pool)
	account := seedAccount(ctx, t, pool)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()

		pool.Close()
		_, dropErr := setupPool.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+quotedSchemaName+" CASCADE")
		if dropErr != nil {
			t.Logf("drop test schema %q: %v", schemaName, dropErr)
		}
		setupPool.Close()
	})

	return pool, account
}

func applyMigrations(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, migrationPath := range migrationFilePaths(t) {
		body, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("read migration %s: %v", migrationPath, err)
		}

		statements, err := upMigrationStatements(string(body))
		if err != nil {
			t.Fatalf("parse migration %s: %v", migrationPath, err)
		}

		for _, statement := range statements {
			_, err := pool.Exec(ctx, statement)
			if err != nil {
				t.Fatalf("apply migration %s statement %q: %v", migrationPath, statement, err)
			}
		}
	}
}

func migrationFilePaths(t *testing.T) []string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}

	apiRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../.."))
	migrationPaths, err := filepath.Glob(filepath.Join(apiRoot, "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("glob migration files: %v", err)
	}
	sort.Strings(migrationPaths)

	return migrationPaths
}

func upMigrationStatements(body string) ([]string, error) {
	upMarkerIndex := strings.Index(body, "-- +goose Up")
	if upMarkerIndex == -1 {
		return nil, errors.New("goose up marker is required")
	}

	downMarkerIndex := strings.Index(body, "-- +goose Down")
	if downMarkerIndex == -1 {
		return nil, errors.New("goose down marker is required")
	}

	if downMarkerIndex <= upMarkerIndex {
		return nil, errors.New("goose down marker must follow up marker")
	}

	upBody := body[upMarkerIndex+len("-- +goose Up") : downMarkerIndex]
	parts := strings.Split(upBody, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement != "" {
			statements = append(statements, statement)
		}
	}

	return statements, nil
}

func seedAccount(ctx context.Context, t *testing.T, pool *pgxpool.Pool) applicationcalendar.AccountContext {
	t.Helper()

	now := testTimestamp()
	userID := uuid.NewString()
	bandID := uuid.NewString()

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, auth_provider, auth_provider_user_id, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
	`, userID, "supabase", "auth_"+userID, userID+"@example.com", now)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO bands (id, name, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
	`, bandID, "Os Testes", "America/Recife", now)
	if err != nil {
		t.Fatalf("seed band: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO band_memberships (id, band_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
	`, uuid.NewString(), bandID, userID, permissions.RoleOwner, now)
	if err != nil {
		t.Fatalf("seed membership: %v", err)
	}

	return applicationcalendar.AccountContext{
		UserID:       userID,
		BandID:       bandID,
		BandTimezone: "America/Recife",
		Role:         permissions.RoleOwner,
	}
}

func validCreateEventCommand(account applicationcalendar.AccountContext) applicationcalendar.CreateEventCommand {
	return applicationcalendar.CreateEventCommand{
		Account:       account,
		Type:          applicationcalendar.EventTypeShow,
		Title:         "Show em Recife",
		Description:   "Set de 45 minutos",
		LocationName:  "Casa de Shows",
		Address:       "Rua Principal, 123",
		StartsAtLocal: localTime(nil, "2026-05-10T20:00:00"),
		EndsAtLocal:   localTime(nil, "2026-05-10T22:00:00"),
		Recurrence: applicationcalendar.Recurrence{
			Frequency: applicationcalendar.RecurrenceFrequencyNone,
		},
		IdempotencyKey: "idem_calendar_create",
		RequestID:      "request_calendar_create",
		CreatedAt:      testTimestamp(),
	}
}

func validUpdateEventCommand(account applicationcalendar.AccountContext, eventID string) applicationcalendar.UpdateEventCommand {
	return applicationcalendar.UpdateEventCommand{
		Account:       account,
		EventID:       eventID,
		Type:          applicationcalendar.EventTypeRehearsal,
		Title:         "Ensaio geral",
		Description:   "Revisar setlist",
		LocationName:  "Estudio",
		Address:       "Rua do Estudio, 456",
		StartsAtLocal: localTime(nil, "2026-05-11T19:00:00"),
		EndsAtLocal:   localTime(nil, "2026-05-11T21:00:00"),
		Recurrence: applicationcalendar.Recurrence{
			Frequency: applicationcalendar.RecurrenceFrequencyNone,
		},
		IdempotencyKey: "idem_calendar_update",
		RequestID:      "request_calendar_update",
		UpdatedAt:      testTimestamp().Add(time.Hour),
	}
}

func localTime(t *testing.T, value string) time.Time {
	location, err := time.LoadLocation("America/Recife")
	if err != nil {
		if t != nil {
			t.Fatalf("load location: %v", err)
		}
		panic(err)
	}

	parsed, err := time.ParseInLocation("2006-01-02T15:04:05", value, location)
	if err != nil {
		if t != nil {
			t.Fatalf("parse local time: %v", err)
		}
		panic(err)
	}

	return parsed
}

func assertTableCount(t *testing.T, pool *pgxpool.Pool, tableName string, whereClause string, args []interface{}, expectedCount int) {
	t.Helper()

	var count int
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", pgx.Identifier{tableName}.Sanitize(), whereClause)
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("count %s rows: %v", tableName, err)
	}

	if count != expectedCount {
		t.Fatalf("expected %d rows in %s, got %d", expectedCount, tableName, count)
	}
}

func testTimestamp() time.Time {
	return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
}
