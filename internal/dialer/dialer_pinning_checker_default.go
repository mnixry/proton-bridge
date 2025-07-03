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

//go:build !build_qa

package dialer

import (
	"crypto/tls"
	"errors"
	"net"
)

// CheckCertificate verifies that the connection presents a known pinned leaf TLS certificate.
func (p *TLSPinChecker) CheckCertificate(conn net.Conn, certificateChainVerificationSkipped bool) error {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return errors.New("connection is not a TLS connection")
	}

	connState := tlsConn.ConnectionState()

	// When certificate chain verification is enabled (e.g., for known API hosts), we expect the TLS handshake to produce verified chains.
	// We then validate that the leaf certificate of at least one verified chain matches a known pinned public key.
	if !certificateChainVerificationSkipped {
		if len(connState.VerifiedChains) == 0 {
			return errors.New("no verified certificate chains")
		}

		for _, chain := range connState.VerifiedChains {
			// Check if the leaf certificate is one of the trusted pins.
			if p.isCertFoundInKnownPins(chain[0]) {
				return nil
			}
		}

		return ErrTLSMismatch
	}

	// When certificate chain verification is skipped (e.g., for DoH proxies using self-signed certs),
	// we only validate the leaf certificate against known pinned public keys.
	if len(connState.PeerCertificates) == 0 {
		return errors.New("no peer certificates available")
	}

	if p.isCertFoundInKnownPins(connState.PeerCertificates[0]) {
		return nil
	}

	return ErrTLSMismatch
}
