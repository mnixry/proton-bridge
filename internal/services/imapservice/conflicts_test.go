// Copyright (c) 2025 Proton AG
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

package imapservice_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ProtonMail/gluon/db"
	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/gluon/reporter"
	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/proton-bridge/v3/internal/services/imapservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type ffProviderFalse struct{}
type ffProviderTrue struct{}

func (f ffProviderFalse) GetFlagValue(_ string) bool {
	return false
}

func (f ffProviderTrue) GetFlagValue(_ string) bool {
	return true
}

type mockLabelNameProvider struct {
	mock.Mock
}

func (m *mockLabelNameProvider) GetUserMailboxByName(ctx context.Context, addrID string, labelName []string) (imap.MailboxData, error) {
	args := m.Called(ctx, addrID, labelName)
	v, ok := args.Get(0).(imap.MailboxData)
	if !ok {
		return imap.MailboxData{}, fmt.Errorf("failed to assert type")
	}
	return v, args.Error(1)
}

type mockIDProvider struct {
	mock.Mock
}

func (m *mockIDProvider) GetGluonID(addrID string) (string, bool) {
	args := m.Called(addrID)
	return args.String(0), args.Bool(1)
}

type mockAPIClient struct {
	mock.Mock
}

func (m *mockAPIClient) GetLabel(ctx context.Context, id string, types ...proton.LabelType) (proton.Label, error) {
	args := m.Called(ctx, id, types)
	v, ok := args.Get(0).(proton.Label)
	if !ok {
		return proton.Label{}, fmt.Errorf("failed to assert type")
	}
	return v, args.Error(1)
}

type mockReporter struct {
	mock.Mock
}

func (m *mockReporter) ReportMessageWithContext(msg string, ctx reporter.Context) error {
	args := m.Called(msg, ctx)
	return args.Error(0)
}

func (m *mockReporter) ReportWarningWithContext(msg string, ctx reporter.Context) error {
	args := m.Called(msg, ctx)
	return args.Error(0)
}

func TestResolveConflict_UnexpectedLabelConflict(t *testing.T) {
	ctx := context.Background()
	label := proton.Label{
		ID:   "label-1",
		Path: []string{"Work"},
		Type: proton.LabelTypeLabel,
	}
	conflictingLabel := proton.Label{
		ID:   "label-2",
		Path: []string{"Work"},
		Type: proton.LabelTypeLabel,
	}
	conflictMbox := imap.MailboxData{
		RemoteID:   "label-2",
		BridgeName: []string{"Labels", "Work"},
	}

	mockLabelProvider := new(mockLabelNameProvider)
	mockIDProvider := new(mockIDProvider)
	mockClient := new(mockAPIClient)
	mockReporter := new(mockReporter)

	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id", imapservice.GetMailboxName(label)).
		Return(conflictMbox, nil)
	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id", true)
	mockClient.On("GetLabel", mock.Anything, "label-2", mock.Anything).
		Return(conflictingLabel, nil)
	mockReporter.On("ReportMessageWithContext", "Unexpected label conflict", mock.Anything).
		Return(nil)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	resolver := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{}).
		NewConflictResolver([]*imapservice.Connector{connector})

	visited := make(map[string]bool)
	_, err := resolver.ResolveConflict(ctx, label, visited)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected label conflict")
}

func TestResolveDiscrepancy_LabelDoesNotExist(t *testing.T) {
	ctx := context.Background()
	label := proton.Label{
		ID:   "label-id-1",
		Name: "Inbox",
		Type: proton.LabelTypeLabel,
	}

	mockLabelProvider := new(mockLabelNameProvider)
	mockIDProvider := new(mockIDProvider)
	mockClient := new(mockAPIClient)
	mockReporter := new(mockReporter)

	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", imapservice.GetMailboxName(label)).
		Return(imap.MailboxData{}, db.ErrNotFound)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewConflictResolver(connectors)

	visited := make(map[string]bool)
	fn, err := resolver.ResolveConflict(ctx, label, visited)

	assert.NoError(t, err)
	updates := fn()
	assert.Len(t, updates, 1)
	muc, ok := updates[0].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(label.ID), muc.Mailbox.ID)
}

