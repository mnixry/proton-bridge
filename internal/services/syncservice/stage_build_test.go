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

package syncservice

import (
	"context"
	"errors"
	"testing"

	"github.com/ProtonMail/gluon/async"
	"github.com/ProtonMail/gluon/imap"
	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/ProtonMail/proton-bridge/v3/internal/bridge/mocks"
	"github.com/ProtonMail/proton-bridge/v3/internal/sentry"
	"github.com/ProtonMail/proton-bridge/v3/internal/services/observability"
	obsMetrics "github.com/ProtonMail/proton-bridge/v3/internal/services/syncservice/observabilitymetrics"
	"github.com/ProtonMail/proton-bridge/v3/internal/unleash"
	"github.com/bradenaw/juniper/xslices"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestSyncChunkSyncBuilderBatch(t *testing.T) {
	// GODT-2424 - Some messages were not fully built due to a bug in the chunking if the total memory used by the
	// message would be higher than the maximum we allowed.
	const totalMessageCount = 100

	msg := proton.FullMessage{
		Message: proton.Message{
			Attachments: []proton.Attachment{
				{
					Size: int64(8 * Megabyte),
				},
			},
		},
		AttData: nil,
	}

	messages := xslices.Repeat(msg, totalMessageCount)

	chunks := chunkSyncBuilderBatch(messages, 16*Megabyte)

	var totalMessagesInChunks int

	for _, v := range chunks {
		totalMessagesInChunks += len(v)
	}

	require.Equal(t, totalMessagesInChunks, totalMessageCount)
}

func TestSyncChunkSyncBuilderBatch_WithExpectedChunks(t *testing.T) {
	type testCase struct {
		name          string
		messageCount  int
		buildMessages func(messageCount int) []proton.FullMessage
		maxChunkSize  uint64
		getExpected   func(maxMemory uint64, messageCount int) (maxMessagesInChunk int, expectedChunks int)
	}

	testCases := []testCase{
		{
			name:         "no single message exceeds max memory",
			messageCount: 20,
			maxChunkSize: 30 * Megabyte,
			buildMessages: func(messageCount int) []proton.FullMessage {
				messages := make([]proton.FullMessage, messageCount)
				for i := range messages {
					messages[i] = proton.FullMessage{
						Message: proton.Message{
							Attachments: []proton.Attachment{
								{
									Size: int64(1 * Megabyte),
								},
							},
						},
					}
				}
				return messages
			},
			getExpected: func(maxMemory uint64, messageCount int) (maxMessagesInChunk int, expectedChunks int) {
				// Body size is 0
				// each message has 1 attachment of 1MB
				// initial DataSize is 0 + 1MB * 2 = 2MB
				// expectedMemSize = nextMemSize = DataSize = 2MB
				// this grows until nextMemSize >= maxMemory
				// maxMemory is 30MB

				// so maxMessagesInChunk = maxMemory / DataSize - 1 = 30MB / 2MB - 1 = 14
				// expectedChunks = messageCount / maxMessagesInChunk = 20 / 14 = 2
				dataSize := 1 * Megabyte
				dataSize *= 2

				maxMessagesInChunk = int((maxMemory / dataSize) - 1)
				t.Logf("maxMessagesInChunk: %d", maxMessagesInChunk)
				expectedChunks = messageCount / maxMessagesInChunk

				if messageCount%maxMessagesInChunk != 0 {
					expectedChunks++
				}
				return maxMessagesInChunk, expectedChunks
			},
		},
		{
			name:         "one message exceeds max memory",
			messageCount: 20,
			maxChunkSize: 30 * Megabyte,
			buildMessages: func(messageCount int) []proton.FullMessage {
				messages := make([]proton.FullMessage, messageCount)
				for i := range messages {
					if i == 19 {
						messages[i] = proton.FullMessage{
							Message: proton.Message{
								Attachments: []proton.Attachment{
									{
										Size: int64(31 * Megabyte),
									},
								},
							},
						}
						continue
					}

					messages[i] = proton.FullMessage{
						Message: proton.Message{
							Attachments: []proton.Attachment{
								{
									Size: int64(1 * Megabyte),
								},
							},
						},
					}
				}
				return messages
			},
			getExpected: func(maxMemory uint64, messageCount int) (maxMessagesInChunk int, expectedChunks int) {
				// Body size is 0
				// each message has 1 attachment of 1MB
				// initial DataSize is 0 + 1MB * 2 = 2MB
				// expectedMemSize = nextMemSize = DataSize = 2MB
				// this grows until nextMemSize >= maxMemory
				// maxMemory is 30MB
				// so maxMessagesInChunk = maxMemory / DataSize - 1 = 30MB / 2MB - 1 = 14
				// expectedChunks = messageCount / maxMessagesInChunk = 20 / 14 = 2 + 1 because we know one message exceeds max memory

				dataSize := 1 * Megabyte
				dataSize *= 2
				maxMessagesInChunk = int((maxMemory / dataSize) - 1)
				expectedChunks = messageCount / maxMessagesInChunk
				if messageCount%maxMessagesInChunk != 0 {
					expectedChunks++
				}
				return maxMessagesInChunk, expectedChunks + 1 // +1 for the message that exceeds max memory
			},
		},
		{
			name:         "all messages exceed max memory",
			messageCount: 20,
			maxChunkSize: 30 * Megabyte,
			buildMessages: func(messageCount int) []proton.FullMessage {
				messages := make([]proton.FullMessage, messageCount)

				for i := range messages {
					messages[i] = proton.FullMessage{
						Message: proton.Message{
							Attachments: []proton.Attachment{
								{
									Size: int64(31 * Megabyte),
								},
							},
						},
					}
				}
				return messages
			},
			getExpected: func(_ uint64, _ int) (maxMessagesInChunk int, expectedChunks int) {
				// Body size is 0
				// each message has 1 attachment of 31MB
				// initial DataSize is 0 + 31MB * 2 = 62MB
				// expectedMemSize = nextMemSize = DataSize = 62MB
				// on each iteration, expectedMemSize >= maxMemory
				// so maxMessagesInChunk = 1
				// expectedChunks = messageCount = 20
				return 1, 20
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			messages := tc.buildMessages(tc.messageCount)

			chunks := chunkSyncBuilderBatch(messages, tc.maxChunkSize)
			maxMessagesInChunk, expectedChunks := tc.getExpected(tc.maxChunkSize, tc.messageCount)

			require.Equal(t, expectedChunks, len(chunks))

			totalMessagesInChunks := 0
			for _, chunk := range chunks {
				require.LessOrEqual(t, len(chunk), maxMessagesInChunk)
				totalMessagesInChunks += len(chunk)
			}
			require.Equal(t, tc.messageCount, totalMessagesInChunks)
		})
	}
}

func TestBuildStage_SuccessRemovesFailedMessage(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	input := NewChannelConsumerProducer[BuildRequest]()
	output := NewChannelConsumerProducer[ApplyRequest]()

	labels := getTestLabels()

	ctx, cancel := context.WithCancel(context.Background())
	tj := newTestJob(ctx, mockCtrl, "u", labels)

	msg := proton.FullMessage{
		Message: proton.Message{
			MessageMetadata: proton.MessageMetadata{
				ID:        "MSG",
				AddressID: "addrID",
			},
		},
	}

	tj.messageBuilder.EXPECT().WithKeys(gomock.Any()).DoAndReturn(func(f func(*crypto.KeyRing, map[string]*crypto.KeyRing) error) error {
		require.NoError(t, f(nil, map[string]*crypto.KeyRing{
			"addrID": {},
		}))
		return nil
	})

	tj.syncReporter.EXPECT().OnProgress(gomock.Any(), gomock.Eq(int64(10)))

	tj.job.begin()
	childJob := tj.job.newChildJob("f", 10)
	tj.job.end()

	buildResult := BuildResult{
		AddressID: "addrID",
		MessageID: "MSG",
		Update:    &imap.MessageCreated{},
	}

	tj.messageBuilder.EXPECT().BuildMessage(gomock.Eq(labels), gomock.Eq(msg), gomock.Any(), gomock.Any()).Return(buildResult, nil)
	tj.state.EXPECT().RemFailedMessageID(gomock.Any(), gomock.Eq("MSG"))

	observabilityService := mocks.NewMockObservabilitySender(mockCtrl)
	observabilityService.EXPECT().AddMetrics(obsMetrics.GenerateMessageBuiltSuccessMetric())

	stage := NewBuildStage(input, output, 1024, &async.NoopPanicHandler{}, observabilityService, sentry.NullSentryReporter{}, unleash.NewNullUnleashService())

	go func() {
		stage.run(ctx)
	}()

	require.NoError(t, input.Produce(ctx, BuildRequest{childJob: childJob, batch: []proton.FullMessage{msg}}))

	req, err := output.Consume(ctx)
	cancel()
	require.NoError(t, err)
	require.Len(t, req.messages, 1)
	require.Equal(t, buildResult, req.messages[0])
}

func TestBuildStage_BuildFailureIsReportedButDoesNotCancelJob(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	input := NewChannelConsumerProducer[BuildRequest]()
	output := NewChannelConsumerProducer[ApplyRequest]()
	mockObservabilityService := mocks.NewMockObservabilitySender(mockCtrl)

	labels := getTestLabels()

	ctx, cancel := context.WithCancel(context.Background())
	tj := newTestJob(ctx, mockCtrl, "u", labels)

	msg := proton.FullMessage{
		Message: proton.Message{
			MessageMetadata: proton.MessageMetadata{
				ID:        "MSG",
				AddressID: "addrID",
			},
		},
	}

	tj.messageBuilder.EXPECT().WithKeys(gomock.Any()).DoAndReturn(func(f func(*crypto.KeyRing, map[string]*crypto.KeyRing) error) error {
		require.NoError(t, f(nil, map[string]*crypto.KeyRing{
			"addrID": {},
		}))
		return nil
	})

	tj.job.begin()
	childJob := tj.job.newChildJob("f", 10)
	tj.job.end()

	buildError := errors.New("it failed")

	tj.messageBuilder.EXPECT().BuildMessage(gomock.Eq(labels), gomock.Eq(msg), gomock.Any(), gomock.Any()).Return(BuildResult{}, buildError)
	tj.state.EXPECT().AddFailedMessageID(gomock.Any(), gomock.Eq([]string{"MSG"}))

	tj.syncReporter.EXPECT().OnProgress(gomock.Any(), gomock.Eq(int64(10)))

	mockObservabilityService.EXPECT().AddDistinctMetrics(observability.SyncError, obsMetrics.GenerateNoUnlockedKeyringMetric())

	stage := NewBuildStage(input, output, 1024, &async.NoopPanicHandler{}, mockObservabilityService, sentry.NullSentryReporter{}, unleash.NewNullUnleashService())

	go func() {
		stage.run(ctx)
	}()

	require.NoError(t, input.Produce(ctx, BuildRequest{childJob: childJob, batch: []proton.FullMessage{msg}}))

	req, err := output.Consume(ctx)
	cancel()
	require.NoError(t, err)
	require.Empty(t, req.messages)
}

func TestBuildStage_FailedToLocateKeyRingIsReportedButDoesNotFailBuild(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	input := NewChannelConsumerProducer[BuildRequest]()
	output := NewChannelConsumerProducer[ApplyRequest]()

	labels := getTestLabels()

	ctx, cancel := context.WithCancel(context.Background())
	tj := newTestJob(ctx, mockCtrl, "u", labels)

	msg := proton.FullMessage{
		Message: proton.Message{
			MessageMetadata: proton.MessageMetadata{
				ID:        "MSG",
				AddressID: "addrID",
			},
		},
	}

	tj.messageBuilder.EXPECT().WithKeys(gomock.Any()).DoAndReturn(func(f func(*crypto.KeyRing, map[string]*crypto.KeyRing) error) error {
		require.NoError(t, f(nil, map[string]*crypto.KeyRing{}))
		return nil
	})

	tj.job.begin()
	childJob := tj.job.newChildJob("f", 10)
	tj.job.end()

	tj.state.EXPECT().AddFailedMessageID(gomock.Any(), gomock.Eq([]string{"MSG"}))

	tj.syncReporter.EXPECT().OnProgress(gomock.Any(), gomock.Eq(int64(10)))

	observabilitySender := mocks.NewMockObservabilitySender(mockCtrl)
	observabilitySender.EXPECT().AddDistinctMetrics(observability.SyncError)

	stage := NewBuildStage(input, output, 1024, &async.NoopPanicHandler{}, observabilitySender, sentry.NullSentryReporter{}, unleash.NewNullUnleashService())

	go func() {
		stage.run(ctx)
	}()

	require.NoError(t, input.Produce(ctx, BuildRequest{childJob: childJob, batch: []proton.FullMessage{msg}}))

	req, err := output.Consume(ctx)
	cancel()
	require.NoError(t, err)
	require.Empty(t, req.messages)
}

func TestBuildStage_OtherErrorsFailJob(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	input := NewChannelConsumerProducer[BuildRequest]()
	output := NewChannelConsumerProducer[ApplyRequest]()

	labels := getTestLabels()

	ctx, cancel := context.WithCancel(context.Background())
	tj := newTestJob(ctx, mockCtrl, "u", labels)

	msg := proton.FullMessage{
		Message: proton.Message{
			MessageMetadata: proton.MessageMetadata{
				ID:        "MSG",
				AddressID: "addrID",
			},
		},
	}

	expectedErr := errors.New("something went wrong")

	tj.messageBuilder.EXPECT().WithKeys(gomock.Any()).DoAndReturn(func(_ func(*crypto.KeyRing, map[string]*crypto.KeyRing) error) error {
		return expectedErr
	})

	tj.job.begin()
	childJob := tj.job.newChildJob("f", 10)
	tj.job.end()

	stage := NewBuildStage(input, output, 1024, &async.NoopPanicHandler{}, mocks.NewMockObservabilitySender(mockCtrl), sentry.NullSentryReporter{}, unleash.NewNullUnleashService())

	go func() {
		stage.run(ctx)
	}()

	require.NoError(t, input.Produce(ctx, BuildRequest{childJob: childJob, batch: []proton.FullMessage{msg}}))

	err := tj.job.waitAndClose(ctx)
	require.Equal(t, expectedErr, err)

	cancel()

	_, err = output.Consume(context.Background())
	require.ErrorIs(t, err, ErrNoMoreInput)
}

func TestBuildStage_CancelledJobIsDiscarded(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	input := NewChannelConsumerProducer[BuildRequest]()
	output := NewChannelConsumerProducer[ApplyRequest]()

	msg := proton.FullMessage{
		Message: proton.Message{
			MessageMetadata: proton.MessageMetadata{
				ID:        "MSG",
				AddressID: "addrID",
			},
		},
	}

	stage := NewBuildStage(input, output, 1024, &async.NoopPanicHandler{}, mocks.NewMockObservabilitySender(mockCtrl), sentry.NullSentryReporter{}, unleash.NewNullUnleashService())

	ctx, cancel := context.WithCancel(context.Background())

	jobCtx, jobCancel := context.WithCancel(context.Background())

	tj := newTestJob(jobCtx, mockCtrl, "", map[string]proton.Label{})

	tj.job.begin()
	defer tj.job.end()
	childJob := tj.job.newChildJob("f", 10)

	go func() {
		stage.run(ctx)
	}()

	jobCancel()
	require.NoError(t, input.Produce(ctx, BuildRequest{
		childJob: childJob,
		batch:    []proton.FullMessage{msg},
	}))

	go func() { cancel() }()

	_, err := output.Consume(context.Background())
	require.ErrorIs(t, err, ErrNoMoreInput)
}

func TestTask_EmptyInputDoesNotCrash(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	input := NewChannelConsumerProducer[BuildRequest]()
	output := NewChannelConsumerProducer[ApplyRequest]()

	labels := getTestLabels()

	ctx, cancel := context.WithCancel(context.Background())
	tj := newTestJob(ctx, mockCtrl, "u", labels)

	tj.syncReporter.EXPECT().OnProgress(gomock.Any(), gomock.Eq(int64(10)))

	tj.job.begin()
	childJob := tj.job.newChildJob("f", 10)
	tj.job.end()

	stage := NewBuildStage(input, output, 1024, &async.NoopPanicHandler{}, mocks.NewMockObservabilitySender(mockCtrl), sentry.NullSentryReporter{}, unleash.NewNullUnleashService())

	go func() {
		stage.run(ctx)
	}()

	require.NoError(t, input.Produce(ctx, BuildRequest{childJob: childJob, batch: []proton.FullMessage{}}))

	req, err := output.Consume(ctx)
	cancel()
	require.NoError(t, err)
	require.Len(t, req.messages, 0)
}
