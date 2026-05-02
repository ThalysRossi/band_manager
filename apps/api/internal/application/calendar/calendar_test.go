package calendar

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

func TestListEventsAllowsViewerReadAccess(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := ListEventsInput{
		Account: viewerAccountContext(),
		From:    "2026-05-01",
		To:      "2026-05-31",
	}

	_, err := ListEvents(context.Background(), &repository, input)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
}

func TestCreateEventRejectsViewerWriteAccess(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := validCreateEventInput()
	input.Account = viewerAccountContext()

	_, err := CreateEvent(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected write access error")
	}
}

func TestCreateEventRejectsInvalidEventType(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := validCreateEventInput()
	input.Type = "invalid"

	_, err := CreateEvent(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected invalid event type error")
	}
}

func TestCreateEventRejectsBlankTitle(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := validCreateEventInput()
	input.Title = " "

	_, err := CreateEvent(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected title error")
	}
}

func TestCreateEventRejectsInvalidLocalDateTime(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := validCreateEventInput()
	input.StartsAtLocal = "2026/05/01 20:00"

	_, err := CreateEvent(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected local datetime error")
	}
}

func TestCreateEventRejectsEndBeforeStart(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := validCreateEventInput()
	input.EndsAtLocal = "2026-05-01T19:00:00"

	_, err := CreateEvent(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected end before start error")
	}
}

func TestCreateEventRejectsRecurrenceEndsOnAndCount(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := validCreateEventInput()
	input.Recurrence = RecurrenceInput{
		Frequency: string(RecurrenceFrequencyWeekly),
		Interval:  1,
		EndsOn:    "2026-06-01",
		Count:     4,
	}

	_, err := CreateEvent(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected recurrence end condition error")
	}
}

func TestListEventsRejectsFromAfterTo(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	_, err := ListEvents(context.Background(), &repository, ListEventsInput{
		Account: ownerAccountContext(),
		From:    "2026-05-31",
		To:      "2026-05-01",
	})
	if err == nil {
		t.Fatal("expected range error")
	}
}

func TestListEventsExpandsDailyWeeklyAndMonthlyRecurrence(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("America/Recife")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	repository := fakeRepository{
		events: []Event{
			recurringEvent("daily", RecurrenceFrequencyDaily, 1, 4, "2026-05-03T20:00:00", location),
			recurringEvent("weekly", RecurrenceFrequencyWeekly, 1, 2, "2026-05-04T20:00:00", location),
			recurringEvent("monthly", RecurrenceFrequencyMonthly, 1, 2, "2026-05-05T20:00:00", location),
		},
	}

	eventList, err := ListEvents(context.Background(), &repository, ListEventsInput{
		Account: ownerAccountContext(),
		From:    "2026-05-01",
		To:      "2026-06-10",
	})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}

	if len(eventList.Events) != 8 {
		t.Fatalf("expected 8 expanded occurrences, got %d", len(eventList.Events))
	}

	if eventList.Events[0].OccurrenceID != "daily:2026-05-03T20:00:00" {
		t.Fatalf("expected first daily occurrence, got %q", eventList.Events[0].OccurrenceID)
	}
	if eventList.Events[7].OccurrenceID != "monthly:2026-06-05T20:00:00" {
		t.Fatalf("expected monthly June occurrence, got %q", eventList.Events[7].OccurrenceID)
	}
}

func TestGetEventReturnsRepositoryErrorWithContext(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{err: ErrCalendarEventNotFound}
	_, err := GetEvent(context.Background(), &repository, GetEventInput{
		Account: ownerAccountContext(),
		EventID: "11111111-1111-1111-1111-111111111111",
	})
	if !errors.Is(err, ErrCalendarEventNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

type fakeRepository struct {
	events        []Event
	createCommand CreateEventCommand
	updateCommand UpdateEventCommand
	deleteCommand SoftDeleteEventCommand
	err           error
}

func (repository *fakeRepository) ListEvents(ctx context.Context, query ListEventsQuery) ([]Event, error) {
	if repository.err != nil {
		return nil, repository.err
	}

	return repository.events, nil
}

func (repository *fakeRepository) GetEvent(ctx context.Context, query GetEventQuery) (Event, error) {
	if repository.err != nil {
		return Event{}, repository.err
	}

	return Event{ID: query.EventID, BandID: query.Account.BandID}, nil
}

func (repository *fakeRepository) CreateEvent(ctx context.Context, command CreateEventCommand) (Event, error) {
	repository.createCommand = command
	if repository.err != nil {
		return Event{}, repository.err
	}

	return Event{
		ID:            "11111111-1111-1111-1111-111111111111",
		BandID:        command.Account.BandID,
		Type:          command.Type,
		Title:         command.Title,
		StartsAtLocal: command.StartsAtLocal,
		EndsAtLocal:   command.EndsAtLocal,
		Timezone:      command.Account.BandTimezone,
		Recurrence:    command.Recurrence,
	}, nil
}

func (repository *fakeRepository) UpdateEvent(ctx context.Context, command UpdateEventCommand) (Event, error) {
	repository.updateCommand = command
	if repository.err != nil {
		return Event{}, repository.err
	}

	return Event{
		ID:            command.EventID,
		BandID:        command.Account.BandID,
		Type:          command.Type,
		Title:         command.Title,
		StartsAtLocal: command.StartsAtLocal,
		EndsAtLocal:   command.EndsAtLocal,
		Timezone:      command.Account.BandTimezone,
		Recurrence:    command.Recurrence,
	}, nil
}

func (repository *fakeRepository) SoftDeleteEvent(ctx context.Context, command SoftDeleteEventCommand) error {
	repository.deleteCommand = command
	return repository.err
}

func ownerAccountContext() AccountContext {
	return AccountContext{
		UserID:       "user_1",
		BandID:       "band_1",
		BandTimezone: "America/Recife",
		Role:         permissions.RoleOwner,
	}
}

func viewerAccountContext() AccountContext {
	account := ownerAccountContext()
	account.Role = permissions.RoleViewer
	return account
}

func validCreateEventInput() CreateEventInput {
	return CreateEventInput{
		Account:       ownerAccountContext(),
		Type:          string(EventTypeShow),
		Title:         "Show em Recife",
		Description:   "Set de 45 minutos",
		LocationName:  "Casa de Shows",
		Address:       "Rua Principal, 123",
		StartsAtLocal: "2026-05-01T20:00:00",
		EndsAtLocal:   "2026-05-01T22:00:00",
		Recurrence: RecurrenceInput{
			Frequency: string(RecurrenceFrequencyNone),
		},
		IdempotencyKey: "idem_1",
		RequestID:      "request_1",
		CreatedAt:      time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}
}

func recurringEvent(id string, frequency RecurrenceFrequency, interval int, count int, startsAtLocalValue string, location *time.Location) Event {
	startsAtLocal, err := time.ParseInLocation(localDateTimeLayout, startsAtLocalValue, location)
	if err != nil {
		panic(err)
	}

	return Event{
		ID:            id,
		BandID:        "band_1",
		Type:          EventTypeRehearsal,
		Title:         id,
		StartsAtLocal: startsAtLocal,
		EndsAtLocal:   startsAtLocal.Add(2 * time.Hour),
		Timezone:      "America/Recife",
		Recurrence: Recurrence{
			Frequency: frequency,
			Interval:  interval,
			Count:     count,
		},
	}
}
