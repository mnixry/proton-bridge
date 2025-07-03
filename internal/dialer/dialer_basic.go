// Copyright (c) 2025 Proton AG
//
// This file is part of Proton Mail Bridge.Bridge.
//
// Proton Mail Bridge is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Proton Mail Bridge is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Proton Mail Bridge. If not, see <https://www.gnu.org/licenses/>.

package dialer

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TLSDialer interface {
	DialTLSContext(ctx context.Context, network, address string) (conn net.Conn, err error)
}

type SecureTLSDialer interface {
	DialTLSContext(ctx context.Context, network, address string) (conn net.Conn, err error)
	ShouldSkipCertificateChainVerification(address string) bool
}

func SetBasicTransportTimeouts(t *http.Transport) {
	t.MaxIdleConns = 100
	t.MaxIdleConnsPerHost = 100
	t.IdleConnTimeout = 5 * time.Minute

	t.ExpectContinueTimeout = 500 * time.Millisecond

	// GODT-126: this was initially 10s but logs from users showed a significant number
	// were hitting this timeout, possibly due to flaky wifi taking >10s to reconnect.
	// Bumping to 30s for now to avoid this problem.
	t.ResponseHeaderTimeout = 30 * time.Second

	// If we allow up to 30 seconds for response headers, it is reasonable to allow up
	// to 30 seconds for the TLS handshake to take place.
	t.TLSHandshakeTimeout = 30 * time.Second
}

// CreateTransportWithDialer creates an http.Transport that uses the given dialer to make TLS connections.
func CreateTransportWithDialer(dialer TLSDialer) *http.Transport {
	t := &http.Transport{
		DialTLSContext: dialer.DialTLSContext,

		Proxy: http.ProxyFromEnvironment,
	}

	SetBasicTransportTimeouts(t)

	return t
}

// BasicTLSDialer implements TLSDialer.
type BasicTLSDialer struct {
	hostURL string
}

// NewBasicTLSDialer returns a new BasicTLSDialer.
func NewBasicTLSDialer(hostURL string) *BasicTLSDialer {
	return &BasicTLSDialer{
		hostURL: hostURL,
	}
}

func extractDomain(hostname string) string {
	parts := strings.Split(hostname, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return hostname
}

// ShouldSkipCertificateChainVerification determines whether certificate chain validation should be skipped.
// It compares the domain of the requested address with the configured host URL domain.
// Returns true if the domains don't match (skip verification), false if they do (perform verification).
//
// NOTE: This assumes single-part TLDs (.com, .me) and won't handle multi-part TLDs correctly.
func (d *BasicTLSDialer) ShouldSkipCertificateChainVerification(address string) bool {
	parsedURL, err := url.Parse(d.hostURL)
	if err != nil {
		return true
	}

	addressHost, _, err := net.SplitHostPort(address)
	if err != nil {
		addressHost = address
	}

	hostDomain := extractDomain(parsedURL.Host)
	addressDomain := extractDomain(addressHost)
	return addressDomain != hostDomain
}

// DialTLSContext returns a connection to the given address using the given network.
func (d *BasicTLSDialer) DialTLSContext(ctx context.Context, network, address string) (conn net.Conn, err error) {
	return (&tls.Dialer{
		NetDialer: &net.Dialer{
			Timeout: 30 * time.Second,
		},
		Config: &tls.Config{
			InsecureSkipVerify: d.ShouldSkipCertificateChainVerification(address), //nolint:gosec
		},
	}).DialContext(ctx, network, address)
}
