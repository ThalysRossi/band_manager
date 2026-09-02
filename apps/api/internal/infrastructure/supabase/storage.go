package supabase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	applicationinventory "github.com/thalys/band-manager/apps/api/internal/application/inventory"
)

const signedUploadExpiration = 2 * time.Hour

type StorageClient struct {
	storageURL      string
	publicObjectURL string
	bucket          string
	secretKey       string
	httpClient      *http.Client
	logger          *slog.Logger
	now             func() time.Time
}

type signedUploadResponse struct {
	URL string `json:"url"`
}

type objectInfoResponse struct {
	Size        *int   `json:"size"`
	ContentType string `json:"content_type"`
}

func NewStorageClient(supabaseURL string, secretKey string, bucket string, httpClient *http.Client, logger *slog.Logger) (StorageClient, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(supabaseURL), "/")
	if baseURL == "" {
		return StorageClient{}, fmt.Errorf("supabase url is required")
	}

	trimmedSecretKey := strings.TrimSpace(secretKey)
	if trimmedSecretKey == "" {
		return StorageClient{}, fmt.Errorf("supabase secret key is required")
	}

	trimmedBucket := strings.TrimSpace(bucket)
	if trimmedBucket == "" {
		return StorageClient{}, fmt.Errorf("supabase storage bucket is required")
	}

	if httpClient == nil {
		return StorageClient{}, fmt.Errorf("supabase storage http client is required")
	}

	if logger == nil {
		return StorageClient{}, fmt.Errorf("supabase storage logger is required")
	}

	storageURL := baseURL + "/storage/v1"
	return StorageClient{
		storageURL:      storageURL,
		publicObjectURL: storageURL + "/object/public/" + pathEscape(trimmedBucket),
		bucket:          trimmedBucket,
		secretKey:       trimmedSecretKey,
		httpClient:      httpClient,
		logger:          logger,
		now:             time.Now,
	}, nil
}

func (client StorageClient) CreateSignedUpload(ctx context.Context, command applicationinventory.CreatePhotoUploadCommand) (applicationinventory.SignedPhotoUpload, error) {
	objectKey := strings.TrimSpace(command.ObjectKey)
	if objectKey == "" {
		return applicationinventory.SignedPhotoUpload{}, fmt.Errorf("storage object key is required")
	}

	requestURL := client.storageURL + "/object/upload/sign/" + pathEscape(client.bucket) + "/" + pathEscapeSegments(objectKey)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, strings.NewReader("{}"))
	if err != nil {
		return applicationinventory.SignedPhotoUpload{}, fmt.Errorf("create supabase signed upload request url=%q object_key=%q: %w", requestURL, objectKey, err)
	}
	client.applyStorageHeaders(request)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return applicationinventory.SignedPhotoUpload{}, fmt.Errorf("execute supabase signed upload request url=%q object_key=%q: %w", requestURL, objectKey, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return applicationinventory.SignedPhotoUpload{}, fmt.Errorf("read supabase signed upload response url=%q object_key=%q status_code=%d: %w", requestURL, objectKey, response.StatusCode, err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return applicationinventory.SignedPhotoUpload{}, fmt.Errorf("supabase signed upload request failed url=%q object_key=%q status_code=%d response_body=%q", requestURL, objectKey, response.StatusCode, string(body))
	}

	var payload signedUploadResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return applicationinventory.SignedPhotoUpload{}, fmt.Errorf("parse supabase signed upload response url=%q object_key=%q status_code=%d response_body=%q: %w", requestURL, objectKey, response.StatusCode, string(body), err)
	}

	signedURL, token, err := client.signedUploadURL(payload.URL)
	if err != nil {
		return applicationinventory.SignedPhotoUpload{}, fmt.Errorf("parse supabase signed upload url object_key=%q raw_url=%q: %w", objectKey, payload.URL, err)
	}

	return applicationinventory.SignedPhotoUpload{
		ObjectKey: objectKey,
		SignedURL: signedURL,
		Token:     token,
		ExpiresAt: client.now().UTC().Add(signedUploadExpiration),
		PublicURL: client.PublicURL(objectKey),
	}, nil
}

