package provider

import (
	"context"

	"github.com/chiririll/CheckScanProviders/pkg/eq"
)

type Provider interface {
	ID() string
	Label() string
	CanHandle(rawQR string) (hash string, ok bool)
	Parse(ctx context.Context, rawQR string) (*eq.Receipt, error)
}

type Registry struct {
	items []Provider
}

func NewRegistry(items ...Provider) *Registry {
	return &Registry{items: items}
}

func (r *Registry) All() []Provider {
	return r.items
}

func (r *Registry) ByID(id string) (Provider, bool) {
	for _, item := range r.items {
		if item.ID() == id {
			return item, true
		}
	}
	return nil, false
}
