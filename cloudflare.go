package main

import (
	"os"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var _api *cloudflare.API

func initializeAPI() error {
	apiToken := os.Getenv("CF_API_TOKEN")
	apiKey := os.Getenv("CF_API_KEY")
	apiEmail := os.Getenv("CF_API_EMAIL")

	var err error

	if apiToken != "" {
		_api, err = cloudflare.NewWithAPIToken(apiToken)
	} else {
		if apiKey == "" {
			return errors.New("No CF_API_KEY or CF_API_TOKEN environment set")
		}

		if apiEmail == "" {
			return errors.New("No CF_API_EMAIL environment set")
		}
		_api, err = cloudflare.New(apiKey, apiEmail)
	}

	return err
}

type dnsParams struct {
	ZoneName string
	Name     string
	Type     string
	Content  string
	TTL      int
	Proxied  bool
}

// Adapted from https://github.com/cloudflare/cloudflare-go/blob/20d8c262f09e199dcef2c70039cc737e0505c0fb/cmd/flarectl/dns.go#L65
func dnsCreateOrUpdate(p dnsParams) error {
	zoneID, err := _api.ZoneIDByName(p.ZoneName)
	if err != nil {
		return errors.Wrapf(err, "error querying zone %s", p.ZoneName)
	}

	r := cloudflare.DNSRecord{
		Name:    p.Name,
		Type:    p.Type,
		Content: p.Content,
		TTL:     p.TTL,
		Proxied: p.Proxied,
	}

	// Look for an existing record
	records, err := _api.DNSRecords(zoneID, cloudflare.DNSRecord{
		Name: p.Name,
		Type: p.Type,
	})
	if err != nil {
		return errors.Wrapf(err, "error fetching DNS records (%s for %s)", p.Type, p.Name)
	}

	if len(records) > 0 {
		// Record exists - find the ID and update it.
		// If there are multiple matching records we just update the first one.
		if err := _api.UpdateDNSRecord(zoneID, records[0].ID, r); err != nil {
			return errors.Wrapf(err, "error updating DNS record (%s for %s)", p.Type, p.Name)
		}
		log.Infof("updated %s record for %s to %s", p.Type, p.Name, p.Content)
	} else {
		// Record doesn't exist - create it.
		if _, err = _api.CreateDNSRecord(zoneID, r); err != nil {
			return errors.Wrapf(err, "error creating DNS record (%s for %s)", p.Type, p.Name)
		}
		log.Infof("created %s record for %s as %s", p.Type, p.Name, p.Content)
	}

	return nil
}
