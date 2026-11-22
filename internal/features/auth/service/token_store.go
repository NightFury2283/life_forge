package auth_service

import (
	"context"
	"golang.org/x/oauth2"
)

// GoogleTokenStore associates provider credentials with an authenticated user.
type GoogleTokenStore interface {
	SaveGoogleToken(context.Context, int, *oauth2.Token) error
	GetGoogleToken(context.Context, int) (*oauth2.Token, error)
}
