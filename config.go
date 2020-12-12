package main

import (
	"io/ioutil"
	"strings"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

const _configTemplate = `[ip]
# 'echo_server' is the URL of a service that echos back the IP address of the
# querying client. Such a service is necessary when the device is behind a NAT.
# You can self-host or use a third-party service like ifconfig.co. Required, no
# default.
#echo_server = "https://ifconfig.co/"

[dns]
# 'ttl' is the TTL for each DNS record set. Must be a positive integer. The
# special value 1 is "automatic" (~300s according to Cloudflare, but might
# subject to changes). Optional, with a default of 60.
#
# This setting affects all configured domains, and can be overridden
# individually.
#ttl = 60

# 'proxied' is a boolean that determines whether the record should be proxied by
# Cloudflare. Optional, with a default of false.
#
# This setting affects all configured domains, and can be overridden
# individually.
#proxied = false

# 'dns.domain' is a TOML array of tables. You can configure multiple
# [[dns.domain]]'s and all of them will be configured to your external IP.
[[dns.domain]]
# 'zone' is the name of the zone as seen on your CloudFlare dashboard. Required.
#zone = "example.com"

# 'domain' is the name of the domain for which you want to configure dynamic DNS.
# 'domain' should be fully qualified. Required.
#domain = "ddns.example.com"

# 'ttl' overrides dns.ttl for this particular domain. Optional.
#ttl = 60

# 'proxied' overrides dns.proxied for this particular domain. Optional.
#proxied = false

[logging]
# 'logfile' is the path to the log file. If configured, logs are written to the
# log file instead of stderr (note that a more human-friendly version is still
# written to stderr if it's connected to a tty). Optional, no default.
#logfile = "/var/log/cloudfloat.log"
`

type ipConfig struct {
	EchoServer string `toml:"echo_server"`
}

type domainConfig struct {
	Zone    string
	Domain  string
	TTL     int
	Proxied *bool
}

type dnsConfig struct {
	Domains []domainConfig `toml:"domain"`
	TTL     int
	Proxied *bool
}

type loggingConfig struct {
	Logfile string
}

type config struct {
	IP      ipConfig
	DNS     dnsConfig
	Logging loggingConfig
}

func newConfig() *config {
	proxied := false
	return &config{
		DNS: dnsConfig{
			TTL:     60,
			Proxied: &proxied,
		},
	}
}

func loadConfig(path string) (*config, error) {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", path)
	}
	return parseConfig(body)
}

func parseConfig(body []byte) (*config, error) {
	conf := newConfig()
	if err := toml.Unmarshal(body, conf); err != nil {
		return nil, err
	}
	if err := validateConfig(conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func validateConfig(conf *config) error {
	if conf.IP.EchoServer == "" {
		return errors.New("ip.echo_server must be configured")
	}
	if conf.DNS.TTL == 0 {
		return errors.New("dns.ttl must be a positive integer")
	}
	if conf.DNS.Proxied == nil {
		// This should be configured with a default in newConfig.
		return errors.New("dns.proxied must be configured")
	}
	if len(conf.DNS.Domains) == 0 {
		return errors.New("at least one dns.domain required")
	}
	for i, d := range conf.DNS.Domains {
		if d.Zone == "" {
			return errors.Errorf("dns.domain[%d].zone must be configured", i)
		}
		if d.Domain == "" {
			return errors.Errorf("dns.domain[%d].domain must be configured", i)
		}
		domain := stripTrailingDot(d.Domain)
		zone := stripTrailingDot(d.Zone)
		if !(domain == zone || strings.HasSuffix(domain, "."+zone)) {
			return errors.Errorf("%s not in zone %s, please use FQDN (trailing dot optional)",
				d.Domain, d.Zone)
		}
	}
	return nil
}

func stripTrailingDot(s string) string {
	if strings.HasSuffix(s, ".") {
		return s[:len(s)-1]
	}
	return s
}
