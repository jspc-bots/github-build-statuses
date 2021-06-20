package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/go-github/v35/github"
	"github.com/jspc/bottom"
	"github.com/lrstanley/girc"
)

type Bot struct {
	bottom bottom.Bottom
	github *github.Client
}

type handlerFunc func(groups [][]byte) error

func New(user, password, server string, verify bool, gh *github.Client) (b Bot, err error) {
	b.github = gh
	b.bottom, err = bottom.New(user, password, server, verify)
	if err != nil {
		return
	}

	b.bottom.Client.Handlers.Add(girc.CONNECTED, func(c *girc.Client, e girc.Event) {
		c.Cmd.Join(Chan)
	})

	router := bottom.NewRouter()
	router.AddRoute(`run\s+([\w\-\_]+)\/([\w\-\_\.]+)\s+(\S+)\s+for\s+(\S+)`, b.trigger)
	router.AddRoute(`run\s+([\w\-\_]+)\/([\w\-\_\.]+)\s+(\S+)$`, b.trigger)

	b.bottom.Middlewares.Push(router)

	return
}

func (b Bot) trigger(sender, channel string, groups []string) (err error) {
	if len(groups) < 4 && len(groups) > 5 {
		return fmt.Errorf("somehow ended up with %d groups, expected at least 3, at most 4", len(groups))
	}

	// groups[0] is the full string
	owner := groups[1]
	repo := groups[2]
	workflow := groups[3]

	var ref string
	if len(groups) == 5 {
		ref = groups[4]
	} else {
		ref = "main"
	}

	b.bottom.Client.Cmd.Messagef(channel, "Running workflow %q on %q/%q (ref: %q)", workflow, owner, repo, ref)

	// Use an intermediate error here to avoid accidentally returning
	// the wrong error
	if w, scErr := strconv.Atoi(workflow); scErr == nil {
		// Got a workflow ID, not a filename
		_, err = b.github.Actions.CreateWorkflowDispatchEventByID(context.Background(), owner, repo, int64(w), github.CreateWorkflowDispatchEventRequest{Ref: ref})

		return
	}

	_, err = b.github.Actions.CreateWorkflowDispatchEventByFileName(context.Background(), owner, repo, workflow, github.CreateWorkflowDispatchEventRequest{Ref: ref})

	if err != nil {
		b.bottom.Client.Cmd.Messagef(channel, "I got this error from github :/ %v", err)
	} else {
		b.bottom.Client.Cmd.Message(channel, "Job triggered (seemingly)")
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

		b.bottom.Client.Cmd.Messagef(Chan, "%s: %q (see: %s/actions)", *n.Repository.FullName, *n.Subject.Title, *n.Repository.HTMLURL)
	}

	return
}
