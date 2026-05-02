package calendarhandler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	applicationcalendar "github.com/thalys/band-manager/apps/api/internal/application/calendar"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
	authhandler "github.com/thalys/band-manager/apps/api/internal/transport/http/auth"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware"
)

type Handler struct {
	authenticator      session.Authenticator
	accountRepository  accounts.BandAccountRepository
	calendarRepository applicationcalendar.Repository
	logger             *slog.Logger
	now                func() time.Time
}

type EventRequest struct {
	Type          string            `json:"type"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	LocationName  string            `json:"locationName"`
	Address       string            `json:"address"`
	StartsAtLocal string            `json:"startsAtLocal"`
	EndsAtLocal   string            `json:"endsAtLocal"`
	Recurrence    RecurrenceRequest `json:"recurrence"`
}

type RecurrenceRequest struct {
	Frequency string `json:"frequency"`
	Interval  *int   `json:"interval"`
	EndsOn    string `json:"endsOn"`
	Count     *int   `json:"count"`
}

type EventListResponse struct {
	Range  EventRangeResponse   `json:"range"`
	Events []OccurrenceResponse `json:"events"`
}

type EventRangeResponse struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Timezone string `json:"timezone"`
}

type EventResponse struct {
	ID            string             `json:"id"`
	BandID        string             `json:"bandId"`
	Type          string             `json:"type"`
	Title         string             `json:"title"`
	Description   string             `json:"description"`
	LocationName  string             `json:"locationName"`
	Address       string             `json:"address"`
	StartsAtLocal string             `json:"startsAtLocal"`
	EndsAtLocal   string             `json:"endsAtLocal"`
	Timezone      string             `json:"timezone"`
	Recurrence    RecurrenceResponse `json:"recurrence"`
	CreatedAt     time.Time          `json:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt"`
}

type OccurrenceResponse struct {
	EventID       string             `json:"eventId"`
	OccurrenceID  string             `json:"occurrenceId"`
	Type          string             `json:"type"`
	Title         string             `json:"title"`
	Description   string             `json:"description"`
	LocationName  string             `json:"locationName"`
	Address       string             `json:"address"`
	StartsAtLocal string             `json:"startsAtLocal"`
	EndsAtLocal   string             `json:"endsAtLocal"`
	Timezone      string             `json:"timezone"`
	Recurring     bool               `json:"recurring"`
	Recurrence    RecurrenceResponse `json:"recurrence"`
}

type RecurrenceResponse struct {
	Frequency string `json:"frequency"`
	Interval  int    `json:"interval"`
	EndsOn    string `json:"endsOn,omitempty"`
	Count     int    `json:"count,omitempty"`
}

func NewHandler(authenticator session.Authenticator, accountRepository accounts.BandAccountRepository, calendarRepository applicationcalendar.Repository, logger *slog.Logger) Handler {
	return Handler{
		authenticator:      authenticator,
		accountRepository:  accountRepository,
		calendarRepository: calendarRepository,
		logger:             logger,
		now:                time.Now,
	}
}

func (handler Handler) ListEvents(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	eventList, err := applicationcalendar.ListEvents(request.Context(), handler.calendarRepository, applicationcalendar.ListEventsInput{
		Account: accountContext,
		From:    request.URL.Query().Get("from"),
		To:      request.URL.Query().Get("to"),
	})
	if err != nil {
		handler.writeCalendarError(response, "calendar list failed", err)
		return
	}

	handler.writeJSON(response, http.StatusOK, toEventListResponse(eventList))
}

func (handler Handler) GetEvent(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	event, err := applicationcalendar.GetEvent(request.Context(), handler.calendarRepository, applicationcalendar.GetEventInput{
		Account: accountContext,
		EventID: chi.URLParam(request, "eventID"),
	})
	if err != nil {
		handler.writeCalendarError(response, "calendar get failed", err)
		return
	}

	handler.writeJSON(response, http.StatusOK, toEventResponse(event))
}

func (handler Handler) CreateEvent(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	var body EventRequest
	if !handler.decodeJSON(response, request, &body) {
		return
	}

	event, err := applicationcalendar.CreateEvent(request.Context(), handler.calendarRepository, applicationcalendar.CreateEventInput{
		Account:        accountContext,
		Type:           body.Type,
		Title:          body.Title,
		Description:    body.Description,
		LocationName:   body.LocationName,
		Address:        body.Address,
		StartsAtLocal:  body.StartsAtLocal,
		EndsAtLocal:    body.EndsAtLocal,
		Recurrence:     toRecurrenceInput(body.Recurrence),
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		CreatedAt:      handler.now().UTC(),
	})
	if err != nil {
		handler.writeCalendarError(response, "calendar create failed", err)
		return
	}

	handler.writeJSON(response, http.StatusCreated, toEventResponse(event))
}

