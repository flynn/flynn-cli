package main

import (
	"fmt"

	"github.com/docopt/docopt-go"
	"github.com/flynn/flynn-controller/client"
)

// NoClient
func runVersion(argv []string, client *controller.Client) error {
	usage := `usage: flynn version

Show flynn version string.
	`
	docopt.Parse(usage, argv, true, "", false)

	fmt.Println(Version)
	return nil
}
