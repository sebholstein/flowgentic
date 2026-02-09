package interceptors

import (
	"context"
	"crypto/subtle"
	"errors"

	"connectrpc.com/connect"
)

const bearerPrefix = "Bearer "

// NewAuth returns a Connect interceptor that validates the shared
// secret on incoming requests and attaches it on outgoing client requests.
func NewAuth(secret string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().IsClient {
				req.Header().Set("Authorization", bearerPrefix+secret)
				return next(ctx, req)
			}

			token := extractBearer(req.Header().Get("Authorization"))
			if subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or missing authorization"))
			}
			return next(ctx, req)
		}
	}
}

// extractBearer strips the "Bearer " prefix from the Authorization header value.
func extractBearer(val string) string {
	if len(val) > len(bearerPrefix) && val[:len(bearerPrefix)] == bearerPrefix {
		return val[len(bearerPrefix):]
	}
	return ""
}
