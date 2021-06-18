package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/google/go-github/v35/github"
	"github.com/lrstanley/girc"
)

type Bot struct {
	client  *girc.Client
	github  *github.Client
	routing map[*regexp.Regexp]handlerFunc
}

type handlerFunc func(groups [][]byte) error

func New(user, password, server string, verify bool, gh *github.Client) (b Bot, err error) {
	b.github = gh

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

	b.client = girc.New(config)
	err = b.addHandlers()

	return
}

func (b *Bot) addHandlers() (err error) {
	b.client.Handlers.Add(girc.CONNECTED, func(c *girc.Client, e girc.Event) {
		c.Cmd.Join(Chan)
	})

	b.routing = make(map[*regexp.Regexp]handlerFunc)

	// Matches `run jspc/irc-build-bot main.yaml for v1.2.1`, `run jspc/irc-build-bot 1 for gh-pages` (and variations), building specified reference
	b.routing[regexp.MustCompile(`run\s+(\w+)\/(\w+)\s+(\S+)\s+for\s+(\S+)`)] = b.trigger

	// Matches `run jspc/irc-build-bot main.yaml`, `run jspc/irc-build-bot 1` (and variations), building main branch
	b.routing[regexp.MustCompile(`run\s+(\w+)\/(\w+)\s+(\S+)$`)] = b.trigger

	// Route messages
	b.client.Handlers.Add(girc.PRIVMSG, b.messageRouter)

	return
}

func (b Bot) messageRouter(c *girc.Client, e girc.Event) {
	var err error

	// skip messages older than a minute (assume it's the replayer)
	cutOff := time.Now().Add(0 - time.Minute)
	if e.Timestamp.Before(cutOff) {
		// ignore
		return
	}

	msg := []byte(e.Last())

	for r, f := range b.routing {
		if r.Match(msg) {
			err = f(r.FindAllSubmatch(msg, -1)[0])
			if err != nil {
				log.Printf("%v error: %s", f, err)
			}

			return
		}
	}

	// Ignore; not a message for us
}

func (b Bot) trigger(groups [][]byte) (err error) {
	if len(groups) < 4 && len(groups) > 5 {
		return fmt.Errorf("somehow ended up with %d groups, expected at least 3, at most 4", len(groups))
	}

	// groups[0] is the full string
	owner := string(groups[1])
	repo := string(groups[2])
	workflow := string(groups[3])

	var ref string
	if len(groups) == 5 {
		ref = string(groups[4])
	} else {
		ref = "main"
	}

	b.client.Cmd.Messagef(Chan, "Running workflow %q on %q/%q (ref: %q)", workflow, owner, repo, ref)

	// Use an intermediate error here to avoid accidentally returning
	// the wrong error
	if w, scErr := strconv.Atoi(workflow); scErr == nil {
		// Got a workflow ID, not a filename
		_, err = b.github.Actions.CreateWorkflowDispatchEventByID(context.Background(), owner, repo, int64(w), github.CreateWorkflowDispatchEventRequest{Ref: ref})

		return
	}

	_, err = b.github.Actions.CreateWorkflowDispatchEventByFileName(context.Background(), owner, repo, workflow, github.CreateWorkflowDispatchEventRequest{Ref: ref})

	if err != nil {
		b.client.Cmd.Messagef(Chan, "I got this error from github :/ %v", err)
	} else {
		b.client.Cmd.Message(Chan, "Job triggered (seemingly)")
	}

	return
}

func (b Bot) processNotifications() (err error) {
	log.Print("loading notifications")

	notifications, _, err := b.github.Activity.ListNotifications(context.Background(), &github.NotificationListOptions{All: false})
	if err != nil {
		return
	}

	b.github.Activity.MarkNotificationsRead(context.Background(), time.Now())

	log.Printf("loaded %d notifications", len(notifications))

	for _, n := range notifications {
		// We're only tracking CI activity rib.githubt now
		if *n.Reason != "ci_activity" {
			continue
		}

		b.client.Cmd.Messagef(Chan, "%s: %q (see: %s/actions)", *n.Repository.FullName, *n.Subject.Title, *n.Repository.HTMLURL)
	}

	return
}