func TestResolveConflict_MailboxFetchError(t *testing.T) {
	ctx := context.Background()
	label := proton.Label{
		ID:   "111",
		Path: []string{"Work"},
		Type: proton.LabelTypeLabel,
	}

	mockLabelProvider := new(mockLabelNameProvider)
	mockIDProvider := new(mockIDProvider)
	mockClient := new(mockAPIClient)
	mockReporter := new(mockReporter)

	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id", imapservice.GetMailboxName(label)).
		Return(imap.MailboxData{}, errors.New("database connection error"))
	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id", true)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	resolver := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{}).
		NewConflictResolver([]*imapservice.Connector{connector})

	visited := make(map[string]bool)
	_, err := resolver.ResolveConflict(ctx, label, visited)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection error")
}

func TestResolveDiscrepancy_ConflictingLabelDeletedRemotely(t *testing.T) {
	ctx := context.Background()
	label := proton.Label{
		ID:   "label-new",
		Path: []string{"Work"},
		Type: proton.LabelTypeLabel,
	}
	conflictMbox := imap.MailboxData{
		RemoteID:   "label-old",
		BridgeName: []string{"Labels", "Work"},
	}

	mockLabelProvider := new(mockLabelNameProvider)
	mockIDProvider := new(mockIDProvider)
	mockClient := new(mockAPIClient)
	mockReporter := new(mockReporter)

	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", imapservice.GetMailboxName(label)).
		Return(conflictMbox, nil)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	mockClient.On("GetLabel", mock.Anything, "label-old", mock.Anything).
		Return(proton.Label{}, proton.ErrNoSuchLabel)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewConflictResolver(connectors)

	visited := make(map[string]bool)
	fn, err := resolver.ResolveConflict(ctx, label, visited)

	assert.NoError(t, err)
	updates := fn()
	assert.Len(t, updates, 2)
	deleted, ok := updates[0].(*imap.MailboxDeleted)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID("label-old"), deleted.MailboxID)

	updated, ok := updates[1].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, "Work", updated.Mailbox.Name[len(updated.Mailbox.Name)-1])
}

func TestResolveDiscrepancy_LabelAlreadyCorrect(t *testing.T) {
	ctx := context.Background()
	label := proton.Label{
		ID:   "label-id-1",
		Name: "Personal",
		Type: proton.LabelTypeLabel,
	}
	mbox := imap.MailboxData{
		RemoteID:   "label-id-1",
		BridgeName: []string{"Labels", "Personal"},
	}

	mockLabelProvider := new(mockLabelNameProvider)
	mockIDProvider := new(mockIDProvider)
	mockClient := new(mockAPIClient)
	mockReporter := new(mockReporter)

	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", imapservice.GetMailboxName(label)).
		Return(mbox, nil)
	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewConflictResolver(connectors)

	visited := make(map[string]bool)
	fn, err := resolver.ResolveConflict(ctx, label, visited)

	assert.NoError(t, err)
	assert.Len(t, fn(), 0)
}

func TestResolveConflict_DeepNestedPath(t *testing.T) {
	ctx := context.Background()
	label := proton.Label{
		ID:   "111",
		Path: []string{"Level1", "Level2", "Level3", "DeepFolder"},
		Type: proton.LabelTypeFolder,
	}

	mockLabelProvider := new(mockLabelNameProvider)
	mockIDProvider := new(mockIDProvider)
	mockClient := new(mockAPIClient)
	mockReporter := new(mockReporter)

	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id", imapservice.GetMailboxName(label)).
		Return(imap.MailboxData{}, db.ErrNotFound)
	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id", true)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	resolver := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{}).
		NewConflictResolver([]*imapservice.Connector{connector})

	visited := make(map[string]bool)
	fn, err := resolver.ResolveConflict(ctx, label, visited)

	assert.NoError(t, err)
	updates := fn()
	assert.Len(t, updates, 1)

	updated, ok := updates[0].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID("111"), updated.Mailbox.ID)
	expectedName := imapservice.GetMailboxName(label)
	assert.Equal(t, expectedName, updated.Mailbox.Name)
}