func (handler Handler) UpdateEvent(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	var body EventRequest
	if !handler.decodeJSON(response, request, &body) {
		return
	}

	event, err := applicationcalendar.UpdateEvent(request.Context(), handler.calendarRepository, applicationcalendar.UpdateEventInput{
		Account:        accountContext,
		EventID:        chi.URLParam(request, "eventID"),
		Type:           body.Type,
		Title:          body.Title,
		Description:    body.Description,
		LocationName:   body.LocationName,
		Address:        body.Address,
		StartsAtLocal:  body.StartsAtLocal,
		EndsAtLocal:    body.EndsAtLocal,
		Recurrence:     toRecurrenceInput(body.Recurrence),
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		UpdatedAt:      handler.now().UTC(),
	})
	if err != nil {
		handler.writeCalendarError(response, "calendar update failed", err)
		return
	}

	handler.writeJSON(response, http.StatusOK, toEventResponse(event))
}

func (handler Handler) SoftDeleteEvent(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	err := applicationcalendar.SoftDeleteEvent(request.Context(), handler.calendarRepository, applicationcalendar.DeleteEventInput{
		Account:        accountContext,
		EventID:        chi.URLParam(request, "eventID"),
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		DeletedAt:      handler.now().UTC(),
	})
	if err != nil {
		handler.writeCalendarError(response, "calendar delete failed", err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler Handler) accountContext(response http.ResponseWriter, request *http.Request) (applicationcalendar.AccountContext, bool) {
	authenticatedUser, ok := handler.authenticate(response, request)
	if !ok {
		return applicationcalendar.AccountContext{}, false
	}

	account, err := accounts.GetCurrentAccount(request.Context(), handler.accountRepository, accounts.CurrentAccountQuery{
		AuthProvider:       authenticatedUser.Provider,
		AuthProviderUserID: authenticatedUser.ProviderUserID,
	})
	if err != nil {
		handler.logger.Warn("calendar account lookup failed", "error", err, "provider", authenticatedUser.Provider, "provider_user_id", authenticatedUser.ProviderUserID)
		handler.writeError(response, http.StatusUnauthorized, "account_not_found", "Authenticated user is not linked to a band account")
		return applicationcalendar.AccountContext{}, false
	}

	return applicationcalendar.AccountContext{
		UserID:       account.UserID,
		BandID:       account.BandID,
		BandTimezone: account.BandTimezone,
		Role:         account.Role,
	}, true
}

func (handler Handler) authenticate(response http.ResponseWriter, request *http.Request) (session.AuthenticatedUser, bool) {
	token, err := session.NormalizeBearerToken(request.Header.Get("Authorization"))
	if err != nil {
		handler.writeError(response, http.StatusUnauthorized, "invalid_authorization", err.Error())
		return session.AuthenticatedUser{}, false
	}

	authenticatedUser, err := handler.authenticator.Authenticate(request.Context(), token)
	if err != nil {
		handler.writeError(response, http.StatusUnauthorized, "invalid_session", err.Error())
		return session.AuthenticatedUser{}, false
	}

	return authenticatedUser, true
}

func (handler Handler) mutationHeaders(response http.ResponseWriter, request *http.Request) (string, string, bool) {
	idempotencyKey := strings.TrimSpace(request.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		handler.writeError(response, http.StatusBadRequest, "missing_idempotency_key", "Idempotency-Key header is required")
		return "", "", false
	}

	requestID, ok := middleware.RequestIDFromContext(request.Context())
	if !ok {
		handler.writeError(response, http.StatusInternalServerError, "missing_request_id", "request id is missing")
		return "", "", false
	}

	return idempotencyKey, requestID, true
}

func (handler Handler) decodeJSON(response http.ResponseWriter, request *http.Request, target interface{}) bool {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		handler.writeError(response, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return false
	}

	return true
}

func (handler Handler) writeCalendarError(response http.ResponseWriter, logMessage string, err error) {
	handler.logger.Warn(logMessage, "error", err)

	switch {
	case errors.Is(err, applicationcalendar.ErrCalendarEventNotFound):
		handler.writeError(response, http.StatusNotFound, "calendar_event_not_found", err.Error())
	case strings.Contains(err.Error(), "alpha write access requires owner role"):
		handler.writeError(response, http.StatusForbidden, "write_forbidden", err.Error())
	case strings.Contains(err.Error(), "alpha read access denied"):
		handler.writeError(response, http.StatusForbidden, "read_forbidden", err.Error())
	case strings.Contains(err.Error(), "list calendar events"),
		strings.Contains(err.Error(), "get calendar event"),
		strings.Contains(err.Error(), "create calendar event"),
		strings.Contains(err.Error(), "update calendar event"),
		strings.Contains(err.Error(), "soft delete calendar event"):
		handler.writeError(response, http.StatusInternalServerError, "calendar_failed", err.Error())
	default:
		handler.writeError(response, http.StatusBadRequest, "invalid_calendar_request", err.Error())
	}
}

func (handler Handler) writeJSON(response http.ResponseWriter, statusCode int, body interface{}) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)

	if err := json.NewEncoder(response).Encode(body); err != nil {
		handler.logger.Error("calendar response encoding failed", "error", err)
	}
}

