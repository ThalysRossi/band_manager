package idempotency

import (
	"context"
	"time"
)

type Operation string

const (
	OperationMutation         Operation = "mutation"
	OperationCheckout         Operation = "checkout"
	OperationPaymentCreation  Operation = "payment_creation"
	OperationSaleFinalization Operation = "sale_finalization"
	OperationWebhook          Operation = "webhook"
)

type KeyScope struct {
	BandID    string
	Operation Operation
	Key       string
}

type StoredResult struct {
	RequestHash  string
	ResponseBody []byte
	StatusCode   int
	ExpiresAt    time.Time
}

type Store interface {
	Find(ctx context.Context, scope KeyScope) (StoredResult, bool, error)
	Save(ctx context.Context, scope KeyScope, result StoredResult) error
}
