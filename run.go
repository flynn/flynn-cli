package main

import (
	"errors"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/docopt/docopt-go"
	"github.com/flynn/flynn-controller/client"
	ct "github.com/flynn/flynn-controller/types"
	"github.com/flynn/go-flynn/demultiplex"
	"github.com/heroku/hk/term"
)

func runRun(argv []string, client *controller.Client) error {
	usage := `usage: flynn run [-d] [-r <release>] <command> [<argument>...]

Run a job.

Options:
   -d, --detached  run job without connecting io streams
   -r <release>    id of release to run (defaults to current app release)
`

	args, _ := docopt.Parse(usage, argv, true, "", false)

	runDetached := args["--detached"].(bool)
	var runRelease string
	if args["-r"] != nil {
		runRelease = args["-r"].(string)
	}

	if runRelease == "" {
		release, err := client.GetAppRelease(mustApp())
		if err == controller.ErrNotFound {
			return errors.New("No app release, specify a release with -release")
		}
		if err != nil {
			return err
		}
		runRelease = release.ID
	}
	req := &ct.NewJob{
		Cmd:       args["<argument>"].([]string),
		TTY:       term.IsTerminal(os.Stdin) && term.IsTerminal(os.Stdout) && !runDetached,
		ReleaseID: runRelease,
	}
	if req.TTY {
		cols, err := term.Cols()
		if err != nil {
			return err
		}
		lines, err := term.Lines()
		if err != nil {
			return err
		}
		req.Columns = cols
		req.Lines = lines
		req.Env = map[string]string{
			"COLUMNS": strconv.Itoa(cols),
			"LINES":   strconv.Itoa(lines),
			"TERM":    os.Getenv("TERM"),
		}
	}

	if runDetached {
		job, err := client.RunJobDetached(mustApp(), req)
		if err != nil {
			return err
		}
		log.Println(job.ID)
		return nil
	}

	rwc, err := client.RunJobAttached(mustApp(), req)
	if err != nil {
		return err
	}
	defer rwc.Close()

	if req.TTY {
		if err := term.MakeRaw(os.Stdin); err != nil {
			return err
		}
		defer term.Restore(os.Stdin)
	}

	go func() {
		io.Copy(rwc, os.Stdin)
		rwc.CloseWrite()
	}()
	if req.TTY {
		_, err = io.Copy(os.Stdout, rwc)
	} else {
		err = demultiplex.Copy(os.Stdout, os.Stderr, rwc)
	}
	// TODO: get exit code and use it
	return err
}
