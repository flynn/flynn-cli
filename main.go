package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"path/filepath"
	"runtime"
	"text/tabwriter"

	"github.com/flynn/go-docopt"
	"github.com/bgentry/pflag"
	"github.com/BurntSushi/toml"
	"github.com/flynn/flynn-controller/client"
)

type Command struct {
	// args does not include the command name
	Run  func(cmd *Command, args []string, client *controller.Client) error
	Flag pflag.FlagSet

	Usage string // first word is the command name
	Short string // `flynn help` output
	Long  string // `flynn help cmd` output

	NoClient bool
}

func (c *Command) printUsage(errExit bool) {
	if c.Runnable() {
		fmt.Printf("Usage: %s %s\n\n", os.Args[0], c.Usage)
	}
	fmt.Println(strings.Trim(c.Long, "\n"))
	if errExit {
		os.Exit(2)
	}
}

func (c *Command) Name() string {
	name := c.Usage
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func (c *Command) Runnable() bool {
	return c.Run != nil
}

func (c *Command) List() bool {
	return c.Short != ""
}

// Running `flynn help` will list commands in this order.
var commands = []*Command{
	//cmdServers,
	//cmdServerAdd,
	//cmdServerRemove,
	//cmdCreate,
	//cmdApps,
	//cmdPs,
	//cmdLog,
	//cmdScale,
	//cmdRun,
	//cmdEnv,
	//cmdEnvSet,
	//cmdEnvGet,
	//cmdEnvUnset,
	//cmdRoutes,
	//cmdRouteAddHTTP,
	//cmdRouteRemove,
	//cmdProviders,
	//cmdResourceAdd,
	//cmdKeys,
	//cmdKeyAdd,
	//cmdKeyRemove,
	//cmdReleaseAddDocker,
	//cmdVersion,
	//cmdHelp,
}

var (
	flagServer = os.Getenv("FLYNN_SERVER")
	flagApp    string
)

func main() {
	log.SetFlags(0)

	usage := `usage: flynn [-a <app>] <command> [<args>...]

Options:
   -a <app>
   -h, --help

Commands:
   help                show usage for a specific command
   cluster             manage clusters
   create              create an app
   apps                list apps
   ps                  list jobs
   kill                kill a job
   log                 get job log
   scale               change formation
   run                 run a job
   env                 manage env variables
   route               manage routes
   provider            manage resource providers
   resource            provision a new resource
   key                 manage SSH public keys
   release             add a docker image release
   version             show flynn version

See 'flynn help <command>' for more information on a specific command.
	`
	args, _ := docopt.Parse(usage, nil, true, Version, true)

	cmd := args.String["<command>"]
	cmdArgs := args.All["<args>"].([]string)

	if cmd == "help" {
		if len(cmdArgs) == 0 { // `flynn help`
			fmt.Println(usage)
			return
		} else { // `flynn help <command>`
			cmd = cmdArgs[0]
			cmdArgs = make([]string, 1)
			cmdArgs[0] = "--help"
		}
	}
	// Run the update command as early as possible to avoid the possibility of
	// installations being stranded without updates due to errors in other code
	if cmd == "update" {
		runUpdate(cmdArgs, nil)
		return
	} else if updater != nil {
		defer updater.backgroundRun() // doesn't run if os.Exit is called
	}

	flagApp = args.String["-a"]
	if flagApp != "" {
		if err := readConfig(); err != nil {
			log.Fatal(err)
		}

		if ra, err := appFromGitRemote(flagApp); err == nil {
			serverConf = ra.Server
			flagApp = ra.Name
		}
	}

	var client *controller.Client
	//if !cmd.NoClient {
		server, err := server()
		if err != nil {
			log.Fatal(err)
		}
		if server.TLSPin != "" {
			pin, err := base64.StdEncoding.DecodeString(server.TLSPin)
			if err != nil {
				log.Fatalln("error decoding tls pin:", err)
			}
			client, err = controller.NewClientWithPin(server.URL, server.Key, pin)
		} else {
			client, err = controller.NewClient(server.URL, server.Key)
		}
		if err != nil {
			log.Fatal(err)
		}
	//}

	if err := runCommand(cmd, cmdArgs, client); err != nil {
		log.Fatal(err)
		return
	}
}

func runCommand(cmd string, args []string, client *controller.Client) (err error) {
	argv := make([]string, 1)
	argv[0] = cmd
	argv = append(argv, args...)

	switch cmd {
	case "version":
		return runVersion(argv, client)
	case "create":
		return runCreate(argv, client)
	case "apps":
		return runApps(argv, client)
	case "ps":
		return runPs(argv, client)
	case "scale":
		return runScale(argv, client)
	case "run":
		return runRun(argv, client)
	case "log":
		return runLog(argv, client)
	case "env":
		return runEnv(argv, client)
	case "key":
		return runKey(argv, client)
	case "kill":
		return runKill(argv, client)
	case "cluster":
		return runCluster(argv, client)
	case "route":
		return runRoute(argv, client)
	case "resource":
		return runResource(argv, client)
	case "provider":
		return runProvider(argv, client)
	case "release":
		return runRelease(argv, client)
	}

	return fmt.Errorf("%s is not a flynn command. See 'flynn help'", cmd)
}

type Config struct {
	Servers []*ServerConfig `toml:"server"`
}

type ServerConfig struct {
	Name    string `json:"name"`
	GitHost string `json:"git_host"`
	URL     string `json:"url"`
	Key     string `json:"key"`
	TLSPin  string `json:"tls_pin"`
}

var config *Config
var serverConf *ServerConfig

func configPath() string {
	p := os.Getenv("FLYNNRC")
	if p == "" {
		p = filepath.Join(homedir(), ".flynnrc")
	}
	return p
}

func readConfig() error {
	if config != nil {
		return nil
	}
	conf := &Config{}
	_, err := toml.DecodeFile(configPath(), conf)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	config = conf
	return nil
}

func homedir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("%APPDATA%")
	}
	return os.Getenv("HOME")
}

var ErrNoServers = errors.New("no servers configured")

func server() (*ServerConfig, error) {
	if serverConf != nil {
		return serverConf, nil
	}
	if err := readConfig(); err != nil {
		return nil, err
	}
	if len(config.Servers) == 0 {
		return nil, ErrNoServers
	}
	if flagServer == "" {
		serverConf = config.Servers[0]
		return serverConf, nil
	}
	for _, s := range config.Servers {
		if s.Name == flagServer {
			serverConf = s
			return s, nil
		}
	}
	return nil, fmt.Errorf("unknown server %q", flagServer)
}

var appName string

func app() (string, error) {
	if flagApp != "" {
		return flagApp, nil
	}
	if app := os.Getenv("FLYNN_APP"); app != "" {
		flagApp = app
		return app, nil
	}
	if err := readConfig(); err != nil {
		return "", err
	}

	ra, err := appFromGitRemote(remoteFromGitConfig())
	if err != nil {
		return "", err
	}
	if ra == nil {
		return "", errors.New("no app found, run from a repo with a flynn remote or specify one with -a")
	}
	serverConf = ra.Server
	flagApp = ra.Name
	return ra.Name, nil
}

func mustApp() string {
	name, err := app()
	if err != nil {
		log.Fatal(err)
	}
	return name
}

func tabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 1, 2, 2, ' ', 0)
}

func listRec(w io.Writer, a ...interface{}) {
	for i, x := range a {
		fmt.Fprint(w, x)
		if i+1 < len(a) {
			w.Write([]byte{'\t'})
		} else {
			w.Write([]byte{'\n'})
		}
	}
}
