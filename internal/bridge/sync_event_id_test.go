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

package bridge_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-proton-api/server"
	"github.com/ProtonMail/proton-bridge/v3/internal/bridge"
	"github.com/ProtonMail/proton-bridge/v3/internal/constants"
	"github.com/ProtonMail/proton-bridge/v3/internal/events"
	"github.com/ProtonMail/proton-bridge/v3/internal/vault"
	"github.com/emersion/go-imap"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const syncEventIDNumMsg = 256

type syncEventIDEnv struct {
	userID    string
	addrID    string
	labelID   string
	allowSync *atomic.Bool
}

// withSyncEventIDUser creates a user with a sync event ID and a label.
func withSyncEventIDUser(t *testing.T, fn func(context.Context, *server.Server, *proton.NetCtl, bridge.Locator, []byte, syncEventIDEnv)) {
	t.Helper()

	withEnv(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte) {
		userID, addrID, err := s.CreateUser("imap", password)
		require.NoError(t, err)

		labelID, err := s.CreateLabel(userID, "folder", "", proton.LabelTypeFolder)
		require.NoError(t, err)

		withClient(ctx, t, s, "imap", password, func(ctx context.Context, c *proton.Client) {
			createNumMessages(ctx, t, c, addrID, labelID, syncEventIDNumMsg)
		})

		allowSync := &atomic.Bool{}
		blockMessageSyncUntil(t, s, allowSync)

		fn(ctx, s, netCtl, locator, storeKey, syncEventIDEnv{
			userID:    userID,
			addrID:    addrID,
			labelID:   labelID,
			allowSync: allowSync,
		})
	}, server.WithTLS(false))
}

// blockMessageSyncUntil blocks message sync until the allowSync flag is set.
func blockMessageSyncUntil(_ *testing.T, s *server.Server, allowSync *atomic.Bool) {
	s.AddStatusHook(func(request *http.Request) (int, bool) {
		if request.Method == "GET" && strings.Contains(request.URL.Path, "/mail/v4/messages/") {
			if !allowSync.Load() {
				return http.StatusTooManyRequests, true
			}
		}

		return 0, false
	})
}

// withSyncEventIDBridge creates a bridge with a sync event ID and a label.
func withSyncEventIDBridge(
	ctx context.Context,
	t *testing.T,
	apiURL string,
	netCtl *proton.NetCtl,
	locator bridge.Locator,
	vaultKey []byte,
	fn func(*bridge.Bridge),
) {
	t.Helper()

	withBridge(ctx, t, apiURL, netCtl, locator, vaultKey, func(b *bridge.Bridge, mocks *bridge.Mocks) {
		mocks.Reporter.EXPECT().ReportMessageWithContext(gomock.Any(), gomock.Any()).AnyTimes()
		fn(b)
	})
}

// syncEventIDContinueEventProcess continues the event process after a sync has completed.
func syncEventIDContinueEventProcess(
	ctx context.Context,
	t *testing.T,
	server *server.Server,
	bridge *bridge.Bridge,
) {
	t.Helper()

	info, err := bridge.QueryUserInfo("imap")
	require.NoError(t, err)

	cli, err := eventuallyDial(fmt.Sprintf("%v:%v", constants.Host, bridge.GetIMAPPort()))
	require.NoError(t, err)
	require.NoError(t, cli.Login(info.Addresses[0], string(info.BridgePass)))
	defer func() { _ = cli.Logout() }()

	randomLabel := uuid.NewString()

	withClient(ctx, t, server, "imap", password, func(ctx context.Context, c *proton.Client) {
		require.NoError(t, getErr(c.CreateLabel(ctx, proton.CreateLabelReq{
			Name:  randomLabel,
			Color: "#f66",
			Type:  proton.LabelTypeLabel,
		})))
	})

	require.Eventually(t, func() bool {
		return slices.IndexFunc(clientList(cli), func(mailbox *imap.MailboxInfo) bool {
			return mailbox.Name == "Labels/"+randomLabel
		}) >= 0
	}, 1*time.Minute, 10*time.Second)
}

// loginAndWaitSyncStarted logs in and waits for the sync to start.
func loginAndWaitSyncStarted(
	ctx context.Context,
	t *testing.T,
	bridge *bridge.Bridge,
	syncStartedCh <-chan events.SyncStarted,
) string {
	t.Helper()

	userID, err := bridge.LoginFull(ctx, "imap", password, nil, nil)
	require.NoError(t, err)
	require.Equal(t, userID, (<-syncStartedCh).UserID)

	return userID
}