func (client StorageClient) GetObjectInfo(ctx context.Context, query applicationinventory.PhotoObjectInfoQuery) (applicationinventory.PhotoObjectInfo, error) {
	objectKey := strings.TrimSpace(query.ObjectKey)
	if objectKey == "" {
		return applicationinventory.PhotoObjectInfo{}, fmt.Errorf("storage object key is required")
	}

	requestURL := client.storageURL + "/object/info/" + pathEscape(client.bucket) + "/" + pathEscapeSegments(objectKey)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return applicationinventory.PhotoObjectInfo{}, fmt.Errorf("create supabase object info request url=%q object_key=%q: %w", requestURL, objectKey, err)
	}
	client.applyStorageHeaders(request)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return applicationinventory.PhotoObjectInfo{}, fmt.Errorf("execute supabase object info request url=%q object_key=%q: %w", requestURL, objectKey, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return applicationinventory.PhotoObjectInfo{}, fmt.Errorf("read supabase object info response url=%q object_key=%q status_code=%d: %w", requestURL, objectKey, response.StatusCode, err)
	}
	if response.StatusCode == http.StatusNotFound {
		return applicationinventory.PhotoObjectInfo{}, fmt.Errorf("%w: object_key=%q", applicationinventory.ErrPhotoObjectNotFound, objectKey)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return applicationinventory.PhotoObjectInfo{}, fmt.Errorf("supabase object info request failed url=%q object_key=%q status_code=%d response_body=%q", requestURL, objectKey, response.StatusCode, string(body))
	}

	var payload objectInfoResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return applicationinventory.PhotoObjectInfo{}, fmt.Errorf("parse supabase object info response url=%q object_key=%q status_code=%d response_body=%q: %w", requestURL, objectKey, response.StatusCode, string(body), err)
	}
	if payload.Size == nil {
		return applicationinventory.PhotoObjectInfo{}, fmt.Errorf("supabase object info size is required object_key=%q response_body=%q", objectKey, string(body))
	}
	if strings.TrimSpace(payload.ContentType) == "" {
		return applicationinventory.PhotoObjectInfo{}, fmt.Errorf("supabase object info content type is required object_key=%q response_body=%q", objectKey, string(body))
	}

	return applicationinventory.PhotoObjectInfo{
		ObjectKey:   objectKey,
		ContentType: strings.TrimSpace(payload.ContentType),
		SizeBytes:   *payload.Size,
	}, nil
}

func (client StorageClient) PublicURL(objectKey string) string {
	return client.publicObjectURL + "/" + pathEscapeSegments(strings.TrimSpace(objectKey))
}

func (client StorageClient) applyStorageHeaders(request *http.Request) {
	request.Header.Set("Authorization", "Bearer "+client.secretKey)
	request.Header.Set("apikey", client.secretKey)
	request.Header.Set("Content-Type", "application/json")
}

func (client StorageClient) signedUploadURL(rawURL string) (string, string, error) {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return "", "", errors.New("signed upload url is required")
	}

	signedURL := trimmedURL
	if strings.HasPrefix(trimmedURL, "/") {
		signedURL = client.storageURL + trimmedURL
	}

	parsedURL, err := url.Parse(signedURL)
	if err != nil {
		return "", "", err
	}

	token := strings.TrimSpace(parsedURL.Query().Get("token"))
	if token == "" {
		return "", "", errors.New("signed upload token is required")
	}

	return parsedURL.String(), token, nil
}

func pathEscape(value string) string {
	return url.PathEscape(strings.TrimSpace(value))
}

func pathEscapeSegments(path string) string {
	segments := strings.Split(strings.TrimSpace(path), "/")
	escapedSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		escapedSegments = append(escapedSegments, url.PathEscape(segment))
	}

	return strings.Join(escapedSegments, "/")
}
