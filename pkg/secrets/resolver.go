package secrets

import (
	"context"
)

type Resolver interface {
	Resolve(ctx context.Context, path string) (string, error)
}
