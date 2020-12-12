# cloudfloat

cloudfloat is a dynamic DNS (DDNS) client for domains managed through Cloudflare.

Only IPv4 (`A` record) is supported at the moment.

# Usage

```
Usage:
  cloudfloat [options] <config_file>

The config file is required. Use -dump-config-template to generate a
documented config file template.

Cloudflare credentials are required and are passed in through environment
variables. You may specify one of the following:

- CF_API_TOKEN (must have the permissions to set DNS records for the
  appropriate zones);
- CF_API_KEY and CF_API_EMAIL.

Options:
  -dump-config-template
```

A simple sample configuration:

```toml
[ip]
echo_server = "https://ifconfig.co/"

[[dns.domain]]
zone = "zmwang.dev"
domain = "home.zmwang.dev"

[logging]
logfile = "/var/log/cloudfloat.log"
```

# License

cloudfloat is provided under WTFPL.
