package middleware

import "context"

type contextKey string

const (
	authIdentityContextKey contextKey = "auth_identity"
	requestIDContextKey    contextKey = "request_id"
)

func GetRequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}