func waitForAPIEventsBeyondInitialBookmark(
	ctx context.Context,
	t *testing.T,
	s *server.Server,
	userID string,
	initialBookmark string,
) {
	t.Helper()

	_, err := s.CreateLabel(userID, uuid.NewString(), "", proton.LabelTypeFolder)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return latestAPIEventID(ctx, t, s, "imap", password) != initialBookmark
	}, 1*time.Minute, 10*time.Second)
}

func requireBookmarksDiffer(t *testing.T, b1, b2 string) {
	t.Helper()

	require.NotEmpty(t, b2)
	require.NotEqual(t, b1, b2, "bookmark should differ after new events during sync")
}

func waitForUserConnected(t *testing.T, b *bridge.Bridge, userID string) {
	t.Helper()

	require.Eventually(t, func() bool {
		return slices.Contains(getConnectedUserIDs(t, b), userID)
	}, 30*time.Second, 1*time.Second)
}

func waitForSyncRestartedWithBookmark(t *testing.T, b *bridge.Bridge, userID, wantBookmark string) {
	t.Helper()

	requireStartSyncEventIDEventually(t, b, userID, wantBookmark)
	require.False(t, loadLiveIMAPSyncStatus(t, b, userID).IsComplete())
}

// Check that the initial sync sets a bookmark.
func TestBridge_SyncEventID_InitialSyncSetsBookmark(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, done := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer done()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)

			bookmark := requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)
			require.NotEmpty(t, bookmark)
			require.False(t, loadLiveIMAPSyncStatus(t, b, env.userID).IsComplete())
		})
	})
}

// Check that the complete sync clears the bookmark and rewinds the sync.
func TestBridge_SyncEventID_CompleteClearsBookmarkAndRewinds(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)

			startBookmark := requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)

			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)

			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == startBookmark
			}, 30*time.Second, 1*time.Second)
		})
	})
}

// Check that the event stream continues after the sync is complete.
func TestBridge_SyncEventID_ContinuesEventStreamAfterComplete(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)

			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)

			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			syncEventIDContinueEventProcess(ctx, t, s, b)
		})
	})
}

// Check that if a resync is performed during a sync, the sync is rewinded and the bookmark is updated.
func TestBridge_SyncEventID_ResyncDuringInitialSync(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			initialBookmark := requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)

			waitForAPIEventsBeyondInitialBookmark(ctx, t, s, env.userID, initialBookmark)

			b.Repair()
			vaultEventID := loadVaultEventID(t, locator, storeKey, env.userID)
			require.NotEmpty(t, vaultEventID)
			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			requireStartSyncEventIDEventually(t, b, env.userID, vaultEventID)
			requireBookmarksDiffer(t, initialBookmark, vaultEventID)

			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)

			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == vaultEventID
			}, 30*time.Second, 1*time.Second)
		})
	})
}

// Check that if a refresh event happens during a sync, the bookmark is updated to the event from the refresh.
func TestBridge_SyncEventID_RefreshDuringInitialSync(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			initialBookmark := requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)

			require.NoError(t, s.RefreshUser(env.userID, proton.RefreshMail))
			refreshEventID := latestAPIEventID(ctx, t, s, "imap", password)
			requireStartSyncEventIDEventually(t, b, env.userID, refreshEventID)
			requireBookmarksDiffer(t, initialBookmark, refreshEventID)

			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)

			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == refreshEventID
			}, 30*time.Second, 1*time.Second)
		})
	})
}

// Check that if during a sync, a resync is performed and then a refresh event happes, the bookmark is updated to first the refresh event and then the resync event.
func TestBridge_SyncEventID_ResyncThenRefreshDuringInitialSync(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			initialBookmark := requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)
			// wait for new events to be created.
			waitForAPIEventsBeyondInitialBookmark(ctx, t, s, env.userID, initialBookmark)

			// trigger a resync
			b.Repair()
			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			resyncBookmark := requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)
			requireBookmarksDiffer(t, initialBookmark, resyncBookmark) // resync should have a different starting bookmark

			time.Sleep(1 * time.Second)
			// trigger a refresh
			require.NoError(t, s.RefreshUser(env.userID, proton.RefreshMail))
			refreshEventID := latestAPIEventID(ctx, t, s, "imap", password)
			requireStartSyncEventIDEventually(t, b, env.userID, refreshEventID)

			// refresh should have a different starting bookmark than both initial and resync bookmarks.
			requireBookmarksDiffer(t, initialBookmark, refreshEventID)
			requireBookmarksDiffer(t, resyncBookmark, refreshEventID)

			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)

			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == refreshEventID
			}, 30*time.Second, 1*time.Second)
		})
	})
}

