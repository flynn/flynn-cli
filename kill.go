package main

import (
  "log"

  "github.com/flynn/flynn-controller/client"
)

var cmdKill = &Command{
  Run:   runKill,
  Usage: "kill <job>",
  Short: "kill a job",
  Long:  `Kill a job`,
}

func runKill(cmd *Command, args []string, client *controller.Client) error {
  if len(args) == 0 {
    cmd.printUsage(true)
  }

  job := args[0]

  if err := client.DeleteJob(mustApp(), job); err != nil {
    return err
  }
  log.Printf("Key %s removed.", job)
  return nil
}
