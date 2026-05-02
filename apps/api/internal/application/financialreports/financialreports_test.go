package financialreports

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

func TestListReportDefaultsToLastThreeMonthsInBandTimezone(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	_, err := ListReport(context.Background(), &repository, ListReportInput{
		Account: validAccountContext(),
		Now:     time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("list report: %v", err)
	}

	if repository.query.Range.From != "2026-02-02" {
		t.Fatalf("expected default from date, got %q", repository.query.Range.From)
	}
	if repository.query.Range.To != "2026-05-02" {
		t.Fatalf("expected default to date, got %q", repository.query.Range.To)
	}
	if repository.query.Range.Timezone != "America/Recife" {
		t.Fatalf("expected timezone, got %q", repository.query.Range.Timezone)
	}
}

func TestListReportParsesExplicitDatesAsLocalBandDates(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	_, err := ListReport(context.Background(), &repository, ListReportInput{
		Account: validAccountContext(),
		From:    "2026-05-01",
		To:      "2026-05-03",
		Now:     time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("list report: %v", err)
	}

	expectedFromUTC := time.Date(2026, 5, 1, 3, 0, 0, 0, time.UTC)
	if !repository.query.FromUTC.Equal(expectedFromUTC) {
		t.Fatalf("expected from UTC %s, got %s", expectedFromUTC, repository.query.FromUTC)
	}

	expectedToExclusiveUTC := time.Date(2026, 5, 4, 3, 0, 0, 0, time.UTC)
	if !repository.query.ToExclusiveUTC.Equal(expectedToExclusiveUTC) {
		t.Fatalf("expected exclusive to UTC %s, got %s", expectedToExclusiveUTC, repository.query.ToExclusiveUTC)
	}
}

func TestListReportRejectsInvalidDates(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	_, err := ListReport(context.Background(), &repository, ListReportInput{
		Account: validAccountContext(),
		From:    "2026/05/01",
		Now:     time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected invalid date error")
	}
}

func TestListReportRejectsFromAfterTo(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	_, err := ListReport(context.Background(), &repository, ListReportInput{
		Account: validAccountContext(),
		From:    "2026-05-04",
		To:      "2026-05-03",
		Now:     time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected date range error")
	}
}

func TestListReportAllowsViewerReadAccess(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	account := validAccountContext()
	account.Role = permissions.RoleViewer

	_, err := ListReport(context.Background(), &repository, ListReportInput{
		Account: account,
		Now:     time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("list report: %v", err)
	}
}

func TestListReportReturnsRepositoryErrorWithContext(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{err: errors.New("database unavailable")}
	_, err := ListReport(context.Background(), &repository, ListReportInput{
		Account: validAccountContext(),
		Now:     time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected repository error")
	}
}

type fakeRepository struct {
	query ReportQuery
	err   error
}

func (repository *fakeRepository) GetReport(ctx context.Context, query ReportQuery) (Report, error) {
	repository.query = query
	if repository.err != nil {
		return Report{}, repository.err
	}

	return Report{Range: query.Range}, nil
}

func validAccountContext() AccountContext {
	return AccountContext{
		UserID:       "user_1",
		BandID:       "band_1",
		BandTimezone: "America/Recife",
		Role:         permissions.RoleOwner,
	}
}
