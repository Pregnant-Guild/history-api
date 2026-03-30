package oauth

import (
	"fmt"
	"history-api/pkg/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func NewGoogleProvider() (*oauth2.Config, error) {
	userGoogle, err := config.GetConfig("GOOGLE_CLIENT_ID")
	if err != nil {
		return nil, err
	}

	passGoogle, err := config.GetConfig("GOOGLE_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}

	redirectURL, err := config.GetConfig("GOOGLE_REDIRECT_URL")
	if err != nil {
		return nil, err
	}

	return &oauth2.Config{
		RedirectURL:  redirectURL,
		ClientID:     fmt.Sprintf("%s.apps.googleusercontent.com", userGoogle),
		ClientSecret: passGoogle,
		Scopes: []string{
			"openid",
			"email",
			"profile",
		},
		Endpoint: google.Endpoint,
	}, nil
}