func TestBridge_SyncEventID_RefreshThenResyncDuringInitialSync(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			initialBookmark := requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)

			// refresh happens during the initial sync, so the bookmark should be updated to the refresh event.
			require.NoError(t, s.RefreshUser(env.userID, proton.RefreshMail))
			refreshEventID := latestAPIEventID(ctx, t, s, "imap", password)
			requireStartSyncEventIDEventually(t, b, env.userID, refreshEventID)
			requireBookmarksDiffer(t, initialBookmark, refreshEventID)

			waitForAPIEventsBeyondInitialBookmark(ctx, t, s, env.userID, initialBookmark)

			// user triggers a resync after the refresh, so the bookmark should be updated to the resync event.
			b.Repair()
			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			resyncBookmark := loadVaultEventID(t, locator, storeKey, env.userID)
			requireStartSyncEventIDEventually(t, b, env.userID, resyncBookmark)
			requireBookmarksDiffer(t, initialBookmark, resyncBookmark)

			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)

			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == resyncBookmark
			}, 30*time.Second, 1*time.Second)
		})
	})
}

func TestBridge_SyncEventID_RefreshAfterSyncComplete(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		env.allowSync.Store(true)

		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			require.Equal(t, env.userID, (<-syncCh).UserID)
			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)

			env.allowSync.Store(false)

			// a refresh event happens after the sync is complete
			require.NoError(t, s.RefreshUser(env.userID, proton.RefreshMail))
			refreshEventID := latestAPIEventID(ctx, t, s, "imap", password)

			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			requireStartSyncEventIDEventually(t, b, env.userID, refreshEventID)

			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)
			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == refreshEventID
			}, 30*time.Second, 1*time.Second)
		})
	})
}

func TestBridge_SyncEventID_ResyncAfterSyncComplete(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		env.allowSync.Store(true)

		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			require.Equal(t, env.userID, (<-syncCh).UserID)

			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)

			vaultEventID := loadVaultEventID(t, locator, storeKey, env.userID)

			b.Repair()
			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			require.Equal(t, env.userID, (<-syncCh).UserID)

			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == vaultEventID
			}, 30*time.Second, 1*time.Second)
		})
	})
}

func TestBridge_SyncEventID_SplitModeDuringInitialSync(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			initialBookmark := requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)

			waitForAPIEventsBeyondInitialBookmark(ctx, t, s, env.userID, initialBookmark)
			require.NoError(t, b.SetAddressMode(ctx, env.userID, vault.SplitMode))
			time.Sleep(1 * time.Second)
			splitModeBookmark := loadVaultEventID(t, locator, storeKey, env.userID)

			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			requireStartSyncEventIDEventually(t, b, env.userID, splitModeBookmark)
			requireBookmarksDiffer(t, initialBookmark, splitModeBookmark)
			require.False(t, loadLiveIMAPSyncStatus(t, b, env.userID).IsComplete())

			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)
			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
		})
	})
}

func TestBridge_SyncEventID_SplitModeThenRefreshDuringInitialSync(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			initialBookmark := requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)

			waitForAPIEventsBeyondInitialBookmark(ctx, t, s, env.userID, initialBookmark)

			require.NoError(t, b.SetAddressMode(ctx, env.userID, vault.SplitMode))
			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			splitBookmark := requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)
			requireBookmarksDiffer(t, initialBookmark, splitBookmark)

			require.NoError(t, s.RefreshUser(env.userID, proton.RefreshMail))
			refreshEventID := latestAPIEventID(ctx, t, s, "imap", password)
			requireStartSyncEventIDEventually(t, b, env.userID, refreshEventID)
			requireBookmarksDiffer(t, initialBookmark, refreshEventID)
			requireBookmarksDiffer(t, splitBookmark, refreshEventID)

			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)

			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == refreshEventID
			}, 30*time.Second, 1*time.Second)
		})
	})
}

