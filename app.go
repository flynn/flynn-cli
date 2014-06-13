package main

import (
	"log"
	"os/exec"

	"github.com/docopt/docopt-go"
	"github.com/flynn/flynn-controller/client"
	ct "github.com/flynn/flynn-controller/types"
)

var cmdCreate = &Command{
	Run:   runCreate,
	Usage: "create [<name>]",
	Short: "create an app",
	Long:  `Create an application in Flynn`,
}

func runCreate(argv []string, client *controller.Client) error {
	usage := `usage: flynn create [<name>]

List flynn apps.
	`
	docopt.Parse(usage, argv, true, "", false)

	if len(args) > 1 {
		cmd.printUsage(true)
	}

	app := &ct.App{}
	if len(args) > 0 {
		app.Name = args[0]
	}

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