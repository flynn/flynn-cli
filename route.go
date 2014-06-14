package main

import (
	"bytes"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/docopt/docopt-go"
	"github.com/flynn/flynn-controller/client"
	"github.com/flynn/strowger/types"
)

func runRoute(argv []string, client *controller.Client) error {
	usage := `usage: flynn route
       flynn route add [-t <type>] [-s <service>] [-c <tls-cert> -k <tls-key>] <domain>
       flynn route remove <id>

Manage routes for application.

Options:
   -t <type>                  route's type (Currently only http supported) [default: http]
   -s, --service <service>    service name to route domain to (defaults to APPNAME-web)
   -c, --tls-cert <tls-cert>  path to PEM encoded certificate for TLS, - for stdin
   -k, --tls-key <tls-key>    path to PEM encoded private key for TLS, - for stdin

Commands:
   With no arguments, shows a list of routes.

   add     adds a route to an app
   remove  removes a route
`
	args, _ := docopt.Parse(usage, argv, true, "", false)

	if args["add"] == true {
		if args["-t"].(string) == "http" {
			return runRouteAddHTTP(args, client)
		} else {
			return fmt.Errorf("Route type %s not supported.", args["-t"])
		}
	} else if args["remove"] == true {
		return runRouteRemove(args, client)
	}

	routes, err := client.RouteList(mustApp())
	if err != nil {
		return err
	}

	w := tabWriter()
	defer w.Flush()

	var route, protocol, service string
	listRec(w, "ROUTE", "SERVICE", "ID")
	for _, k := range routes {
		switch k.Type {
		case "tcp":
			protocol = "tcp"
			route = strconv.Itoa(k.TCPRoute().Port)
			service = k.TCPRoute().Service
		case "http":
			route = k.HTTPRoute().Domain
			service = k.TCPRoute().Service
			if k.HTTPRoute().TLSCert == "" {
				protocol = "http"
			} else {
				protocol = "https"
			}
		}
		listRec(w, protocol+":"+route, service, k.ID)
	}
	return nil
}

func runRouteAddHTTP(args map[string]interface{}, client *controller.Client) error {
	var tlsCert []byte
	var tlsKey []byte

	var routeHTTPService string
	if args["--service"] == nil {
		routeHTTPService = mustApp() + "-web"
	} else {
		routeHTTPService = args["--service"].(string)
	}

	if args["tls-cert"] != nil && args["tls-key"] != nil {
		var stdin []byte
		var err error

		tlsCertPath := args["tls-cert"].(string)
		tlsKeyPath := args["tls-key"].(string)

		if tlsCertPath == "-" || tlsKeyPath == "-" {
			stdin, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("Failed to read from stdin: %s", err)
			}
		}

		tlsCert, err = readPEM("CERTIFICATE", tlsCertPath, stdin)
		if err != nil {
			return errors.New("Failed to read TLS Cert")
		}
		tlsKey, err = readPEM("PRIVATE KEY", tlsKeyPath, stdin)
		if err != nil {
			return errors.New("Failed to read TLS Key")
		}
	} else if args["tls-cert"] != nil || args["tls-key"] != nil  {
		return errors.New("Both the TLS certificate AND private key need to be specified")
	}

	hr := &strowger.HTTPRoute{
		Service: routeHTTPService,
		Domain:  args["<domain>"].(string),
		TLSCert: string(tlsCert),
		TLSKey:  string(tlsKey),
	}
	route := hr.ToRoute()
	if err := client.CreateRoute(mustApp(), route); err != nil {
		return err
	}
	fmt.Println(route.ID)
	return nil
}

func readPEM(typ string, path string, stdin []byte) ([]byte, error) {
	if path == "-" {
		var buf bytes.Buffer
		var block *pem.Block
		for {
			block, stdin = pem.Decode(stdin)
			if block == nil {
				break
			}
			if block.Type == typ {
				pem.Encode(&buf, block)
			}
		}
		if buf.Len() > 0 {
			return buf.Bytes(), nil
		}
		return nil, errors.New("No PEM blocks found in stdin")
	}
	return ioutil.ReadFile(path)
}

func runRouteRemove(args map[string]interface{}, client *controller.Client) error {
	routeID := args["<id>"].(string)

	if err := client.DeleteRoute(mustApp(), routeID); err != nil {
		return err
	}
	fmt.Printf("Route %s removed.\n", routeID)
	return nil
}