func TestBridge_SyncEventID_ResyncThenSplitModeDuringInitialSync(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			initialBookmark := requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)

			waitForAPIEventsBeyondInitialBookmark(ctx, t, s, env.userID, initialBookmark)

			b.Repair()
			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			vaultEventID := loadVaultEventID(t, locator, storeKey, env.userID)
			requireStartSyncEventIDEventually(t, b, env.userID, vaultEventID)
			requireBookmarksDiffer(t, initialBookmark, vaultEventID)

			require.NoError(t, b.SetAddressMode(ctx, env.userID, vault.SplitMode))
			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			requireStartSyncEventIDEventually(t, b, env.userID, vaultEventID)
			requireBookmarksDiffer(t, initialBookmark, vaultEventID)

			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)
			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == vaultEventID
			}, 30*time.Second, 1*time.Second)
		})
	})
}

// Check that when a bridge instance closes and another instance starts, bridge rewinds correctly to the initial bookmark.
func TestBridge_SyncEventID_BridgeStopsSyncingThenContinuesAndRewinds(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		var initialBookmark string

		// First bridge instance runs the initial sync and sets the initial bookmark then stops.
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			initialBookmark = requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)
		})

		// Check if the initial bookmark is still the same after bridge instance stops.
		require.Equal(t, initialBookmark, loadIMAPSyncStatusFromDisk(t, locator, env.userID).StartSyncEventID)

		// Start another instance and wait for the sync to restart and rewind to the initial bookmark.
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			waitForUserConnected(t, b, env.userID)

			afterRestartBookmark := loadIMAPSyncStatusFromDisk(t, locator, env.userID).StartSyncEventID
			require.Equal(t, initialBookmark, afterRestartBookmark)

			waitForSyncRestartedWithBookmark(t, b, env.userID, afterRestartBookmark)

			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)

			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == afterRestartBookmark
			}, 30*time.Second, 1*time.Second)
		})
	})
}

// Check that when a bridge instance closes and another instance starts and a refresh event happens which resets the bookmark then rewinds correctly to the refresh bookmark.
func TestBridge_SyncEventID_BridgeStopsThenContinuesAndRefreshEventHappens(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		initialBookmark := ""
		// Start a bridge instance and wait for the sync to start and sets the initial bookmark.
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			initialBookmark = requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)
			waitForAPIEventsBeyondInitialBookmark(ctx, t, s, env.userID, initialBookmark)
		})

		// Bridge closes and another instance starts
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			_, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			syncFinishedCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))

			defer func() {
				doneStarted()
				doneSync()
			}()

			// Wait for the sync to restart and check that the initial bookmark is persisted.
			waitForUserConnected(t, b, env.userID)
			afterRestartBookmark := loadIMAPSyncStatusFromDisk(t, locator, env.userID).StartSyncEventID
			require.Equal(t, initialBookmark, afterRestartBookmark)
			waitForSyncRestartedWithBookmark(t, b, env.userID, afterRestartBookmark)

			// Refresh event happens during sync
			require.NoError(t, s.RefreshUser(env.userID, proton.RefreshMail))
			refreshEventID := latestAPIEventID(ctx, t, s, "imap", password)
			require.NotEqual(t, initialBookmark, refreshEventID)

			requireStartSyncEventIDEventually(t, b, env.userID, refreshEventID)
			requireBookmarksDiffer(t, initialBookmark, refreshEventID)

			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncFinishedCh).UserID)

			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == refreshEventID
			}, 30*time.Second, 1*time.Second)
		})
	})
}