func (handler Handler) writeError(response http.ResponseWriter, statusCode int, code string, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)

	err := json.NewEncoder(response).Encode(authhandler.ErrorResponse{
		Code:    code,
		Message: message,
	})
	if err != nil {
		handler.logger.Error("calendar error response encoding failed", "error", err, "code", code, "status_code", statusCode)
	}
}

func toRecurrenceInput(request RecurrenceRequest) applicationcalendar.RecurrenceInput {
	interval := 0
	if request.Interval != nil {
		interval = *request.Interval
	}

	count := 0
	if request.Count != nil {
		count = *request.Count
	}

	return applicationcalendar.RecurrenceInput{
		Frequency: request.Frequency,
		Interval:  interval,
		EndsOn:    request.EndsOn,
		Count:     count,
	}
}

func toEventListResponse(eventList applicationcalendar.EventList) EventListResponse {
	return EventListResponse{
		Range: EventRangeResponse{
			From:     eventList.Range.From,
			To:       eventList.Range.To,
			Timezone: eventList.Range.Timezone,
		},
		Events: toOccurrenceResponses(eventList.Events),
	}
}

func toEventResponse(event applicationcalendar.Event) EventResponse {
	return EventResponse{
		ID:            event.ID,
		BandID:        event.BandID,
		Type:          string(event.Type),
		Title:         event.Title,
		Description:   event.Description,
		LocationName:  event.LocationName,
		Address:       event.Address,
		StartsAtLocal: formatLocalDateTime(event.StartsAtLocal),
		EndsAtLocal:   formatLocalDateTime(event.EndsAtLocal),
		Timezone:      event.Timezone,
		Recurrence:    toRecurrenceResponse(event.Recurrence),
		CreatedAt:     event.CreatedAt,
		UpdatedAt:     event.UpdatedAt,
	}
}

func toOccurrenceResponses(occurrences []applicationcalendar.Occurrence) []OccurrenceResponse {
	responses := make([]OccurrenceResponse, 0, len(occurrences))
	for _, occurrence := range occurrences {
		responses = append(responses, OccurrenceResponse{
			EventID:       occurrence.EventID,
			OccurrenceID:  occurrence.OccurrenceID,
			Type:          string(occurrence.Type),
			Title:         occurrence.Title,
			Description:   occurrence.Description,
			LocationName:  occurrence.LocationName,
			Address:       occurrence.Address,
			StartsAtLocal: formatLocalDateTime(occurrence.StartsAtLocal),
			EndsAtLocal:   formatLocalDateTime(occurrence.EndsAtLocal),
			Timezone:      occurrence.Timezone,
			Recurring:     occurrence.Recurring,
			Recurrence:    toRecurrenceResponse(occurrence.Recurrence),
		})
	}

	return responses
}

func toRecurrenceResponse(recurrence applicationcalendar.Recurrence) RecurrenceResponse {
	return RecurrenceResponse{
		Frequency: string(recurrence.Frequency),
		Interval:  recurrence.Interval,
		EndsOn:    recurrence.EndsOn,
		Count:     recurrence.Count,
	}
}

func formatLocalDateTime(value time.Time) string {
	return value.Format("2006-01-02T15:04:05")
}