func TestResolveLabelDiscrepancy_LabelSwap(t *testing.T) {
	apiLabels := []proton.Label{
		{
			ID:   "111",
			Path: []string{"X"},
			Type: proton.LabelTypeLabel,
		},
		{
			ID:   "222",
			Path: []string{"Y"},
			Type: proton.LabelTypeLabel,
		},
	}

	gluonLabels := []imap.MailboxData{
		{
			RemoteID:   "111",
			BridgeName: []string{"Labels", "Y"},
		},
		{
			RemoteID:   "222",
			BridgeName: []string{"Labels", "X"},
		},
	}

	mockLabelProvider := new(mockLabelNameProvider)
	mockClient := new(mockAPIClient)
	mockIDProvider := new(mockIDProvider)
	mockReporter := new(mockReporter)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	for _, mbox := range gluonLabels {
		mockLabelProvider.
			On("GetUserMailboxByName", mock.Anything, "gluon-id-1", mbox.BridgeName).
			Return(mbox, nil)
	}

	for _, label := range apiLabels {
		mockClient.
			On("GetLabel", mock.Anything, label.ID, []proton.LabelType{proton.LabelTypeFolder, proton.LabelTypeLabel, proton.LabelTypeSystem}).
			Return(label, nil)
	}

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewConflictResolver(connectors)

	visited := make(map[string]bool)
	fn, err := resolver.ResolveConflict(context.Background(), apiLabels[0], visited)
	require.NoError(t, err)

	updates := fn()
	assert.NotEmpty(t, updates)
	assert.Equal(t, 3, len(updates)) // We expect three calls to be made for a swap operation.

	updateOne, ok := updates[0].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[0].ID), updateOne.Mailbox.ID)
	assert.Equal(t, "tmp_X", updateOne.Mailbox.Name[len(updateOne.Mailbox.Name)-1])

	updateTwo, ok := updates[1].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[1].ID), updateTwo.Mailbox.ID)
	assert.Equal(t, "Y", updateTwo.Mailbox.Name[len(updateTwo.Mailbox.Name)-1])

	updateThree, ok := updates[2].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[0].ID), updateThree.Mailbox.ID)
	assert.Equal(t, "X", updateThree.Mailbox.Name[len(updateThree.Mailbox.Name)-1])
}

func TestResolveLabelDiscrepancy_LabelSwapExtended(t *testing.T) {
	apiLabels := []proton.Label{
		{
			ID:   "111",
			Path: []string{"X"},
			Type: proton.LabelTypeLabel,
		},
		{
			ID:   "222",
			Path: []string{"Y"},
			Type: proton.LabelTypeLabel,
		},
		{
			ID:   "333",
			Path: []string{"Z"},
			Type: proton.LabelTypeLabel,
		},
		{
			ID:   "444",
			Path: []string{"D"},
			Type: proton.LabelTypeLabel,
		},
	}

	gluonLabels := []imap.MailboxData{
		{
			RemoteID:   "111",
			BridgeName: []string{"Labels", "D"},
		},
		{
			RemoteID:   "222",
			BridgeName: []string{"Labels", "Z"},
		},
		{
			RemoteID:   "333",
			BridgeName: []string{"Labels", "Y"},
		},
		{
			RemoteID:   "444",
			BridgeName: []string{"Labels", "X"},
		},
	}

	mockLabelProvider := new(mockLabelNameProvider)
	mockClient := new(mockAPIClient)
	mockIDProvider := new(mockIDProvider)
	mockReporter := new(mockReporter)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	for _, mbox := range gluonLabels {
		mockLabelProvider.
			On("GetUserMailboxByName", mock.Anything, "gluon-id-1", mbox.BridgeName).
			Return(mbox, nil)
	}

	for _, label := range apiLabels {
		mockClient.
			On("GetLabel", mock.Anything, label.ID, []proton.LabelType{proton.LabelTypeFolder, proton.LabelTypeLabel, proton.LabelTypeSystem}).
			Return(label, nil)
	}

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewConflictResolver(connectors)

	fn, err := resolver.ResolveConflict(context.Background(), apiLabels[0], make(map[string]bool))
	require.NoError(t, err)

	updates := fn()
	assert.NotEmpty(t, updates)
	// Three calls yet again for a swap operation.
	assert.Equal(t, 3, len(updates))
	updateOne, ok := updates[0].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[0].ID), updateOne.Mailbox.ID)
	assert.Equal(t, "tmp_X", updateOne.Mailbox.Name[len(updateOne.Mailbox.Name)-1])

	updateTwo, ok := updates[1].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[3].ID), updateTwo.Mailbox.ID)
	assert.Equal(t, "D", updateTwo.Mailbox.Name[len(updateTwo.Mailbox.Name)-1])

	updateThree, ok := updates[2].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[0].ID), updateThree.Mailbox.ID)
	assert.Equal(t, "X", updateThree.Mailbox.Name[len(updateThree.Mailbox.Name)-1])

	// Fix the secondary swap.
	fn, err = resolver.ResolveConflict(context.Background(), apiLabels[1], make(map[string]bool))
	require.NoError(t, err)

	updates = fn()
	assert.Equal(t, 3, len(updates))
	updateOne, ok = updates[0].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[1].ID), updateOne.Mailbox.ID)
	assert.Equal(t, "tmp_Y", updateOne.Mailbox.Name[len(updateOne.Mailbox.Name)-1])

	updateTwo, ok = updates[1].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[2].ID), updateTwo.Mailbox.ID)
	assert.Equal(t, "Z", updateTwo.Mailbox.Name[len(updateTwo.Mailbox.Name)-1])

	updateThree, ok = updates[2].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[1].ID), updateThree.Mailbox.ID)
	assert.Equal(t, "Y", updateThree.Mailbox.Name[len(updateThree.Mailbox.Name)-1])
}

