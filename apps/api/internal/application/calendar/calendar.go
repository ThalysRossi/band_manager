package calendar

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

const dateLayout = "2006-01-02"
const localDateTimeLayout = "2006-01-02T15:04:05"
const maxExpandedOccurrences = 10000

var ErrCalendarEventNotFound = errors.New("calendar event not found")

type Repository interface {
	ListEvents(ctx context.Context, query ListEventsQuery) ([]Event, error)
	GetEvent(ctx context.Context, query GetEventQuery) (Event, error)
	CreateEvent(ctx context.Context, command CreateEventCommand) (Event, error)
	UpdateEvent(ctx context.Context, command UpdateEventCommand) (Event, error)
	SoftDeleteEvent(ctx context.Context, command SoftDeleteEventCommand) error
}

type AccountContext struct {
	UserID       string
	BandID       string
	BandTimezone string
	Role         permissions.Role
}

type EventType string

const (
	EventTypeShow      EventType = "show"
	EventTypeRehearsal EventType = "rehearsal"
	EventTypeRelease   EventType = "release"
	EventTypeMeeting   EventType = "meeting"
	EventTypeTask      EventType = "task"
	EventTypeOther     EventType = "other"
)

type RecurrenceFrequency string

const (
	RecurrenceFrequencyNone    RecurrenceFrequency = "none"
	RecurrenceFrequencyDaily   RecurrenceFrequency = "daily"
	RecurrenceFrequencyWeekly  RecurrenceFrequency = "weekly"
	RecurrenceFrequencyMonthly RecurrenceFrequency = "monthly"
)

type RecurrenceInput struct {
	Frequency string
	Interval  int
	EndsOn    string
	Count     int
}

type ListEventsInput struct {
	Account AccountContext
	From    string
	To      string
}

type GetEventInput struct {
	Account AccountContext
	EventID string
}

type CreateEventInput struct {
	Account        AccountContext
	Type           string
	Title          string
	Description    string
	LocationName   string
	Address        string
	StartsAtLocal  string
	EndsAtLocal    string
	Recurrence     RecurrenceInput
	IdempotencyKey string
	RequestID      string
	CreatedAt      time.Time
}

type UpdateEventInput struct {
	Account        AccountContext
	EventID        string
	Type           string
	Title          string
	Description    string
	LocationName   string
	Address        string
	StartsAtLocal  string
	EndsAtLocal    string
	Recurrence     RecurrenceInput
	IdempotencyKey string
	RequestID      string
	UpdatedAt      time.Time
}

type DeleteEventInput struct {
	Account        AccountContext
	EventID        string
	IdempotencyKey string
	RequestID      string
	DeletedAt      time.Time
}

type ListEventsQuery struct {
	Account          AccountContext
	Range            EventRange
	FromLocal        time.Time
	ToExclusiveLocal time.Time
}

type GetEventQuery struct {
	Account AccountContext
	EventID string
}

type CreateEventCommand struct {
	Account        AccountContext
	Type           EventType
	Title          string
	Description    string
	LocationName   string
	Address        string
	StartsAtLocal  time.Time
	EndsAtLocal    time.Time
	Recurrence     Recurrence
	IdempotencyKey string
	RequestID      string
	CreatedAt      time.Time
}

type UpdateEventCommand struct {
	Account        AccountContext
	EventID        string
	Type           EventType
	Title          string
	Description    string
	LocationName   string
	Address        string
	StartsAtLocal  time.Time
	EndsAtLocal    time.Time
	Recurrence     Recurrence
	IdempotencyKey string
	RequestID      string
	UpdatedAt      time.Time
}

type SoftDeleteEventCommand struct {
	Account        AccountContext
	EventID        string
	IdempotencyKey string
	RequestID      string
	DeletedAt      time.Time
}

type EventRange struct {
	From     string
	To       string
	Timezone string
}

type EventList struct {
	Range  EventRange
	Events []Occurrence
}

