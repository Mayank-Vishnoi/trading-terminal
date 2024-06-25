package config

/*
 * This is just a simple way to store the token in the context.
 * Previous flow:
 * Hit the login endpoint to get the authorization code
 * Save that code in the .env file
 * Restart the server
 * Get the token in the main context and pass it around.
 */

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