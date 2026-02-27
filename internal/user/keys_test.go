// Copyright (c) 2026 Proton AG
//
// This file is part of Proton Mail Bridge.
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
// along with Proton Mail Bridge.  If not, see <https://www.gnu.org/licenses/>.

package user

import (
	"context"
	"testing"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-proton-api/server"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/ProtonMail/gopenpgp/v2/helper"
	"github.com/ProtonMail/proton-bridge/v3/internal/usertypes"
	"github.com/stretchr/testify/require"
)

func BenchmarkAddrKeyRing(b *testing.B) {
	b.StopTimer()

	withAPI(b, context.Background(), func(ctx context.Context, s *server.Server, m *proton.Manager) {
		withAccount(b, s, "username", "password", []string{"email@pm.me"}, func(_ string, _ []string) {
			withUser(b, ctx, s, m, "username", "password", func(user *User) {
				b.StartTimer()

				apiUser, err := user.identityService.GetAPIUser(ctx)
				require.NoError(b, err)

				apiAddrs, err := user.identityService.GetAddresses(ctx)
				require.NoError(b, err)

				for i := 0; i < b.N; i++ {
					require.NoError(b, usertypes.WithAddrKRs(apiUser, apiAddrs, user.vault.KeyPass(), func(_ *crypto.KeyRing, _ map[string]*crypto.KeyRing) error {
						return nil
					}))
				}
			})
		})
	})
}

func TestEncryptSignDecryptVerifyAStringWithRsa1023BitKey(t *testing.T) {
	t.Setenv("GODEBUG", "rsa1024min=0")

	publicArmoredKey := `-----BEGIN PGP PUBLIC KEY BLOCK-----

xo0EXNlzAwED/0/cDTqLl8DD7L9FUbuUAq11QSL8Q+5CmaV8M3L/lwcST3y+Ec9Y
v935Z2KMBfS/1TPDPT4nQTiHMlfrXPTEXs7PJEfSST20RcofihHkApU0gldMS2cL
reVcS9ImhiIKswmv58FBJXdPmbQeFSerkDh4GXahXolW3las+FpXepYzABEBAAHN
LU1heCBNdXN0ZXJtYW5uIDxtYXgubXVzdGVybWFubkBwcm90b25tYWlsLmNoPsLA
PQQTAQgAcQWCXNlzAwMLCQcJEOq/NtslSMoHNRQAAAAAABwAEHNhbHRAbm90YXRp
b25zLm9wZW5wZ3Bqcy5vcmdT5SPSMVJ06kwHPd9QwAU3AhUIAxYAAgIZAQKbAwIe
ARYhBLc3rWsVSha/PcTN0Oq/NtslSMoHAABRWQP8CfBPPbnHWF2SXvu+qj/fL+dE
xClYFhn/SZWfaSw2u8TlnDwh2L3y5LxumkqPOi0xlkm66bXqFBBfrItA8iPhCePt
RazsQ8DsQfAa/FBfSlIIyLoOJWOR54WzknuzMO1WipKVRVmVZ0kpuLRXjWG6zekh
bMSkX1r9H+QK2daXibXOjQRc2XMDAQP/c+gsC5g6vncq3Npzay2fKbWehRSlYmFb
V5tVLbVnW7HbVp8qPxm4K6Pex0IDWqJsKWYjNl/p1ng7g55vfzADC9r8HJD/gM9d
S5dWu+LemDNbF7WomdFLrYbZWbfr9vBsU0r0UidSNO8gN/e2kFkwCEasUZKDXmWs
mWpBkk1zDy8AEQEAAcLALAQYAQgAYAWCXNlzAwkQ6r822yVIygc1FAAAAAAAHAAQ
c2FsdEBub3RhdGlvbnMub3BlbnBncGpzLm9yZ4A6kC9d2C9RrE3eZgSmgR0CmwwW
IQS3N61rFUoWvz3EzdDqvzbbJUjKBwAAB3cD/jMHHLkXYMAcgYZJknHKJuoc+PVw
e3LFYoxAz68zUK2EYWAgulz6gTekadlcmIQYhDUEzCJgSdQUVqHnczrmqwA1P8th
yKB0iP207sgZ7JYN2t1XQA0nNrJo73S2Vnc/zt2bmsIrA4QzGbAkIWSagaRdxIDI
Tyl5lZo5J6eKNs6X
=V+am
-----END PGP PUBLIC KEY BLOCK-----`

	privateArmoredKey := `-----BEGIN PGP PRIVATE KEY BLOCK-----

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
=bMnk
-----END PGP PRIVATE KEY BLOCK-----`

	privateKeyPassphrase := "7NgO4d0h72zt4XuFLOUbg352vhrn.tu"

	message := "message\nnewline"
	armor, err := helper.EncryptMessageArmored(publicArmoredKey, message)
	require.NoError(t, err)

	decrypted, err := helper.DecryptMessageArmored(privateArmoredKey, []byte(privateKeyPassphrase), armor)
	require.NoError(t, err)

	require.Equal(t, message, decrypted)
}