type Event struct {
	ID            string
	BandID        string
	Type          EventType
	Title         string
	Description   string
	LocationName  string
	Address       string
	StartsAtLocal time.Time
	EndsAtLocal   time.Time
	Timezone      string
	Recurrence    Recurrence
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Recurrence struct {
	Frequency RecurrenceFrequency
	Interval  int
	EndsOn    string
	Count     int
}

type Occurrence struct {
	EventID       string
	OccurrenceID  string
	Type          EventType
	Title         string
	Description   string
	LocationName  string
	Address       string
	StartsAtLocal time.Time
	EndsAtLocal   time.Time
	Timezone      string
	Recurring     bool
	Recurrence    Recurrence
}

func ListEvents(ctx context.Context, repository Repository, input ListEventsInput) (EventList, error) {
	query, err := validateListEventsInput(input)
	if err != nil {
		return EventList{}, err
	}

	events, err := repository.ListEvents(ctx, query)
	if err != nil {
		return EventList{}, fmt.Errorf("list calendar events band_id=%q from=%q to=%q timezone=%q: %w", query.Account.BandID, query.Range.From, query.Range.To, query.Range.Timezone, err)
	}

	occurrences, err := expandEvents(events, query)
	if err != nil {
		return EventList{}, err
	}

	return EventList{
		Range:  query.Range,
		Events: occurrences,
	}, nil
}

func GetEvent(ctx context.Context, repository Repository, input GetEventInput) (Event, error) {
	query, err := validateGetEventInput(input)
	if err != nil {
		return Event{}, err
	}

	event, err := repository.GetEvent(ctx, query)
	if err != nil {
		return Event{}, fmt.Errorf("get calendar event band_id=%q event_id=%q: %w", query.Account.BandID, query.EventID, err)
	}

	return event, nil
}

func CreateEvent(ctx context.Context, repository Repository, input CreateEventInput) (Event, error) {
	command, err := validateCreateEventInput(input)
	if err != nil {
		return Event{}, err
	}

	event, err := repository.CreateEvent(ctx, command)
	if err != nil {
		return Event{}, fmt.Errorf("create calendar event band_id=%q title=%q: %w", command.Account.BandID, command.Title, err)
	}

	return event, nil
}

func UpdateEvent(ctx context.Context, repository Repository, input UpdateEventInput) (Event, error) {
	command, err := validateUpdateEventInput(input)
	if err != nil {
		return Event{}, err
	}

	event, err := repository.UpdateEvent(ctx, command)
	if err != nil {
		return Event{}, fmt.Errorf("update calendar event band_id=%q event_id=%q: %w", command.Account.BandID, command.EventID, err)
	}

	return event, nil
}

func SoftDeleteEvent(ctx context.Context, repository Repository, input DeleteEventInput) error {
	command, err := validateSoftDeleteEventInput(input)
	if err != nil {
		return err
	}

	if err := repository.SoftDeleteEvent(ctx, command); err != nil {
		return fmt.Errorf("soft delete calendar event band_id=%q event_id=%q: %w", command.Account.BandID, command.EventID, err)
	}

	return nil
}

func validateListEventsInput(input ListEventsInput) (ListEventsQuery, error) {
	if err := validateReadAccount(input.Account); err != nil {
		return ListEventsQuery{}, err
	}

	location, err := time.LoadLocation(input.Account.BandTimezone)
	if err != nil {
		return ListEventsQuery{}, fmt.Errorf("band timezone %q is invalid: %w", input.Account.BandTimezone, err)
	}

	from, err := parseLocalDate("from", input.From, location)
	if err != nil {
		return ListEventsQuery{}, err
	}

	to, err := parseLocalDate("to", input.To, location)
	if err != nil {
		return ListEventsQuery{}, err
	}

	if from.After(to) {
		return ListEventsQuery{}, fmt.Errorf("from date %q must be on or before to date %q", from.Format(dateLayout), to.Format(dateLayout))
	}

	return ListEventsQuery{
		Account: input.Account,
		Range: EventRange{
			From:     from.Format(dateLayout),
			To:       to.Format(dateLayout),
			Timezone: input.Account.BandTimezone,
		},
		FromLocal:        from,
		ToExclusiveLocal: to.AddDate(0, 0, 1),
	}, nil
}

func validateGetEventInput(input GetEventInput) (GetEventQuery, error) {
	if err := validateReadAccount(input.Account); err != nil {
		return GetEventQuery{}, err
	}

	eventID, err := validateID("event id", input.EventID)
	if err != nil {
		return GetEventQuery{}, err
	}

	return GetEventQuery{Account: input.Account, EventID: eventID}, nil
}

func validateCreateEventInput(input CreateEventInput) (CreateEventCommand, error) {
	if err := validateWriteAccount(input.Account); err != nil {
		return CreateEventCommand{}, err
	}

	fields, err := validateEventFields(input.Account, input.Type, input.Title, input.Description, input.LocationName, input.Address, input.StartsAtLocal, input.EndsAtLocal, input.Recurrence)
	if err != nil {
		return CreateEventCommand{}, err
	}

	idempotencyKey, requestID, err := validateMutationMetadata(input.IdempotencyKey, input.RequestID, input.CreatedAt)
	if err != nil {
		return CreateEventCommand{}, err
	}

	return CreateEventCommand{
		Account:        input.Account,
		Type:           fields.Type,
		Title:          fields.Title,
		Description:    fields.Description,
		LocationName:   fields.LocationName,
		Address:        fields.Address,
		StartsAtLocal:  fields.StartsAtLocal,
		EndsAtLocal:    fields.EndsAtLocal,
		Recurrence:     fields.Recurrence,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		CreatedAt:      input.CreatedAt.UTC(),
	}, nil
}

func validateUpdateEventInput(input UpdateEventInput) (UpdateEventCommand, error) {
	if err := validateWriteAccount(input.Account); err != nil {
		return UpdateEventCommand{}, err
	}

	eventID, err := validateID("event id", input.EventID)
	if err != nil {
		return UpdateEventCommand{}, err
	}

	fields, err := validateEventFields(input.Account, input.Type, input.Title, input.Description, input.LocationName, input.Address, input.StartsAtLocal, input.EndsAtLocal, input.Recurrence)
	if err != nil {
		return UpdateEventCommand{}, err
	}

	idempotencyKey, requestID, err := validateMutationMetadata(input.IdempotencyKey, input.RequestID, input.UpdatedAt)
	if err != nil {
		return UpdateEventCommand{}, err
	}

	return UpdateEventCommand{
		Account:        input.Account,
		EventID:        eventID,
		Type:           fields.Type,
		Title:          fields.Title,
		Description:    fields.Description,
		LocationName:   fields.LocationName,
		Address:        fields.Address,
		StartsAtLocal:  fields.StartsAtLocal,
		EndsAtLocal:    fields.EndsAtLocal,
		Recurrence:     fields.Recurrence,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		UpdatedAt:      input.UpdatedAt.UTC(),
	}, nil
}

func validateSoftDeleteEventInput(input DeleteEventInput) (SoftDeleteEventCommand, error) {
	if err := validateWriteAccount(input.Account); err != nil {
		return SoftDeleteEventCommand{}, err
	}

	eventID, err := validateID("event id", input.EventID)
	if err != nil {
		return SoftDeleteEventCommand{}, err
	}

	idempotencyKey, requestID, err := validateMutationMetadata(input.IdempotencyKey, input.RequestID, input.DeletedAt)
	if err != nil {
		return SoftDeleteEventCommand{}, err
	}

	return SoftDeleteEventCommand{
		Account:        input.Account,
		EventID:        eventID,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		DeletedAt:      input.DeletedAt.UTC(),
	}, nil
}

type validatedEventFields struct {
	Type          EventType
	Title         string
	Description   string
	LocationName  string
	Address       string
	StartsAtLocal time.Time
	EndsAtLocal   time.Time
	Recurrence    Recurrence
}

func validateEventFields(account AccountContext, eventTypeInput string, titleInput string, descriptionInput string, locationNameInput string, addressInput string, startsAtLocalInput string, endsAtLocalInput string, recurrenceInput RecurrenceInput) (validatedEventFields, error) {
	location, err := time.LoadLocation(account.BandTimezone)
	if err != nil {
		return validatedEventFields{}, fmt.Errorf("band timezone %q is invalid: %w", account.BandTimezone, err)
	}

	eventType, err := parseEventType(eventTypeInput)
	if err != nil {
		return validatedEventFields{}, err
	}

	title := strings.TrimSpace(titleInput)
	if title == "" {
		return validatedEventFields{}, fmt.Errorf("title is required")
	}

	startsAtLocal, err := parseLocalDateTime("startsAtLocal", startsAtLocalInput, location)
	if err != nil {
		return validatedEventFields{}, err
	}

	endsAtLocal, err := parseLocalDateTime("endsAtLocal", endsAtLocalInput, location)
	if err != nil {
		return validatedEventFields{}, err
	}

	if !endsAtLocal.After(startsAtLocal) {
		return validatedEventFields{}, fmt.Errorf("endsAtLocal must be after startsAtLocal")
	}

	recurrence, err := validateRecurrence(recurrenceInput, startsAtLocal)
	if err != nil {
		return validatedEventFields{}, err
	}

	return validatedEventFields{
		Type:          eventType,
		Title:         title,
		Description:   strings.TrimSpace(descriptionInput),
		LocationName:  strings.TrimSpace(locationNameInput),
		Address:       strings.TrimSpace(addressInput),
		StartsAtLocal: startsAtLocal,
		EndsAtLocal:   endsAtLocal,
		Recurrence:    recurrence,
	}, nil
}

func validateRecurrence(input RecurrenceInput, startsAtLocal time.Time) (Recurrence, error) {
	frequency, err := parseRecurrenceFrequency(input.Frequency)
	if err != nil {
		return Recurrence{}, err
	}

	if frequency == RecurrenceFrequencyNone {
		if input.Interval != 0 || strings.TrimSpace(input.EndsOn) != "" || input.Count != 0 {
			return Recurrence{}, fmt.Errorf("non-recurring events cannot set recurrence interval, endsOn, or count")
		}

		return Recurrence{Frequency: RecurrenceFrequencyNone}, nil
	}

	if input.Interval <= 0 {
		return Recurrence{}, fmt.Errorf("recurrence interval must be greater than zero")
	}

	endsOn := strings.TrimSpace(input.EndsOn)
	if endsOn != "" && input.Count != 0 {
		return Recurrence{}, fmt.Errorf("recurrence endsOn and count are mutually exclusive")
	}

	if input.Count < 0 {
		return Recurrence{}, fmt.Errorf("recurrence count must be greater than zero")
	}

	if endsOn != "" {
		parsedEndsOn, err := parseLocalDate("recurrence.endsOn", endsOn, startsAtLocal.Location())
		if err != nil {
			return Recurrence{}, err
		}
		startDate := dateOnly(startsAtLocal)
		if parsedEndsOn.Before(startDate) {
			return Recurrence{}, fmt.Errorf("recurrence endsOn %q must be on or after startsAtLocal date %q", endsOn, startDate.Format(dateLayout))
		}
		endsOn = parsedEndsOn.Format(dateLayout)
	}

	return Recurrence{
		Frequency: frequency,
		Interval:  input.Interval,
		EndsOn:    endsOn,
		Count:     input.Count,
	}, nil
}

func expandEvents(events []Event, query ListEventsQuery) ([]Occurrence, error) {
	occurrences := make([]Occurrence, 0)
	for _, event := range events {
		expandedEventOccurrences, err := expandEvent(event, query.FromLocal, query.ToExclusiveLocal)
		if err != nil {
			return nil, err
		}
		occurrences = append(occurrences, expandedEventOccurrences...)
	}

	sortOccurrences(occurrences)

	return occurrences, nil
}

func expandEvent(event Event, fromLocal time.Time, toExclusiveLocal time.Time) ([]Occurrence, error) {
	if event.Recurrence.Frequency == RecurrenceFrequencyNone {
		if !overlaps(event.StartsAtLocal, event.EndsAtLocal, fromLocal, toExclusiveLocal) {
			return []Occurrence{}, nil
		}

		return []Occurrence{occurrenceForEvent(event, event.StartsAtLocal, event.EndsAtLocal, false)}, nil
	}

	duration := event.EndsAtLocal.Sub(event.StartsAtLocal)
	occurrences := make([]Occurrence, 0)
	occurrenceStart := event.StartsAtLocal
	occurrenceNumber := 1
	for {
		if occurrenceNumber > maxExpandedOccurrences {
			return nil, fmt.Errorf("calendar event recurrence expansion exceeded %d occurrences event_id=%q", maxExpandedOccurrences, event.ID)
		}

		if recurrenceExhausted(event.Recurrence, occurrenceStart, occurrenceNumber) {
			break
		}

		occurrenceEnd := occurrenceStart.Add(duration)
		if !occurrenceStart.Before(toExclusiveLocal) {
			break
		}

		if overlaps(occurrenceStart, occurrenceEnd, fromLocal, toExclusiveLocal) {
			occurrences = append(occurrences, occurrenceForEvent(event, occurrenceStart, occurrenceEnd, true))
		}

		occurrenceStart = nextOccurrenceStart(occurrenceStart, event.Recurrence)
		occurrenceNumber++
	}

	return occurrences, nil
}

func recurrenceExhausted(recurrence Recurrence, occurrenceStart time.Time, occurrenceNumber int) bool {
	if recurrence.Count > 0 && occurrenceNumber > recurrence.Count {
		return true
	}

	if strings.TrimSpace(recurrence.EndsOn) == "" {
		return false
	}

	endsOn, err := time.ParseInLocation(dateLayout, recurrence.EndsOn, occurrenceStart.Location())
	if err != nil {
		return true
	}

	return dateOnly(occurrenceStart).After(endsOn)
}

func occurrenceForEvent(event Event, startsAtLocal time.Time, endsAtLocal time.Time, recurring bool) Occurrence {
	occurrenceID := event.ID
	if recurring {
		occurrenceID = event.ID + ":" + startsAtLocal.Format(localDateTimeLayout)
	}

	return Occurrence{
		EventID:       event.ID,
		OccurrenceID:  occurrenceID,
		Type:          event.Type,
		Title:         event.Title,
		Description:   event.Description,
		LocationName:  event.LocationName,
		Address:       event.Address,
		StartsAtLocal: startsAtLocal,
		EndsAtLocal:   endsAtLocal,
		Timezone:      event.Timezone,
		Recurring:     recurring,
		Recurrence:    event.Recurrence,
	}
}

func nextOccurrenceStart(value time.Time, recurrence Recurrence) time.Time {
	switch recurrence.Frequency {
	case RecurrenceFrequencyDaily:
		return value.AddDate(0, 0, recurrence.Interval)
	case RecurrenceFrequencyWeekly:
		return value.AddDate(0, 0, 7*recurrence.Interval)
	case RecurrenceFrequencyMonthly:
		return value.AddDate(0, recurrence.Interval, 0)
	default:
		return value
	}
}

func overlaps(start time.Time, end time.Time, from time.Time, toExclusive time.Time) bool {
	return end.After(from) && start.Before(toExclusive)
}

func sortOccurrences(occurrences []Occurrence) {
	for index := 1; index < len(occurrences); index++ {
		current := occurrences[index]
		position := index - 1
		for position >= 0 && occurrenceAfter(occurrences[position], current) {
			occurrences[position+1] = occurrences[position]
			position--
		}
		occurrences[position+1] = current
	}
}

func occurrenceAfter(left Occurrence, right Occurrence) bool {
	if left.StartsAtLocal.After(right.StartsAtLocal) {
		return true
	}
	if left.StartsAtLocal.Before(right.StartsAtLocal) {
		return false
	}

	return left.OccurrenceID > right.OccurrenceID
}

func parseEventType(value string) (EventType, error) {
	eventType := EventType(strings.TrimSpace(value))
	switch eventType {
	case EventTypeShow, EventTypeRehearsal, EventTypeRelease, EventTypeMeeting, EventTypeTask, EventTypeOther:
		return eventType, nil
	default:
		return "", fmt.Errorf("invalid calendar event type %q", value)
	}
}

func parseRecurrenceFrequency(value string) (RecurrenceFrequency, error) {
	frequency := RecurrenceFrequency(strings.TrimSpace(value))
	switch frequency {
	case RecurrenceFrequencyNone, RecurrenceFrequencyDaily, RecurrenceFrequencyWeekly, RecurrenceFrequencyMonthly:
		return frequency, nil
	default:
		return "", fmt.Errorf("invalid recurrence frequency %q", value)
	}
}

func parseLocalDate(label string, value string, location *time.Location) (time.Time, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return time.Time{}, fmt.Errorf("%s date is required", label)
	}

	parsed, err := time.ParseInLocation(dateLayout, trimmedValue, location)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s date %q must use YYYY-MM-DD", label, value)
	}

	if parsed.Format(dateLayout) != trimmedValue {
		return time.Time{}, fmt.Errorf("%s date %q must use YYYY-MM-DD", label, value)
	}

	return parsed, nil
}

