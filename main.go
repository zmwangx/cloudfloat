package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/gofrs/flock"
	"github.com/mattn/go-isatty"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
)

const _progName = "cloudfloat"

func init() {
	log.SetLevel(log.InfoLevel)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `Usage:
  %s [options] <config_file>

The config file is required. Use -dump-config-template to generate a
documented config file template.

Cloudflare credentials are required and are passed in through environment
variables. You may specify one of the following:

- CF_API_TOKEN (must have the permissions to set DNS records for the
  appropriate zones);
- CF_API_KEY and CF_API_EMAIL.

Options:
`, os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	// Command line parsing.
	var dumpConfigTemplate bool
	flag.BoolVar(&dumpConfigTemplate, "dump-config-template", false, "")
	flag.Parse()

	if dumpConfigTemplate {
		os.Stdout.WriteString(_configTemplate)
		log.Exit(0)
	}

	if len(flag.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "wrong number of arguments")
		flag.Usage()
		log.Exit(1)
	}
	configFile := flag.Arg(0)

	// Make sure there are no competing instances running.
	lockPath := filepath.Join(os.TempDir(), _progName+".lock")
	lock := flock.New(lockPath)
	ok, err := lock.TryLock()
	if err != nil {
		log.Fatalf("failed to acquire lock on %s: %s", lockPath, err)
	}
	if !ok {
		log.Fatalf("another instance of %s already running", _progName)
	}
	log.DeferExitHandler(func() {
		if err = lock.Unlock(); err != nil {
			log.Fatalf("failed to unlock %s: %s", lockPath, err)
		}
	})

	// Load config and configure logging.
	conf, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("failed to load and parse config: %s", err)
	}

	if conf.Logging.Logfile != "" {
		logfile, err := os.OpenFile(conf.Logging.Logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			log.Fatalf("failed to open logfile: %s", err)
		}
		log.DeferExitHandler(func() { _ = logfile.Chdir() })
		// Use a third-party hook so that we can write colored logs to stderr
		// and uncolored logs to the file.
		log.AddHook(lfshook.NewHook(logfile, &log.TextFormatter{}))

		// Suppress logging to stderr if it's not connected to a tty.
		if !isatty.IsTerminal(os.Stderr.Fd()) {
			log.SetOutput(ioutil.Discard)
		}
	}

	// Init Cloudflare API client.
	err = initializeAPI()
	if err != nil {
		log.Fatalf("failed to init CloudFlare API: %s", err)
	}

	// Fetch external IP.
	var ip net.IP
	dieTrying("fetching external IP address", func() error {
		var err error
		ip, err = getIPAddress(conf.IP.EchoServer)
		return err
	})
	log.Infof("external IP address: %s", ip.String())

	// Set DNS records.
	var wg sync.WaitGroup
	errs := make(chan error, len(conf.DNS.Domains))
	for _, dd := range conf.DNS.Domains {
		wg.Add(1)
		go func(d domainConfig) {
			zone := d.Zone
			domain := d.Domain

			ttl := d.TTL
			if ttl == 0 {
				ttl = conf.DNS.TTL
			}

			var proxied bool
			if d.Proxied != nil {
				proxied = *d.Proxied
			} else {
				proxied = *conf.DNS.Proxied
			}

			err := failTrying("configuring DNS for "+domain, func() error {
				return dnsCreateOrUpdate(dnsParams{
					ZoneName: zone,
					Name:     domain,
					Type:     "A",
					Content:  ip.String(),
					TTL:      ttl,
					Proxied:  proxied,
				})
			})
			if err != nil {
				errs <- err
			}
			wg.Done()
		}(dd)
	}
	wg.Wait()
	select {
	case <-errs:
		log.Exit(1)
	default:
		log.Exit(0)
	}
}
