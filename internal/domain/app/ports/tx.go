package ports

import "context"

// TxManager runs fn in a transaction carried by ctx; repositories called
// with that ctx take part in it.
type TxManager interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
