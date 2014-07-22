package main

import (
	"github.com/flynn/flynn-controller/client"
	"github.com/flynn/go-docopt"
	u "github.com/flynn/go-flynn/updater"
)

func runUpdate(argv []string, client *controller.Client) error {
	usage := `usage: flynn update

Update Flynn components.
	`
	_, _ = docopt.Parse(usage, argv, true, "", false)

	updater := &u.Updater{Client: client}
	if err := updater.Update(); err != nil {
		return err
	}

	return nil
}
