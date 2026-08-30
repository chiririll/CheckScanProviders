package provider

import (
	"context"

	"github.com/chiririll/CheckScanProviders/pkg/eq"
)

type remoteKey struct{}
type waitKey struct{}

func WithRemote(ctx context.Context, remote bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, remoteKey{}, remote)
}

func Remote(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	remote, _ := ctx.Value(remoteKey{}).(bool)
	return remote
}

func WithWait(ctx context.Context, wait bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, waitKey{}, wait)
}

func Wait(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	wait, _ := ctx.Value(waitKey{}).(bool)
	return wait
}

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
