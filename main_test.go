package main

import (
	"log"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func AssertEqual(t *testing.T, act interface{}, exp interface{}) {
	if !reflect.DeepEqual(act, exp) {
		t.Fatalf("Assertion failed.\n\n  actual:   %v\n  expected: %v", act, exp)
	}
}

func AssertDateEqual(t *testing.T, act time.Time, exp time.Time) {
	diff := int64(math.Abs(float64(act.Sub(exp).Milliseconds())))
	if diff > 500 {
		t.Fatalf("Assertion failed.\n\n  actual:   %v\n  expected: %v", act, exp)
	}
}

func AssertContains(t *testing.T, act string, exp string) {
	if strings.Contains(act, exp) {
		t.Fatalf("Assertion failed because expected was not found within actual.\n\n  actual:   %v\n  expected: %v", act, exp)
	}
}

func TestNoArgs(t *testing.T) {
	err := Run([]string{"prog"})
	AssertEqual(t, err.Error(), "No configuration file with jobs to run. Usage: rept <path/to/job_config.toml>")
}

func TestTooManyArgs(t *testing.T) {
	err := Run([]string{"prog", "1", "2"})
	AssertEqual(t, err.Error(), "Arguments not understood. Usage: rept <path/to/job_config.toml>")
}

func TestConfigEmpty(t *testing.T) {
	err := Run([]string{"prog", "data/config_empty.toml"})
	AssertEqual(t, err.Error(), "The configuration has no jobs to run.")
}

func TestConfigWrongTimezone(t *testing.T) {
	err := Run([]string{"prog", "data/config_timezone.toml"})
	AssertEqual(t, err.Error(), "The configuration has no jobs to run.")
}

func TestConfigMisnamed(t *testing.T) {
	err := Run([]string{"prog", "data/config_misnamed.toml"})
	AssertEqual(t, err.Error(), "line 5: field corresponding to `Command' is not defined in main.Job")
}

func TestConfigCmdWrongType(t *testing.T) {
	err := Run([]string{"prog", "data/config_cmd_wrong_type.toml"})
	AssertEqual(t, err.Error(), "line 6: (main.Job.Cmd) cannot unmarshal TOML string into []string")
}

func TestConfigCmdMissing(t *testing.T) {
	err := Run([]string{"prog", "data/config_cmd_missing.toml"})
	AssertEqual(t, err.Error(), "Job 'Backup' has no 'Cmd' property but it is required.")
}

func TestConfigTimeMissing(t *testing.T) {
	err := Run([]string{"prog", "data/config_time_missing.toml"})
	AssertEqual(t, err.Error(), "Job 'Backup' has no 'Every' property but it is required.")
}

func TestConfig(t *testing.T) {
	config, err := ParseConfig("data/config_log.toml")
	AssertEqual(t, err, nil)
	AssertEqual(t, config, Config{
		Options: Options{
			LogPath: "/tmp/test.log",
		},
		Jobs: map[string]Job{
			"BackupDaily": Job{
				Cmd:   []string{"ls", "-lha"},
				Every: "Day",
				At:    "2:15",
			},
			"BackupWeekly": Job{
				Cmd:   []string{"ls", "-lh"},
				Every: "Tuesday",
				At:    "2:00",
			},
		},
	})
}

func TestLogFile(t *testing.T) {
	err := Run([]string{"prog", "data/config.toml"})
	AssertEqual(t, err.Error(), "mkdir /var/log/rept: permission denied")
}

func TestLogFileOptions(t *testing.T) {
	err := Run([]string{"prog", "data/config_log.toml"})
	AssertEqual(t, err, nil)

	_, err = os.Stat("/tmp/test.log")
	if os.IsNotExist(err) {
		t.Fatal("File does not exist")
	}
}

func TestParseTimeErrors(t *testing.T) {
	_, err := ParseTime("JOB_NAME", "daily", "__default__", 0, time.Local)
	AssertEqual(
		t,
		err.Error(),
		"Job 'JOB_NAME' has invalid value for 'Every': 'daily' "+ValidEvery,
	)

	_, err = ParseTime("JOB_NAME", " second asd  ", "__default__", 0, time.Local)
	AssertEqual(
		t,
		err.Error(),
		"Job 'JOB_NAME' has invalid value for 'Every': 'second asd' "+ValidEvery,
	)

	_, err = ParseTime("JOB_NAME", "minute", "15", 0, time.Local)
	AssertEqual(
		t,
		err.Error(),
		"Job 'JOB_NAME' has invalid use of 'At': 'Every' is set to 'minute'",
	)

	_, err = ParseTime("JOB_NAME", "hour", "15", 0, time.Local)
	AssertEqual(
		t,
		err.Error(),
		"Job 'JOB_NAME' has invalid use of 'At': 'Every' is set to 'hour'",
	)

	_, err = ParseTime("JOB_NAME", "second", "15", 0, time.Local)
	AssertEqual(
		t,
		err.Error(),
		"Job 'JOB_NAME' has invalid use of 'At': 'Every' is set to 'second'",
	)
}

func TestParseTime(t *testing.T) {
	now := time.Now()
	y := now.Year()
	m := now.Month()
	d := now.Day()
	H := now.Hour()
	M := now.Minute()
	S := now.Second()
	wd := int(now.Weekday())
	L := time.Local

	next := RunParseTime(t, "15 second", "__default__", 0)
	AssertDateEqual(t, next, time.Date(y, m, d, H, M, S-S%15+15, 0, L))

	next = RunParseTime(t, "5 Minutes", "__default__", 0)
	AssertDateEqual(t, next, time.Date(y, m, d, H, M-M%5+5, 0, 0, L))

	next = RunParseTime(t, "Hour", "__default__", 0)
	AssertDateEqual(t, next, time.Date(y, m, d, H+1, 0, 0, 0, L))

	next = RunParseTime(t, "Day", "2:00", 0)
	if H >= 2 {
		AssertDateEqual(t, next, time.Date(y, m, d+1, 2, 0, 0, 0, L))
	} else {
		AssertDateEqual(t, next, time.Date(y, m, d, 2, 0, 0, 0, L))
	}

	next = RunParseTime(t, "Tuesday", "2:15", 0)
	if wd < 2 {
		AssertDateEqual(t, next, time.Date(y, m, d+2, 2, 15, 0, 0, L))
	} else if wd == 2 && H < 2 || M < 15 {
		AssertDateEqual(t, next, time.Date(y, m, d, 2, 15, 0, 0, L))
	} else {
		AssertDateEqual(t, next, time.Date(y, m, d+(9-wd), 2, 15, 0, 0, L))
	}

	next = RunParseTime(t, "Week", "0", 0)
	AssertDateEqual(t, next, time.Date(y, m, d+(7-wd), 0, 0, 0, 0, L))

	next = RunParseTime(t, "Month", "0", 5)
	AssertDateEqual(t, next, time.Date(y, m+1, 5, 0, 0, 0, 0, L))
}

func RunParseTime(t *testing.T, every string, at string, dayOfMonth int) time.Time {
	scheduler, err := ParseTime("JOB_NAME", every, at, dayOfMonth, time.Local)
	AssertEqual(t, err, nil)

	scheduler.StartAsync()
	_, next := scheduler.NextRun()
	scheduler.Stop()

	return next
}

func TestCronRun(t *testing.T) {
	err := Run([]string{"prog", "data/config_run.toml"})
	AssertEqual(t, err, nil)

	var buf strings.Builder
	log.SetOutput(&buf)
	AssertEqual(t, buf.String(), "")

	time.Sleep(2 * time.Second)
	AssertContains(t, buf.String(), "Executing job Test by running command [ls -lha]")
	buf.Reset()

	time.Sleep(2 * time.Second)
	AssertContains(t, buf.String(), "Executing job Test by running command [ls -lha]")
	buf.Reset()
}