func TestResolveLabelDiscrepancy_LabelSwapCyclic(t *testing.T) {
	apiLabels := []proton.Label{
		{ID: "111", Path: []string{"A"}, Type: proton.LabelTypeLabel},
		{ID: "222", Path: []string{"B"}, Type: proton.LabelTypeLabel},
		{ID: "333", Path: []string{"C"}, Type: proton.LabelTypeLabel},
		{ID: "444", Path: []string{"D"}, Type: proton.LabelTypeLabel},
	}

	gluonLabels := []imap.MailboxData{
		{RemoteID: "111", BridgeName: []string{"Labels", "D"}}, // A <- D
		{RemoteID: "222", BridgeName: []string{"Labels", "A"}}, // B <- A
		{RemoteID: "333", BridgeName: []string{"Labels", "B"}}, // C <- B
		{RemoteID: "444", BridgeName: []string{"Labels", "C"}}, // D <- C
	}

	mockLabelProvider := new(mockLabelNameProvider)
	mockClient := new(mockAPIClient)
	mockIDProvider := new(mockIDProvider)
	mockReporter := new(mockReporter)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	for _, mbox := range gluonLabels {
		mockLabelProvider.
			On("GetUserMailboxByName", mock.Anything, "gluon-id-1", mbox.BridgeName).
			Return(mbox, nil)
	}

	for _, label := range apiLabels {
		mockClient.
			On("GetLabel", mock.Anything, label.ID, []proton.LabelType{proton.LabelTypeFolder, proton.LabelTypeLabel, proton.LabelTypeSystem}).
			Return(label, nil)
	}

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewConflictResolver(connectors)

	fn, err := resolver.ResolveConflict(context.Background(), apiLabels[0], make(map[string]bool))
	require.NoError(t, err)

	updates := fn()
	assert.NotEmpty(t, updates)
	assert.Equal(t, 5, len(updates))

	updateOne, ok := updates[0].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[0].ID), updateOne.Mailbox.ID)
	assert.Equal(t, "tmp_A", updateOne.Mailbox.Name[len(updateOne.Mailbox.Name)-1])

	updateTwo, ok := updates[1].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[3].ID), updateTwo.Mailbox.ID)
	assert.Equal(t, "D", updateTwo.Mailbox.Name[len(updateTwo.Mailbox.Name)-1])

	updateThree, ok := updates[2].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[2].ID), updateThree.Mailbox.ID)
	assert.Equal(t, "C", updateThree.Mailbox.Name[len(updateThree.Mailbox.Name)-1])

	updateFour, ok := updates[3].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[1].ID), updateFour.Mailbox.ID)
	assert.Equal(t, "B", updateFour.Mailbox.Name[len(updateFour.Mailbox.Name)-1])

	updateFive, ok := updates[4].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[0].ID), updateFive.Mailbox.ID)
	assert.Equal(t, "A", updateFive.Mailbox.Name[len(updateFive.Mailbox.Name)-1])
}

