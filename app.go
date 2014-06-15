package main

import (
	"log"
	"os/exec"

	"github.com/flynn/go-docopt"
	"github.com/flynn/flynn-controller/client"
	ct "github.com/flynn/flynn-controller/types"
)

func runCreate(argv []string, client *controller.Client) error {
	usage := `usage: flynn create [<name>]

Create an application in Flynn.
	`
	args, _ := docopt.Parse(usage, argv, true, "", false)

	app := &ct.App{}
	app.Name = args.String["<name>"]

	if err := client.CreateApp(app); err != nil {
		return err
	}

	exec.Command("git", "remote", "add", "flynn", gitURLPre(serverConf.GitHost)+app.Name+gitURLSuf).Run()
	log.Printf("Created %s", app.Name)
	return nil
}

func runApps(argv []string, client *controller.Client) error {
	usage := `usage: flynn apps

List flynn apps.
	`
	docopt.Parse(usage, argv, true, "", false)

	apps, err := client.AppList()
	if err != nil {
		return err
	}

	w := tabWriter()
	defer w.Flush()

	listRec(w, "ID", "NAME")
	for _, a := range apps {
		listRec(w, a.ID, a.Name)
	}
	return nil
}