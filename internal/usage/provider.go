package usage

import "context"

// Provider is the abstraction every usage source must implement.
type Provider interface {
	Name() string
	Fetch(ctx context.Context) (ProviderUsage, error)
}
