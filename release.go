package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/flynn/go-docopt"
	"github.com/flynn/flynn-controller/client"
	ct "github.com/flynn/flynn-controller/types"
)

func runRelease(argv []string, client *controller.Client) error {
	usage := `usage: flynn release add [-t <type>] <image> <tag>

Manage app releases.

Options:
   -t <type>  type of the release. Currently only 'docker' is supported. [default: docker]
Commands:
   add   add a new release
	`
	args, _ := docopt.Parse(usage, argv, true, "", false)

	if args.Bool["add"] {
		if args.String["-t"] == "docker" {
			return runReleaseAddDocker(args, client)
		} else {
			return fmt.Errorf("Release type %s not supported.", args.String["-t"])
		}
	}

	log.Fatal("Toplevel command not implemented.")
	return nil
}

func runReleaseAddDocker(args *docopt.Args, client *controller.Client) error {
	image := args.String["<image>"]
	tag := args.String["<tag>"]

	if !strings.Contains(image, ".") {
		image = "/" + image
	}

	artifact := &ct.Artifact{
		Type: "docker",
		URI:  fmt.Sprintf("docker://%s?tag=%s", image, tag),
	}
	if err := client.CreateArtifact(artifact); err != nil {
		return err
	}

	release := &ct.Release{ArtifactID: artifact.ID}
	if err := client.CreateRelease(release); err != nil {
		return err
	}

	if err := client.SetAppRelease(mustApp(), release.ID); err != nil {
		return err
	}

	log.Printf("Created release %s.", release.ID)

	return nil
}
