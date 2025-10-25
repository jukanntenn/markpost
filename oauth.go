package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

var oauthConfig *oauth2.Config

func initOAuthConfig() {
	oauthConfig = &oauth2.Config{
		ClientID:     config.GitHub.ClientID,
		ClientSecret: config.GitHub.ClientSecret,
		RedirectURL:  config.GitHub.RedirectURL,
		Scopes:       []string{},
		Endpoint:     github.Endpoint,
	}
}

func getGitHubUser(token *oauth2.Token) (*GitHubUser, error) {
	ctx := context.Background()
	client := oauthConfig.Client(ctx, token)

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var githubUser GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, err
	}

	log.Printf("GitHub OAuth Debug - User data: ID=%d, Login='%s'", githubUser.ID, githubUser.Login)

	if githubUser.ID == 0 || githubUser.Login == "" {
		log.Printf("GitHub OAuth Error - Invalid user data: ID=%d, Login='%s'", githubUser.ID, githubUser.Login)
		return nil, fmt.Errorf("invalid GitHub user data: ID=%d, Login='%s'", githubUser.ID, githubUser.Login)
	}

	return &githubUser, nil
}