func parseLocalDateTime(label string, value string, location *time.Location) (time.Time, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return time.Time{}, fmt.Errorf("%s is required", label)
	}

	parsed, err := time.ParseInLocation(localDateTimeLayout, trimmedValue, location)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s %q must use YYYY-MM-DDTHH:mm:ss", label, value)
	}

	if parsed.Format(localDateTimeLayout) != trimmedValue {
		return time.Time{}, fmt.Errorf("%s %q must use YYYY-MM-DDTHH:mm:ss", label, value)
	}

	return parsed, nil
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func validateReadAccount(account AccountContext) error {
	if strings.TrimSpace(account.UserID) == "" {
		return fmt.Errorf("user id is required")
	}

	if strings.TrimSpace(account.BandID) == "" {
		return fmt.Errorf("band id is required")
	}

	if strings.TrimSpace(account.BandTimezone) == "" {
		return fmt.Errorf("band timezone is required")
	}

	if !permissions.CanReadInAlpha(account.Role) {
		return fmt.Errorf("alpha read access denied for role %q", account.Role)
	}

	return nil
}

func validateWriteAccount(account AccountContext) error {
	if err := validateReadAccount(account); err != nil {
		return err
	}

	if err := permissions.RequireAlphaWrite(account.Role); err != nil {
		return err
	}

	return nil
}

func validateMutationMetadata(idempotencyKeyInput string, requestIDInput string, timestamp time.Time) (string, string, error) {
	idempotencyKey := strings.TrimSpace(idempotencyKeyInput)
	if idempotencyKey == "" {
		return "", "", fmt.Errorf("idempotency key is required")
	}

	requestID := strings.TrimSpace(requestIDInput)
	if requestID == "" {
		return "", "", fmt.Errorf("request id is required")
	}

	if timestamp.IsZero() {
		return "", "", fmt.Errorf("mutation timestamp is required")
	}

	return idempotencyKey, requestID, nil
}

func validateID(label string, value string) (string, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return "", fmt.Errorf("%s is required", label)
	}

	if _, err := uuid.Parse(trimmedValue); err != nil {
		return "", fmt.Errorf("%s must be a valid UUID", label)
	}

	return trimmedValue, nil
}