func TestResolveLabelDiscrepancy_LabelSwapCyclicWithDeletedLabel(t *testing.T) {
	apiLabels := []proton.Label{
		{ID: "111", Path: []string{"A"}, Type: proton.LabelTypeLabel},
		{ID: "333", Path: []string{"C"}, Type: proton.LabelTypeLabel},
		{ID: "444", Path: []string{"D"}, Type: proton.LabelTypeLabel},
	}

	gluonLabels := []imap.MailboxData{
		{RemoteID: "111", BridgeName: []string{"Labels", "D"}},
		{RemoteID: "222", BridgeName: []string{"Labels", "A"}},
		{RemoteID: "333", BridgeName: []string{"Labels", "B"}},
		{RemoteID: "444", BridgeName: []string{"Labels", "C"}},
	}

	mockLabelProvider := new(mockLabelNameProvider)
	mockClient := new(mockAPIClient)
	mockIDProvider := new(mockIDProvider)
	mockReporter := new(mockReporter)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	for _, mbox := range gluonLabels {
		mockLabelProvider.
			On("GetUserMailboxByName", mock.Anything, "gluon-id-1", mbox.BridgeName).
			Return(mbox, nil)
	}

	for _, label := range apiLabels {
		mockClient.
			On("GetLabel", mock.Anything, label.ID, []proton.LabelType{proton.LabelTypeFolder, proton.LabelTypeLabel, proton.LabelTypeSystem}).
			Return(label, nil)
	}
	mockClient.On("GetLabel", mock.Anything, "222", []proton.LabelType{proton.LabelTypeFolder, proton.LabelTypeLabel, proton.LabelTypeSystem}).Return(proton.Label{}, proton.ErrNoSuchLabel)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewConflictResolver(connectors)

	fn, err := resolver.ResolveConflict(context.Background(), apiLabels[2], make(map[string]bool))
	require.NoError(t, err)

	updates := fn()
	assert.NotEmpty(t, updates)
	assert.Equal(t, 3, len(updates))

	updateOne, ok := updates[0].(*imap.MailboxDeleted)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID("222"), updateOne.MailboxID)

	updateTwo, ok := updates[1].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[0].ID), updateTwo.Mailbox.ID)
	assert.Equal(t, "A", updateTwo.Mailbox.Name[len(updateTwo.Mailbox.Name)-1])

	updateThree, ok := updates[2].(*imap.MailboxUpdatedOrCreated)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID(apiLabels[2].ID), updateThree.Mailbox.ID)
	assert.Equal(t, "D", updateThree.Mailbox.Name[len(updateThree.Mailbox.Name)-1])
}

func TestResolveLabelDiscrepancy_LabelSwapCyclicWithDeletedLabel_KillSwitchEnabled(t *testing.T) {
	apiLabels := []proton.Label{
		{ID: "111", Path: []string{"A"}, Type: proton.LabelTypeLabel},
		{ID: "333", Path: []string{"C"}, Type: proton.LabelTypeLabel},
		{ID: "444", Path: []string{"D"}, Type: proton.LabelTypeLabel},
	}

	gluonLabels := []imap.MailboxData{
		{RemoteID: "111", BridgeName: []string{"Labels", "D"}},
		{RemoteID: "222", BridgeName: []string{"Labels", "A"}},
		{RemoteID: "333", BridgeName: []string{"Labels", "B"}},
		{RemoteID: "444", BridgeName: []string{"Labels", "C"}},
	}

	mockLabelProvider := new(mockLabelNameProvider)
	mockClient := new(mockAPIClient)
	mockIDProvider := new(mockIDProvider)
	mockReporter := new(mockReporter)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	for _, mbox := range gluonLabels {
		mockLabelProvider.
			On("GetUserMailboxByName", mock.Anything, "gluon-id-1", mbox.BridgeName).
			Return(mbox, nil)
	}

	for _, label := range apiLabels {
		mockClient.
			On("GetLabel", mock.Anything, label.ID, []proton.LabelType{proton.LabelTypeFolder, proton.LabelTypeLabel, proton.LabelTypeSystem}).
			Return(label, nil)
	}
	mockClient.On("GetLabel", mock.Anything, "222", []proton.LabelType{proton.LabelTypeFolder, proton.LabelTypeLabel, proton.LabelTypeSystem}).Return(proton.Label{}, proton.ErrNoSuchLabel)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderTrue{})
	resolver := manager.NewConflictResolver(connectors)

	fn, err := resolver.ResolveConflict(context.Background(), apiLabels[2], make(map[string]bool))
	require.NoError(t, err)

	updates := fn()
	assert.Empty(t, updates)
}

