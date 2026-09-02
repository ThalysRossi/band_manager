package supabase

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	applicationinventory "github.com/thalys/band-manager/apps/api/internal/application/inventory"
)

func TestStorageClientCreatesSignedUpload(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/storage/v1/object/upload/sign/inventory-photos/bands/band_1/photo.webp" {
			t.Fatalf("unexpected path %q", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer secret-key" {
			t.Fatalf("expected secret key authorization header")
		}

		return jsonResponse(http.StatusOK, `{"url":"/object/upload/sign/inventory-photos/bands/band_1/photo.webp?token=upload-token"}`), nil
	})}

	client, err := NewStorageClient("https://example.supabase.co", "secret-key", "inventory-photos", httpClient, slog.Default())
	if err != nil {
		t.Fatalf("new storage client: %v", err)
	}
	client.now = func() time.Time {
		return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	}

	upload, err := client.CreateSignedUpload(context.Background(), applicationinventory.CreatePhotoUploadCommand{
		ObjectKey: "bands/band_1/photo.webp",
	})
	if err != nil {
		t.Fatalf("create signed upload: %v", err)
	}

	if upload.Token != "upload-token" {
		t.Fatalf("expected upload token, got %q", upload.Token)
	}
	if upload.ExpiresAt != time.Date(2026, 5, 1, 14, 0, 0, 0, time.UTC) {
		t.Fatalf("expected two hour expiration, got %s", upload.ExpiresAt)
	}
}

func TestStorageClientReadsObjectInfo(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/storage/v1/object/info/inventory-photos/bands/band_1/photo.webp" {
			t.Fatalf("unexpected path %q", request.URL.Path)
		}

		return jsonResponse(http.StatusOK, `{"content_type":"image/webp","size":1024}`), nil
	})}

	client, err := NewStorageClient("https://example.supabase.co", "secret-key", "inventory-photos", httpClient, slog.Default())
	if err != nil {
		t.Fatalf("new storage client: %v", err)
	}

	info, err := client.GetObjectInfo(context.Background(), applicationinventory.PhotoObjectInfoQuery{
		ObjectKey: "bands/band_1/photo.webp",
	})
	if err != nil {
		t.Fatalf("get object info: %v", err)
	}

	if info.ContentType != "image/webp" {
		t.Fatalf("expected image/webp, got %q", info.ContentType)
	}
	if info.SizeBytes != 1024 {
		t.Fatalf("expected size 1024, got %d", info.SizeBytes)
	}
}

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
