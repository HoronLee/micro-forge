package authn

import "context"

type authenticationKey struct{}

func withAuthentication(ctx context.Context, authentication Authentication) context.Context {
	return context.WithValue(ctx, authenticationKey{}, authentication)
}

// AuthenticationFrom reads the standard validated authentication result.
func AuthenticationFrom(ctx context.Context) (Authentication, bool) {
	authentication, ok := ctx.Value(authenticationKey{}).(Authentication)
	if !ok || authentication.Subject == "" {
		return Authentication{}, false
	}
	return authentication, true
}

// SubjectFrom reads the validated subject from the standard authentication result.
func SubjectFrom(ctx context.Context) (string, bool) {
	authentication, ok := AuthenticationFrom(ctx)
	if !ok {
		return "", false
	}
	return authentication.Subject, true
}
