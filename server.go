package main

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/flynn/flynn-controller/client"
)


var cmdServers = &Command{
	Run:      runServers,
	Usage:    "servers",
	Short:    "list servers",
	Long:     `List all servers in the ~/.flynnrc configuration file`,
	NoClient: true,
}

func runServers(cmd *Command, args []string, client *controller.Client) error {
	if len(args) != 0 {
		cmd.printUsage(true)
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


var cmdServerAdd = &Command{
	Run:      runServerAdd,
	Usage:    "server-add [-g <githost>] [-p <tlspin>] <server-name> <url> <key>",
	Short:    "add a server",
	Long:     `Command server-add adds a server to the ~/.flynnrc configuration file`,
	NoClient: true,
}

var serverGitHost string
var serverTLSPin string

func init() {
	cmdServerAdd.Flag.StringVarP(&serverGitHost, "git-host", "g", "", "git host (if host differs from api URL host)")
	cmdServerAdd.Flag.StringVarP(&serverTLSPin, "tls-pin", "p", "", "SHA256 of the server's TLS cert (useful if it is self-signed)")
}

func runServerAdd(cmd *Command, args []string, client *controller.Client) error {
	if len(args) != 3 {
		cmd.printUsage(true)
	}
	if err := readConfig(); err != nil {
		return err
	}

	s := &ServerConfig{
		Name:    args[0],
		URL:     args[1],
		Key:     args[2],
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

var cmdServerRemove = &Command{
	Run:      runServerRemove,
	Usage:    "server-remove <server-name>",
	Short:    "remove a server",
	Long:     `Command server-remove removes a server from the ~/.flynnrc configuration file`,
	NoClient: true,
}

func runServerRemove(cmd *Command, args []string, client *controller.Client) error {
	if len(args) != 1 {
		cmd.printUsage(true)
	}
	if err := readConfig(); err != nil {
		return err
	}

	name := args[0]

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
