package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/go-github/v35/github"
	"golang.org/x/oauth2"
)

const (
	Nick = "build-bot"
	Chan = "#dashboard"
)

var (
	Username    = os.Getenv("SASL_USER")
	Password    = os.Getenv("SASL_PASSWORD")
	Server      = os.Getenv("SERVER")
	VerifyTLS   = os.Getenv("VERIFY_TLS") == "true"
	GithubToken = os.Getenv("GITHUB_TOKEN")
	PollPeriod  = os.Getenv("POLL_PERIOD")
)

func must(i interface{}, err error) interface{} {
	if err != nil {
		panic(err)
	}

	return i
}

func githubClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	tc := oauth2.NewClient(context.Background(), ts)

	return github.NewClient(tc)
}

func main() {
	seconds, err := strconv.Atoi(PollPeriod)
	if err != nil {
		panic(err)
	}

	c, err := New(Username, Password, Server, VerifyTLS, githubClient(GithubToken))
	if err != nil {
		panic(err)
	}

	go c.client.Connect()

	ticker := time.NewTicker(time.Second * time.Duration(seconds))

	for range ticker.C {
		err = c.processNotifications()
		if err != nil {
			log.Print(err)
		}
	}
}
