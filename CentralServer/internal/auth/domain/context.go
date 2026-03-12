package domain

import "context"

type authContextKey string

const (
	userIDKey   authContextKey = "auth.user_id"
	usernameKey authContextKey = "auth.username"
)

// WithAuthContext stores auth identity in context.
func WithAuthContext(ctx context.Context, userID, username string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, usernameKey, username)
	return ctx
}

// UserIDFromContext retrieves user id from context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	if v := ctx.Value(userIDKey); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s, true
		}
	}
	return "", false
}

// UsernameFromContext retrieves username from context.
func UsernameFromContext(ctx context.Context) (string, bool) {
	if v := ctx.Value(usernameKey); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s, true
		}
	}
	return "", false
}