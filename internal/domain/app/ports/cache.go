package ports

import (
	"context"

	"github.com/wb-go/wbf/retry"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}) error
	GetWithRetry(ctx context.Context, strategy retry.Strategy, key string) (string, error)
	SetWithRetry(ctx context.Context, strategy retry.Strategy, key string, value interface{}) error
	BatchWriter(ctx context.Context, in <-chan [2]string)
}
