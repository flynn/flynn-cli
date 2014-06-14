package main

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/BurntSushi/toml"
	"github.com/flynn/flynn-controller/client"
)

func runCluster(argv []string, client *controller.Client) error {
	usage := `usage: flynn cluster
       flynn cluster add [-g <githost>] [-p <tlspin>] <cluster-name> <url> <key>
       flynn cluster remove <cluster-name>

Manage clusters in the ~/.flynnrc configuration file.

Options:
   -g, --git-host <githost>  git host (if host differs from api URL host)
   -p, --tls-pin <tlspin>    SHA256 of the server's TLS cert (useful if it is self-signed)

Commands:
   With no arguments, shows a list of clusters.

   add     adds a cluster to the ~/.flynnrc configuration file
   remove  removes a cluster from the ~/.flynnrc configuration file
`
	args, _ := docopt.Parse(usage, argv, true, "", false)

	if args["add"] == true {
		return runClusterAdd(args, client)
	} else if args["remove"] == true {
		return runClusterRemove(args, client)
	}

	if err := readConfig(); err != nil {
		return err
	}

	w := tabWriter()
	defer w.Flush()

	listRec(w, "NAME", "URL")
	for _, s := range config.Servers {
		listRec(w, s.Name, s.URL)
	}
	return nil
}

func runClusterAdd(args map[string]interface{}, client *controller.Client) error {
	if err := readConfig(); err != nil {
		return err
	}

	var serverGitHost string;
	if args["--git-host"] != nil {
		serverGitHost = args["--git-host"].(string)
	}
	var serverTLSPin string;
	if args["--tls-pin"] != nil {
		serverGitHost = args["--tls-pin"].(string)
	}

	s := &ServerConfig{
		Name:    args["<cluster-name>"].(string),
		URL:     args["<url>"].(string),
		Key:     args["<key>"].(string),
		GitHost: serverGitHost,
		TLSPin:  serverTLSPin,
	}
	if serverGitHost == "" {
		u, err := url.Parse(s.URL)
		if err != nil {
			return err
		}
		if host, _, err := net.SplitHostPort(u.Host); err == nil {
			s.GitHost = host
		} else {
			s.GitHost = u.Host
		}
	}

	for _, existing := range config.Servers {
		if existing.Name == s.Name {
			return fmt.Errorf("Server %q already exists in ~/.flynnrc", s.Name)
		}
		if existing.URL == s.URL {
			return fmt.Errorf("A server with the URL %q already exists in ~/.flynnrc", s.URL)
		}
		if existing.GitHost == s.GitHost {
			return fmt.Errorf("A server with the git host %q already exists in ~/.flynnrc", s.GitHost)
		}
	}

	config.Servers = append(config.Servers, s)

	f, err := os.Create(configPath())
	if err != nil {
		return err
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(config); err != nil {
		return err
	}

	log.Printf("Server %q added.", s.Name)
	return nil
}

func runClusterRemove(args map[string]interface{}, client *controller.Client) error {
	if err := readConfig(); err != nil {
		return err
	}

	name := args["<cluster-name>"].(string)

	for i, s := range config.Servers {
		if s.Name == name {
			config.Servers = append(config.Servers[:i], config.Servers[i+1:]...)

			f, err := os.Create(configPath())
			if err != nil {
				return err
			}
			defer f.Close()

			if len(config.Servers) != 0 {
				if err := toml.NewEncoder(f).Encode(config); err != nil {
					return err
				}
			}

			log.Printf("Server %q removed.", s.Name)
			return nil
		}
	}
	return nil
}