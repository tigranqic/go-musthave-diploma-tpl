package auth

import (
	"context"
)

type ContextKey string

const IdentityKey ContextKey = "auth.identity"

func With(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, IdentityKey, id)
}

func From(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(IdentityKey).(Identity)
	return id, ok
}