func TestInternalLabelConflictResolver_NoConflicts(t *testing.T) {
	ctx := context.Background()

	mockLabelProvider := new(mockLabelNameProvider)
	mockClient := new(mockAPIClient)
	mockIDProvider := new(mockIDProvider)
	mockReporter := new(mockReporter)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Folders"}).
		Return(imap.MailboxData{}, db.ErrNotFound)
	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Labels"}).
		Return(imap.MailboxData{}, db.ErrNotFound)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewInternalLabelConflictResolver(connectors)

	apiLabels := make(map[string]proton.Label)
	fn, err := resolver.ResolveConflict(ctx, apiLabels)
	assert.NoError(t, err)

	updates := fn()
	assert.Empty(t, updates)
}

func TestInternalLabelConflictResolver_CorrectIDs(t *testing.T) {
	ctx := context.Background()

	mockLabelProvider := new(mockLabelNameProvider)
	mockClient := new(mockAPIClient)
	mockIDProvider := new(mockIDProvider)
	mockReporter := new(mockReporter)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Folders"}).
		Return(imap.MailboxData{RemoteID: "Folders", BridgeName: []string{"Folders"}}, nil)
	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Labels"}).
		Return(imap.MailboxData{RemoteID: "Labels", BridgeName: []string{"Labels"}}, nil)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewInternalLabelConflictResolver(connectors)

	apiLabels := make(map[string]proton.Label)
	fn, err := resolver.ResolveConflict(ctx, apiLabels)
	assert.NoError(t, err)

	updates := fn()
	assert.Empty(t, updates)
}

type mockMailboxCountProvider struct {
	mock.Mock
}

func (m *mockMailboxCountProvider) GetUserMailboxCountByInternalID(ctx context.Context, addrID string, internalID imap.InternalMailboxID) (int, error) {
	args := m.Called(ctx, addrID, internalID)
	return args.Int(0), args.Error(1)
}

func TestInternalLabelConflictResolver_ConflictingNonAPILabel_ZeroCount(t *testing.T) {
	ctx := context.Background()

	mockLabelProvider := new(mockLabelNameProvider)
	mockClient := new(mockAPIClient)
	mockIDProvider := new(mockIDProvider)
	mockReporter := new(mockReporter)
	mockCountProvider := new(mockMailboxCountProvider)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	// Mock mailbox fetch to return conflicting mailbox
	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Folders"}).
		Return(imap.MailboxData{RemoteID: "wrong-id", BridgeName: []string{"Folders"}, InternalID: imap.InternalMailboxID(123)}, nil)
	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Labels"}).
		Return(imap.MailboxData{}, db.ErrNotFound)

	// Mock message count fetch to return 0 messages.
	mockLabelProvider.On("GetMailboxMessageCount", mock.Anything, "gluon-id-1", imap.InternalMailboxID(123)).
		Return(0, nil)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	mockCountProvider.On("GetUserMailboxCountByInternalID",
		mock.Anything,
		"addr-1",
		imap.InternalMailboxID(123)).
		Return(0, nil)

	connector.SetMailboxCountProviderTest(mockCountProvider)
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewInternalLabelConflictResolver(connectors)

	// API labels don't contain the conflicting label ID
	apiLabels := make(map[string]proton.Label)
	fn, err := resolver.ResolveConflict(ctx, apiLabels)
	assert.NoError(t, err)

	updates := fn()
	assert.Len(t, updates, 1)

	deleted, ok := updates[0].(*imap.MailboxDeletedSilent)
	assert.True(t, ok)
	assert.Equal(t, imap.MailboxID("wrong-id"), deleted.MailboxID)
}

