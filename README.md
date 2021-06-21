# Github Build Status Bot

Retrieve updates from github actions, and write them to a channel somewhere. Additionally, re-run jobs and trigger workflows too.

This bot makes a couple of assumptions:

1. You've a SASL account for this bot to use
2. You've enabled actions notifications in github for failed/successful runs (these notifications need to be web notifications)

This bot requires the following env vars:

* `$GITHUB_TOKEN` - a github token with notifications and actions enabled
* `$SASL_USER` - the user to connect with
* `$SASL_PASSWORD` - the password to connect with
* `$SERVER` - IRC connection details, as `irc://server:6667` or `ircs://server:6697` (`ircs` implies irc-over-tls)
* `$VERIFY_TLS` - Verify TLS, or sack it off. This is of interest to people, like me, running an ircd on localhost with a self-signed cert. Matches "true" as true, and anything else as false
* `$POLL_PERIOD` - Time, in seconds, between polls of the github notification endpoint

The SASL mechanism is hardcoded to PLAIN. It's not a big job to change, I just don't know how to test it.

## Building

This bot can be built using pretty standard go tools:

```bash
$ go build
```

Or via docker:

```bash
$ docker build -t foo .
```

## Running

If you've built the app yourself, then happy day- there's your binary!

Otherwise I suggest via docker:

```bash
$ docker run jspc/build-bot
```

(Setting the above environment variables accordingly)
