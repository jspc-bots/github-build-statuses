package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/google/go-github/v35/github"
	"github.com/lrstanley/girc"
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

func ircClient(user, password, server string, verify bool) (c *girc.Client) {
	u, err := url.Parse(server)
	if err != nil {
		return
	}

	config := girc.Config{
		Server: u.Hostname(),
		Port:   must(strconv.Atoi(u.Port())).(int),
		Nick:   Nick,
		User:   Nick,
		Name:   Nick,
		SASL: &girc.SASLPlain{
			User: user,
			Pass: password,
		},
		SSL: u.Scheme == "ircs",
		TLSConfig: &tls.Config{
			InsecureSkipVerify: !verify,
		},
	}

	c = girc.New(config)
	c.Handlers.Add(girc.CONNECTED, func(c *girc.Client, e girc.Event) {
		c.Cmd.Join(Chan)
	})

	return
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

	irc := ircClient(Username, Password, Server, VerifyTLS)
	if err != nil {
		panic(err)
	}

	go func(c *girc.Client) {
		log.Panic(c.Connect())
	}(irc)

	gh := githubClient(GithubToken)
	if err != nil {
		panic(err)
	}

	ticker := time.NewTicker(time.Second * time.Duration(seconds))

	for range ticker.C {
		log.Print("loading notifications")

		notifications, _, err := gh.Activity.ListNotifications(context.Background(), &github.NotificationListOptions{All: false})
		if err != nil {
			panic(err)
		}

		gh.Activity.MarkNotificationsRead(context.Background(), time.Now())

		log.Printf("loaded %d notifications", len(notifications))

		for _, n := range notifications {
			// We're only tracking CI activity right now
			if *n.Reason != "ci_activity" {
				continue
			}

			log.Printf("sending: %q", *n.Subject.Title)
			irc.Cmd.Message(Chan, *n.Subject.Title)
		}
	}
}