func TestInternalLabelConflictResolver_ConflictingNonAPILabel_PositiveCount(t *testing.T) {
	ctx := context.Background()

	mockLabelProvider := new(mockLabelNameProvider)
	mockClient := new(mockAPIClient)
	mockIDProvider := new(mockIDProvider)
	mockReporter := new(mockReporter)
	mockCountProvider := new(mockMailboxCountProvider)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	mockReporter.On("ReportWarningWithContext", mock.Anything, mock.Anything).
		Return(nil)

	// Mock mailbox fetch to return conflicting mailbox
	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Folders"}).
		Return(imap.MailboxData{RemoteID: "wrong-id", BridgeName: []string{"Folders"}, InternalID: imap.InternalMailboxID(123)}, nil)
	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Labels"}).
		Return(imap.MailboxData{}, db.ErrNotFound)

	// Mock message count fetch to return 0 messages.
	mockLabelProvider.On("GetMailboxMessageCount", mock.Anything, "gluon-id-1", imap.InternalMailboxID(123)).
		Return(0, nil)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	mockCountProvider.On("GetUserMailboxCountByInternalID",
		mock.Anything,
		"addr-1",
		imap.InternalMailboxID(123)).
		Return(10, nil)

	connector.SetMailboxCountProviderTest(mockCountProvider)
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewInternalLabelConflictResolver(connectors)

	// API labels don't contain the conflicting label ID
	apiLabels := make(map[string]proton.Label)
	fn, err := resolver.ResolveConflict(ctx, apiLabels)
	assert.EqualError(t, err, "internal mailbox conflicting non-api label has associated messages")

	updates := fn()
	assert.Empty(t, updates, 0)
}

func TestInternalLabelConflictResolver_ConflictingAPILabelSameName(t *testing.T) {
	ctx := context.Background()

	mockLabelProvider := new(mockLabelNameProvider)
	mockClient := new(mockAPIClient)
	mockIDProvider := new(mockIDProvider)
	mockReporter := new(mockReporter)
	mockCountProvider := new(mockMailboxCountProvider)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Folders"}).
		Return(imap.MailboxData{RemoteID: "api-label-id", BridgeName: []string{"Folders"}}, nil)
	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Labels"}).
		Return(imap.MailboxData{}, db.ErrNotFound)

	mockReporter.On("ReportMessageWithContext", "Internal mailbox name conflict. Same-name mailbox is returned by API", mock.Anything).
		Return(nil)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connector.SetMailboxCountProviderTest(mockCountProvider)
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewInternalLabelConflictResolver(connectors)

	// API user label with empty path.
	apiLabels := map[string]proton.Label{
		"api-label-id": {
			ID:   "api-label-id",
			Name: "Folders",
			Path: []string{""},
			Type: proton.LabelTypeFolder,
		},
	}

	_, err := resolver.ResolveConflict(ctx, apiLabels)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API label")
	assert.Contains(t, err.Error(), "conflicts with internal label")
}

func TestInternalLabelConflictResolver_MailboxFetchError(t *testing.T) {
	ctx := context.Background()

	mockLabelProvider := new(mockLabelNameProvider)
	mockClient := new(mockAPIClient)
	mockIDProvider := new(mockIDProvider)
	mockReporter := new(mockReporter)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Folders"}).
		Return(imap.MailboxData{}, errors.New("database connection error"))

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderFalse{})
	resolver := manager.NewInternalLabelConflictResolver(connectors)

	apiLabels := make(map[string]proton.Label)
	_, err := resolver.ResolveConflict(ctx, apiLabels)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection error")
}

func TestNewInternalLabelConflictResolver_KillSwitchEnabled(t *testing.T) {
	ctx := context.Background()

	mockLabelProvider := new(mockLabelNameProvider)
	mockClient := new(mockAPIClient)
	mockIDProvider := new(mockIDProvider)
	mockReporter := new(mockReporter)

	mockIDProvider.On("GetGluonID", "addr-1").Return("gluon-id-1", true)

	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Folders"}).
		Return(imap.MailboxData{RemoteID: "wrong-folders-id", BridgeName: []string{"Folders"}}, nil)
	mockLabelProvider.On("GetUserMailboxByName", mock.Anything, "gluon-id-1", []string{"Labels"}).
		Return(imap.MailboxData{RemoteID: "wrong-labels-id", BridgeName: []string{"Labels"}}, nil)

	connector := &imapservice.Connector{}
	connector.SetAddrIDTest("addr-1")
	connectors := []*imapservice.Connector{connector}

	manager := imapservice.NewLabelConflictManager(mockLabelProvider, mockIDProvider, mockClient, mockReporter, ffProviderTrue{})
	resolver := manager.NewInternalLabelConflictResolver(connectors)

	apiLabels := map[string]proton.Label{
		"some-api-label": {
			ID:   "some-api-label",
			Name: "SomeLabel",
			Path: []string{"SomeLabel"},
			Type: proton.LabelTypeLabel,
		},
	}

	fn, err := resolver.ResolveConflict(ctx, apiLabels)
	assert.NoError(t, err)

	updates := fn()
	assert.Empty(t, updates)
}
