package auth

import (
	"butler-server/models"
	"context"
)

type contextKey string

const userKey contextKey = "user"

// SetUser adds a user to the context
func SetUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// GetUser retrieves the authenticated user from context
func GetUser(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userKey).(*models.User)
	return user, ok
}

// MustGetUser retrieves the authenticated user from context and panics if not found
func MustGetUser(ctx context.Context) *models.User {
	user, ok := GetUser(ctx)
	if !ok {
		panic("no authenticated user found in context")
	}
	return user
}
