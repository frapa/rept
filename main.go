package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/naoina/toml"
)

const (
	ValidEvery = "(valid values are 'second', 'minute', 'hour', 'day', 'week', " +
		"'month' and 'monday' to 'sunday', optionally preceded by an integer)"
)

func main() {
	err := Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

func Run(args []string) error {
	if len(args) == 1 {
		return errors.New("No configuration file with jobs to run. Usage: rept <path/to/job_config.toml>")
	} else if len(args) > 2 {
		return errors.New("Arguments not understood. Usage: rept <path/to/job_config.toml>")
	}

	config, err := ParseConfig(args[1])
	if err != nil {
		return err
	}

	if len(config.Jobs) == 0 {
		return errors.New("The configuration has no jobs to run.")
	}

	err = SetupOptions(config)
	if err != nil {
		return err
	}

	err = SetupJobs(config)
	if err != nil {
		return err
	}

	return nil
}

type Job struct {
	Cmd        []string
	Every      string
	At         string
	DayOfMonth int
	Timezone   string
}

type Options struct {
	LogPath      string
	KeepLogsDays uint
	Timezone     string
}

type Config struct {
	Options Options
	Jobs    map[string]Job
}

func ParseConfig(configPath string) (Config, error) {
	document, err := os.Open(configPath)
	if err != nil {
		return Config{}, err
	}

	config := Config{}
	err = toml.NewDecoder(document).Decode(&config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func SetupJobs(config Config) error {
	for jobName, job := range config.Jobs {
		if len(job.Cmd) == 0 {
			msg := "Job '%v' has no 'Cmd' property but it is required."
			return errors.New(fmt.Sprintf(msg, jobName))
		}

		if job.Every == "" {
			msg := "Job '%v' has no 'Every' property but it is required."
			return errors.New(fmt.Sprintf(msg, jobName))
		}

		if job.Timezone == "" {
			job.Timezone = config.Options.Timezone
		}

		err := Schedule(jobName, job)
		if err != nil {
			return err
		}
	}

	return nil
}

func Schedule(jobName string, job Job) error {
	at := job.At
	if at == "" {
		at = "__default__"
	}

	dayOfMonth := job.DayOfMonth
	if dayOfMonth == 0 {
		dayOfMonth = 1
	}

	timezone := time.Local
	if strings.ToLower(job.Timezone) == "utc" {
		timezone = time.UTC
	}

	scheduler, err := ParseTime(
		jobName,
		job.Every,
		at,
		dayOfMonth,
		timezone,
	)
	if err != nil {
		return err
	}

	cronJob, err := scheduler.Do(Execute, jobName, job.Cmd)
	if err != nil {
		return err
	}

	scheduler.StartAsync()

	log.Printf(
		"Scheduled job '%v', next run at %v",
		jobName, cronJob.NextRun().Round(time.Second),
	)

	return nil
}

func ParseTime(
	jobName string,
	every string,
	at string,
	dayOfTheMonth int,
	timezone *time.Location,
) (*gocron.Scheduler, error) {
	every = NormString(every)
	at = NormString(at)

	value := time.Duration(1)
	if strings.Contains(every, " ") {
		parts := strings.SplitN(every, " ", 2)
		valueStr := parts[0]
		originalEvery := every
		every = parts[1]

		valueInt, err := strconv.Atoi(valueStr)
		if err != nil {
			return nil, InvalidEveryError(jobName, originalEvery)
		}
		value = time.Duration(valueInt)
	}

	now := time.Now()
	y := now.Year()
	m := now.Month()
	d := now.Day()

	startAt := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	scheduler := gocron.NewScheduler(timezone)
	switch every {
	case "second", "seconds":
		scheduler = scheduler.Every(value * time.Second).StartAt(startAt)

		if at != "__default__" {
			return nil, InvalidAtUseError(jobName, every)
		}
	case "minute", "minutes":
		scheduler = scheduler.Every(value * time.Minute).StartAt(startAt)

		if at != "__default__" {
			return nil, InvalidAtUseError(jobName, every)
		}
	case "hour", "hours":
		scheduler = scheduler.Every(value * time.Hour).StartAt(startAt)

		if at != "__default__" {
			return nil, InvalidAtUseError(jobName, every)
		}
	case "day", "days":
		scheduler = scheduler.Every(int(value)).Days()
	case "month", "months":
		scheduler = scheduler.Every(int(value)).Months(dayOfTheMonth)
	case "monday", "mondays":
		scheduler = scheduler.Every(int(value)).Weeks().Monday()
	case "tuesday", "tuesdays":
		scheduler = scheduler.Every(int(value)).Weeks().Tuesday()
	case "wednesday", "wednesdays":
		scheduler = scheduler.Every(int(value)).Weeks().Wednesday()
	case "thursday", "thursdays":
		scheduler = scheduler.Every(int(value)).Weeks().Thursday()
	case "friday", "fridays":
		scheduler = scheduler.Every(int(value)).Weeks().Friday()
	case "saturday", "saturdays":
		scheduler = scheduler.Every(int(value)).Weeks().Saturday()
	case "week", "weeks", "sunday", "sundays":
		scheduler = scheduler.Every(int(value)).Weeks().Sunday()
	default:
		return nil, InvalidEveryError(jobName, every)
	}

	if at != "__default__" {
		scheduler = scheduler.At(at)
	}

	return scheduler, nil
}

func InvalidAtUseError(jobName string, every string) error {
	return errors.New(
		fmt.Sprintf(
			"Job '%v' has invalid use of 'At': 'Every' is set to '%v'",
			jobName, every,
		),
	)
}

func InvalidEveryError(jobName string, every string) error {
	return errors.New(
		fmt.Sprintf(
			"Job '%v' has invalid value for 'Every': '%v' "+ValidEvery,
			jobName, every,
		),
	)
}

func NormString(time string) string {
	time = strings.ToLower(time)
	time = strings.TrimSpace(time)
	time = regexp.MustCompile(`\s+`).ReplaceAllString(time, " ")
	return time
}

func Execute(jobName string, cmd []string) {
	log.Printf("Executing job '%v'. Running command %v\n", jobName, cmd)

	proc := exec.Command(cmd[0], cmd[1:]...)
	output, err := proc.Output()
	if err != nil {
		log.Printf("ERROR in job '%v':\n%v\n", jobName, err)
		return
	}

	log.Printf("Job '%v' succeeded with output:\n%v\n", jobName, string(output))
}

func SetupOptions(config Config) error {
	err := SetupLogger(config.Options.LogPath, config.Options.KeepLogsDays)
	if err != nil {
		return err
	}

	return nil
}

func CheckTimezone(timezone string) error {
	tz := strings.ToLower(timezone)
	if tz != "local" && tz != "utc" {
		return errors.New(
			fmt.Sprintf(
				"'Timezone' cannot be set to '%v' (valid values are 'Local', 'UTC')",
				timezone,
			),
		)
	}

	return nil
}

func SetupLogger(logPath string, keepFor uint) error {
	if logPath == "" {
		logPath = "/var/log/rept/rept.log"
	}

	if keepFor == 0 {
		keepFor = 7
	}

	err := os.MkdirAll(filepath.Dir(logPath), 0755)
	if err != nil {
		return err
	}

	logfile, err := rotatelogs.New(
		logPath+".%Y-%m-%d",
		rotatelogs.WithLinkName(logPath), // Path of the symlink to the current log
		rotatelogs.WithMaxAge(-1),        // Do not delete with max age
		rotatelogs.WithRotationSize(2),   // Keep at most 2 logs
		rotatelogs.WithRotationTime(time.Duration(keepFor)*24*time.Hour), // Rotate logs each week
	)
	if err != nil {
		return err
	}

	writer := io.MultiWriter(os.Stdout, logfile)
	log.SetOutput(writer)

	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("- ")

	return nil
}
