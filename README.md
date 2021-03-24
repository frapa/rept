# Rept

A utility to run jobs periodically, like the classic `cron` but
with better logging and a little more human friendly.

It's written in Go, so it's zero dependency and the executable can
just be dropped and run. I also provide docker images if you want
to use it in a container.

## Features

- Simple configuration.
- Logs everything by default, including your job standard output.
- Easy to run standalone for testing your configuration.
- Reports configuration errors clearly.
- Reports planned execution times for easy debugging.
- Automatically rotate logs & delete old ones.
- Zero dependency.
- Docker-friendly. Easily containerize your periodic jobs.
- Very low feature bloat (e.g. simple and dumb).

## Quick Start

Install latest release with:

```bash
curl -o /usr/local/bin/rept -L https://github.com/frapa/rept/releases/latest/download/rept
```

Then run it with:

```bash
rept <path/to/config.toml>
```

If you want to run this as a systemd service (which means `rept` will be run
automatically at boot and restarted if it crashes), you can run the following
commands:

```bash
curl -o /etc/systemd/system/rept.service -L https://raw.githubusercontent.com/frapa/rept/main/rept.service
systemctl daemon-reload
systemctl enable rept.service
systemctl start rept.service
```

The service file above will automatically read the configuration from
`/etc/rept/rept.toml`. After changing the configuration file,
you have to restart the service with `systemctl restart rept.service`
for the changed to take effect.

For those in a hurry a simple template with two jobs:

```toml
[Options]
LogPath = "/path/to/dump.log"

[Jobs.BackupDaily]
Cmd = ["bash", "backup.sh", "daily"]
Every = "Day"
At = "2:15"

[Jobs.BackupWeekly]
Cmd = ["bash", "backup.sh", "weekly"]
Every = "Tuesday"
At = "2:00"
```

## Complete Quick Reference

This shows all options and possible values:

```toml
[Options]
# If not set, defaults to '/var/log/rept/rept.log'.
# The folder must be writable by the user running `rept`.
LogPath = "/tmp/test.log"
# This is the minimum number of days to keep the logs for.
KeepLogsDays = 7
# Either 'Local' or 'UTC', by default 'Local'
Timezone = "Local"

# Jobs is a map of job names to specifications.
[Jobs.MyJobName]
# Cmd is required and must be a command with a list
# of command line arguments. If you want to run
# multiple commands, you need to wrap them in a script.
Cmd = ["ls", "-lha"]
# This defines how often the job should be repeated.
# Valid values are 'second', 'minute', 'hour', 'day',
# 'week', 'month' and 'monday' to 'sunday'.
# These can be preceded by an integer to decrease the
# frequency. Plural forms are accepted.
Every = "2 Months"
# When `Every = "Month"`, you can set the day of the
# month to run the job, starting from 1.
# If negative, it will go back to the previous month,
# so you can use -5 to run the job on the fifth-last
# day of the preceding month.
DayOfMonth = 3
# The time to run the job at. Invalid when
# `Every` is 'second', 'minute', 'hour'.
At = "2:15"
# Override timezone only for this job
Timezone = "Local"

# Concluding, the job spec above will run the job every
# two months, on the third day of the month at 2:15
# in your server local time.
```

## Usage With Docker

I provide two docker images with `rept` pre-installed:

- `frapasa/rept:{version}` - A plain image with nothing
  but the `rept` executable. Useful as base image if you
  just need to periodically run a static executable.
- `frapasa/rept:{version}-buster` - A more useful,
  debian-based image that contain all the usual goodies.

For both images, the configuration will be read from
`/etc/rept/rept.toml`.

To learn more, please visit [Docker Hub](https://hub.docker.com/repository/docker/frapasa/rept).

## Motivation

My main complaint with cron it's that it's not transparent on whether
the jobs are properly configured and running and it's hard to debug.
Some examples:

- It's hard to say if the configuration is right. The only feedback is
  a log message buried in a syslog and it does not even say what's wrong
  with your config.
- It has a bunch of quirks, for instance the config files must end with
  a newline otherwise the configs are ignored.
- It's not possible to check when it's the next time the job will be run
  and thus prove if your configuration is right.
- When jobs do not run, there's no logging of the output by default, you
  either have to do it manually and then rotate logs, or log to syslog.

I do not find the cron syntax difficult to understand, but I have to look
it up every time. I think a bit more verbose syntax helps.

## Limitations

Currently, it's not easy or even possible in some cases to configure
`rept` to run at some intervals. This is in part intentional, as I want
to keep `rept` simple and dumb, but if you see something obviously missing
please open an issue.

Another design choice to keep it simple is having a single configuration file.
