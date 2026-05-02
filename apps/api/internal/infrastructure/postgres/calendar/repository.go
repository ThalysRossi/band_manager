package calendar

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	applicationcalendar "github.com/thalys/band-manager/apps/api/internal/application/calendar"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return Repository{pool: pool}
}

func (repository Repository) ListEvents(ctx context.Context, query applicationcalendar.ListEventsQuery) ([]applicationcalendar.Event, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id, band_id, event_type, title, description, location_name, address,
			starts_at_local, ends_at_local, timezone, recurrence_frequency,
			recurrence_interval, recurrence_ends_on, recurrence_count, created_at, updated_at
		FROM calendar_events
		WHERE band_id = $1
			AND deleted_at IS NULL
			AND (
				(
					recurrence_frequency = 'none'
					AND ends_at_local > $2
					AND starts_at_local < $3
				)
				OR
				(
					recurrence_frequency <> 'none'
					AND starts_at_local < $3
					AND (recurrence_ends_on IS NULL OR recurrence_ends_on >= $4)
				)
			)
		ORDER BY starts_at_local ASC, id ASC
	`, query.Account.BandID, query.FromLocal, query.ToExclusiveLocal, query.Range.From)
	if err != nil {
		return nil, fmt.Errorf("query calendar events band_id=%q from=%q to=%q: %w", query.Account.BandID, query.Range.From, query.Range.To, err)
	}
	defer rows.Close()

	events := make([]applicationcalendar.Event, 0)
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan calendar event band_id=%q: %w", query.Account.BandID, err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate calendar events band_id=%q: %w", query.Account.BandID, err)
	}

	return events, nil
}

func (repository Repository) GetEvent(ctx context.Context, query applicationcalendar.GetEventQuery) (applicationcalendar.Event, error) {
	event, err := getEventByID(ctx, repository.pool, query.Account.BandID, query.EventID)
	if err != nil {
		return applicationcalendar.Event{}, err
	}

	return event, nil
}

func (repository Repository) CreateEvent(ctx context.Context, command applicationcalendar.CreateEventCommand) (applicationcalendar.Event, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return applicationcalendar.Event{}, fmt.Errorf("begin calendar event create transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	eventID := uuid.NewString()
	_, err = tx.Exec(ctx, `
		INSERT INTO calendar_events (
			id, band_id, created_by_user_id, event_type, title, description,
			location_name, address, starts_at_local, ends_at_local, timezone,
			recurrence_frequency, recurrence_interval, recurrence_ends_on,
			recurrence_count, idempotency_key, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $17)
	`, eventID, command.Account.BandID, command.Account.UserID, command.Type, command.Title, command.Description, command.LocationName, command.Address, command.StartsAtLocal, command.EndsAtLocal, command.Account.BandTimezone, command.Recurrence.Frequency, command.Recurrence.Interval, nullableDate(command.Recurrence.EndsOn), nullableInt(command.Recurrence.Count), command.IdempotencyKey, command.CreatedAt)
	if err != nil {
		return applicationcalendar.Event{}, fmt.Errorf("insert calendar event band_id=%q title=%q: %w", command.Account.BandID, command.Title, err)
	}

	if err := insertAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "calendar.event_created", "calendar_event", eventID, command.RequestID, command.IdempotencyKey, command.CreatedAt); err != nil {
		return applicationcalendar.Event{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return applicationcalendar.Event{}, fmt.Errorf("commit calendar event create transaction band_id=%q event_id=%q: %w", command.Account.BandID, eventID, err)
	}

	return applicationcalendar.Event{
		ID:            eventID,
		BandID:        command.Account.BandID,
		Type:          command.Type,
		Title:         command.Title,
		Description:   command.Description,
		LocationName:  command.LocationName,
		Address:       command.Address,
		StartsAtLocal: command.StartsAtLocal,
		EndsAtLocal:   command.EndsAtLocal,
		Timezone:      command.Account.BandTimezone,
		Recurrence:    command.Recurrence,
		CreatedAt:     command.CreatedAt,
		UpdatedAt:     command.CreatedAt,
	}, nil
}

func (repository Repository) UpdateEvent(ctx context.Context, command applicationcalendar.UpdateEventCommand) (applicationcalendar.Event, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return applicationcalendar.Event{}, fmt.Errorf("begin calendar event update transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	commandTag, err := tx.Exec(ctx, `
		UPDATE calendar_events
		SET event_type = $1,
			title = $2,
			description = $3,
			location_name = $4,
			address = $5,
			starts_at_local = $6,
			ends_at_local = $7,
			timezone = $8,
			recurrence_frequency = $9,
			recurrence_interval = $10,
			recurrence_ends_on = $11,
			recurrence_count = $12,
			idempotency_key = $13,
			updated_at = $14
		WHERE id = $15 AND band_id = $16 AND deleted_at IS NULL
	`, command.Type, command.Title, command.Description, command.LocationName, command.Address, command.StartsAtLocal, command.EndsAtLocal, command.Account.BandTimezone, command.Recurrence.Frequency, command.Recurrence.Interval, nullableDate(command.Recurrence.EndsOn), nullableInt(command.Recurrence.Count), command.IdempotencyKey, command.UpdatedAt, command.EventID, command.Account.BandID)
	if err != nil {
		return applicationcalendar.Event{}, fmt.Errorf("update calendar event band_id=%q event_id=%q: %w", command.Account.BandID, command.EventID, err)
	}
	if commandTag.RowsAffected() == 0 {
		return applicationcalendar.Event{}, fmt.Errorf("%w: band_id=%q event_id=%q", applicationcalendar.ErrCalendarEventNotFound, command.Account.BandID, command.EventID)
	}

	if err := insertAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "calendar.event_updated", "calendar_event", command.EventID, command.RequestID, command.IdempotencyKey, command.UpdatedAt); err != nil {
		return applicationcalendar.Event{}, err
	}

	event, err := getEventByID(ctx, tx, command.Account.BandID, command.EventID)
	if err != nil {
		return applicationcalendar.Event{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return applicationcalendar.Event{}, fmt.Errorf("commit calendar event update transaction band_id=%q event_id=%q: %w", command.Account.BandID, command.EventID, err)
	}

	return event, nil
}

func (repository Repository) SoftDeleteEvent(ctx context.Context, command applicationcalendar.SoftDeleteEventCommand) error {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin calendar event delete transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	commandTag, err := tx.Exec(ctx, `
		UPDATE calendar_events
		SET deleted_at = $1, deleted_by = $2, updated_at = $1
		WHERE id = $3 AND band_id = $4 AND deleted_at IS NULL
	`, command.DeletedAt, command.Account.UserID, command.EventID, command.Account.BandID)
	if err != nil {
		return fmt.Errorf("soft delete calendar event band_id=%q event_id=%q: %w", command.Account.BandID, command.EventID, err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("%w: band_id=%q event_id=%q", applicationcalendar.ErrCalendarEventNotFound, command.Account.BandID, command.EventID)
	}

	if err := insertAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "calendar.event_deleted", "calendar_event", command.EventID, command.RequestID, command.IdempotencyKey, command.DeletedAt); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit calendar event delete transaction band_id=%q event_id=%q: %w", command.Account.BandID, command.EventID, err)
	}

	return nil
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func getEventByID(ctx context.Context, queryer queryer, bandID string, eventID string) (applicationcalendar.Event, error) {
	event, err := scanEvent(queryer.QueryRow(ctx, `
		SELECT id, band_id, event_type, title, description, location_name, address,
			starts_at_local, ends_at_local, timezone, recurrence_frequency,
			recurrence_interval, recurrence_ends_on, recurrence_count, created_at, updated_at
		FROM calendar_events
		WHERE id = $1 AND band_id = $2 AND deleted_at IS NULL
	`, eventID, bandID))
	if errors.Is(err, pgx.ErrNoRows) {
		return applicationcalendar.Event{}, fmt.Errorf("%w: band_id=%q event_id=%q", applicationcalendar.ErrCalendarEventNotFound, bandID, eventID)
	}
	if err != nil {
		return applicationcalendar.Event{}, fmt.Errorf("scan calendar event band_id=%q event_id=%q: %w", bandID, eventID, err)
	}

	return event, nil
}

func scanEvent(row pgx.Row) (applicationcalendar.Event, error) {
	var event applicationcalendar.Event
	var eventType string
	var recurrenceFrequency string
	var recurrenceEndsOn sql.NullTime
	var recurrenceCount sql.NullInt64
	err := row.Scan(
		&event.ID,
		&event.BandID,
		&eventType,
		&event.Title,
		&event.Description,
		&event.LocationName,
		&event.Address,
		&event.StartsAtLocal,
		&event.EndsAtLocal,
		&event.Timezone,
		&recurrenceFrequency,
		&event.Recurrence.Interval,
		&recurrenceEndsOn,
		&recurrenceCount,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err != nil {
		return applicationcalendar.Event{}, err
	}

	event.Type = applicationcalendar.EventType(eventType)
	event.Recurrence.Frequency = applicationcalendar.RecurrenceFrequency(recurrenceFrequency)
	if recurrenceEndsOn.Valid {
		event.Recurrence.EndsOn = recurrenceEndsOn.Time.Format("2006-01-02")
	}
	if recurrenceCount.Valid {
		event.Recurrence.Count = int(recurrenceCount.Int64)
	}

	startsAtLocal, err := localTimeInEventLocation(event.StartsAtLocal, event.Timezone)
	if err != nil {
		return applicationcalendar.Event{}, err
	}
	event.StartsAtLocal = startsAtLocal

	endsAtLocal, err := localTimeInEventLocation(event.EndsAtLocal, event.Timezone)
	if err != nil {
		return applicationcalendar.Event{}, err
	}
	event.EndsAtLocal = endsAtLocal

	return event, nil
}

func localTimeInEventLocation(value time.Time, timezone string) (time.Time, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("load calendar event timezone %q: %w", timezone, err)
	}

	parsed, err := time.ParseInLocation("2006-01-02T15:04:05", value.Format("2006-01-02T15:04:05"), location)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse calendar event local timestamp timezone=%q value=%q: %w", timezone, value.Format("2006-01-02T15:04:05"), err)
	}

	return parsed, nil
}

func nullableDate(value string) interface{} {
	if value == "" {
		return nil
	}

	return value
}

func nullableInt(value int) interface{} {
	if value == 0 {
		return nil
	}

	return value
}

func insertAuditLog(ctx context.Context, tx pgx.Tx, userID string, bandID string, action string, entityType string, entityID string, requestID string, idempotencyKey string, createdAt time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, band_id, action, entity_type, entity_id, request_id, idempotency_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`, uuid.NewString(), userID, bandID, action, entityType, entityID, requestID, idempotencyKey, createdAt)
	if err != nil {
		return fmt.Errorf("insert calendar audit log user_id=%q band_id=%q action=%q entity_type=%q entity_id=%q: %w", userID, bandID, action, entityType, entityID, err)
	}

	return nil
}
