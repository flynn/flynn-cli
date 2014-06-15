package main

import (
	"io"
	"os"

	"github.com/flynn/go-docopt"
	"github.com/flynn/flynn-controller/client"
	"github.com/flynn/go-flynn/demultiplex"
)

func runLog(argv []string, client *controller.Client) error {
	usage := `usage: flynn log [options] <job>

Stream log for a specific job.

Options:
    -s, --split-stderr    send stderr lines to stderr
	`
	args, _ := docopt.Parse(usage, argv, true, "", false)

	rc, err := client.GetJobLog(mustApp(), args.String["<job>"])
	if err != nil {
		return err
	}
	var stderr io.Writer
	if args.Bool["--split-stderr"] {
		stderr = os.Stderr
	}
	demultiplex.Copy(os.Stdout, stderr, rc)
	rc.Close()
	return nil
}
