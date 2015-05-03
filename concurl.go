package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/hoisie/web"
)

const (
	cliName        = "concurl"
	cliDescription = "A small tool that fetches an url and concats the output to its own payload"
)

var (
	globalFlagset = flag.NewFlagSet(cliName, flag.ExitOnError)
	globalFlags   = struct {
		debug   bool
		version bool
		dep     string
		payload string
	}{}

	projectVersion = "dev"
	projectBuild   string
)

func init() {
	globalFlagset.BoolVar(&globalFlags.debug, "debug", false, "Print out more debug information to stderr")
	globalFlagset.BoolVar(&globalFlags.version, "version", false, "Print the version and exit")
	globalFlagset.StringVar(&globalFlags.dep, "dep", "", "Dependency to request data from")
	globalFlagset.StringVar(&globalFlags.payload, "payload", "", "Payload to be added to the payload")
}

func get(backend string) string {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/", backend), nil)
	if err != nil {
		fmt.Printf("req %s", err)
		os.Exit(1)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("res %s", err)
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("body %s", err)
		os.Exit(1)
	}

	return string(body)
}

func concurl(val string) string {
	payload := globalFlags.payload

	if globalFlags.dep != "" {
		backends := strings.Split(globalFlags.dep, ",")

		for _, backend := range backends {
			body := get(backend)
			fmt.Printf("%s", body)
			payload += body
		}
	}
	return payload
}

func main() {
	globalFlagset.Parse(os.Args[1:])

	// deal specially with --version
	if globalFlags.version {
		fmt.Println("concurl version", projectVersion, projectBuild)
		os.Exit(0)
	}

	web.Get("/(.*)", concurl)
	go web.Run("0.0.0.0:80")

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	// Stop the service gracefully.
	web.Close()
}
