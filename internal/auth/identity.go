package auth

import (
	"context"
)

type keyType struct{}

var key = keyType{}

func With(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, key, id)
}

func From(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(key).(Identity)
	return id, ok
}
