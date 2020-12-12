package main

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	_client          *http.Client
	_privateIPBlocks []*net.IPNet
)

func init() {
	// Initialize an IPv4-only HTTP client with proper timeout settings.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		// Force IPv4-only.
		return dialer.DialContext(ctx, "tcp4", addr)
	}
	_client = &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	// Initialize private IP blocks.
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	} {
		_, block, _ := net.ParseCIDR(cidr)
		_privateIPBlocks = append(_privateIPBlocks, block)
	}
}

// Supports IPv4 exclusively, for now at least.
func getIPAddress(ipEchoServer string) (net.IP, error) {
	resp, err := _client.Get(ipEchoServer)
	if err != nil {
		return nil, errors.Wrapf(err, "error querying %s", ipEchoServer)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "error querying %s", ipEchoServer)
	}
	addr := strings.TrimSpace(string(body))
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil, errors.Errorf("failed to parse echo server response as IP: %#v", shorten(addr, 100))
	}
	if !isPublicIPv4(ip) {
		return nil, errors.Errorf("not a public IPv4 address: %s", addr)
	}
	return ip, nil
}

func isPublicIPv4(ip net.IP) bool {
	if ip.To4() == nil {
		// Not IPv4.
		return false
	}
	if !ip.IsGlobalUnicast() {
		return false
	}
	for _, block := range _privateIPBlocks {
		if block.Contains(ip) {
			return false
		}
	}
	return true
}

func shorten(s string, length int) string {
	runes := []rune(s)
	if len(runes) > length {
		return string(runes[:length]) + "..."
	}
	return s
}