// Check that when a bridge instance closes and another instance starts and a refresh event happens, then closes again and another instance starts it rewinds correctly to the refresh bookmark.
func TestBridge_SyncEventID_BridgeRestartRefreshRestartContinuesSync(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		initialBookmark := ""
		// First instance initiates a sync and sets the initial bookmark.
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			initialBookmark = requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)
			waitForAPIEventsBeyondInitialBookmark(ctx, t, s, env.userID, initialBookmark)
		})

		refreshBookmark := ""
		// Bridge closes and another instance starts and a refresh event happens which resets the bookmark, bridge closes again.
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			_, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))

			defer func() {
				doneStarted()
			}()

			// Wait for the sync to restart and check that the initial bookmark is persisted.
			waitForUserConnected(t, b, env.userID)
			afterRestartBookmark := loadIMAPSyncStatusFromDisk(t, locator, env.userID).StartSyncEventID
			require.Equal(t, initialBookmark, afterRestartBookmark)
			waitForSyncRestartedWithBookmark(t, b, env.userID, afterRestartBookmark)

			// Refresh event happens during sync
			require.NoError(t, s.RefreshUser(env.userID, proton.RefreshMail))
			refreshBookmark = latestAPIEventID(ctx, t, s, "imap", password)
			require.NotEqual(t, initialBookmark, refreshBookmark)

			requireStartSyncEventIDEventually(t, b, env.userID, refreshBookmark)
			requireBookmarksDiffer(t, initialBookmark, refreshBookmark)
		})

		// Third instance starts and wait for the sync to finish and check that the right event is rewinded to -> refreshBookmark.
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			_, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			syncFinishedCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer func() {
				doneStarted()
				doneSync()
			}()

			waitForUserConnected(t, b, env.userID)
			afterRestartBookmark := loadIMAPSyncStatusFromDisk(t, locator, env.userID).StartSyncEventID
			require.Equal(t, refreshBookmark, afterRestartBookmark)
			waitForSyncRestartedWithBookmark(t, b, env.userID, afterRestartBookmark)

			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncFinishedCh).UserID)
			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)

			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == refreshBookmark
			}, 30*time.Second, 1*time.Second)
		})
	})
}

func TestBridge_SyncEventID_BridgeStopsSyncingThenContinuesAndResyncHappens(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		var initialBookmark string

		// Start bridge instance and initiate a sync.
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			initialBookmark = requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)
			waitForAPIEventsBeyondInitialBookmark(ctx, t, s, env.userID, initialBookmark)
		})

		// Bridge closes and another instance starts.
		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			// Wait for the sync to restart and check that the initial bookmark is persisted.
			waitForUserConnected(t, b, env.userID)
			afterRestartBookmark := loadIMAPSyncStatusFromDisk(t, locator, env.userID).StartSyncEventID
			require.Equal(t, initialBookmark, afterRestartBookmark)
			waitForSyncRestartedWithBookmark(t, b, env.userID, afterRestartBookmark)

			b.Repair()
			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			resyncBookmark := requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)
			requireBookmarksDiffer(t, initialBookmark, resyncBookmark)

			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)

			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == resyncBookmark
			}, 30*time.Second, 1*time.Second)
		})
	})
}

func TestBridge_SyncEventID_BridgeStopsSyncingThenContinuesAndSplitModeToggles(t *testing.T) {
	withSyncEventIDUser(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, storeKey []byte, env syncEventIDEnv) {
		var initialBookmark string

		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			loginAndWaitSyncStarted(ctx, t, b, syncStartedCh)
			initialBookmark = requireStartSyncEventIDEventuallyNotEmpty(t, b, env.userID)
			waitForAPIEventsBeyondInitialBookmark(ctx, t, s, env.userID, initialBookmark)
		})

		withSyncEventIDBridge(ctx, t, s.GetHostURL(), netCtl, locator, storeKey, func(b *bridge.Bridge) {
			syncStartedCh, doneStarted := chToType[events.Event, events.SyncStarted](b.GetEvents(events.SyncStarted{}))
			defer doneStarted()

			syncCh, doneSync := chToType[events.Event, events.SyncFinished](b.GetEvents(events.SyncFinished{}))
			defer doneSync()

			waitForUserConnected(t, b, env.userID)
			waitForSyncRestartedWithBookmark(t, b, env.userID, initialBookmark)

			vaultEventID := loadVaultEventID(t, locator, storeKey, env.userID)
			require.NoError(t, b.SetAddressMode(ctx, env.userID, vault.SplitMode))
			require.Equal(t, env.userID, (<-syncStartedCh).UserID)
			requireStartSyncEventIDEventually(t, b, env.userID, vaultEventID)
			requireBookmarksDiffer(t, initialBookmark, vaultEventID)

			env.allowSync.Store(true)
			require.Equal(t, env.userID, (<-syncCh).UserID)
			requireStartSyncEventIDEventuallyEmpty(t, b, env.userID)
			require.Eventually(t, func() bool {
				return loadVaultEventID(t, locator, storeKey, env.userID) == vaultEventID
			}, 30*time.Second, 1*time.Second)
		})
	})
}
