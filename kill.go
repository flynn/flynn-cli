package main

import (
	"log"

	"github.com/docopt/docopt-go"
	"github.com/flynn/flynn-controller/client"
)

func runKill(argv []string, client *controller.Client) error {
	usage := `usage: flynn kill <job>

Kill a job.`
	args, _ := docopt.Parse(usage, argv, true, "", false)
	job := args["<job>"].(string)

	if err := client.DeleteJob(mustApp(), job); err != nil {
		return err
	}
	log.Printf("Job %s killed.", job)
	return nil
}
