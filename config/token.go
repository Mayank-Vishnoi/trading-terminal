package config

import "context"

type contextKey string

var contextKeyAuthtoken = contextKey("auth-token")

func SetToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, contextKeyAuthtoken, token)
}

func GetToken(ctx context.Context) (string, bool) {
	tokenStr, ok := ctx.Value(contextKeyAuthtoken).(string)
	return tokenStr, ok
}