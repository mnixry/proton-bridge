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
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/ProtonMail/gluon/async"
	"github.com/ProtonMail/gluon/logging"
	"github.com/ProtonMail/gluon/reporter"
	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/ProtonMail/proton-bridge/v3/internal/services/observability"
	obsMetrics "github.com/ProtonMail/proton-bridge/v3/internal/services/syncservice/observabilitymetrics"
	"github.com/ProtonMail/proton-bridge/v3/internal/unleash"
	"github.com/ProtonMail/proton-bridge/v3/pkg/message"
	"github.com/ProtonMail/proton-bridge/v3/pkg/utils"
	"github.com/bradenaw/juniper/parallel"
	"github.com/bradenaw/juniper/xslices"
	"github.com/sirupsen/logrus"
)

var buildBufferPool = sync.Pool{ //nolint:gochecknoglobals
	New: func() any {
		return new(bytes.Buffer)
	},
}

type BuildRequest struct {
	childJob
	batch []proton.FullMessage
}

type (
	BuildStageInput  = StageInputConsumer[BuildRequest]
	BuildStageOutput = StageOutputProducer[ApplyRequest]
)

// BuildStage is in charge of decrypting and converting the downloaded messages from the previous stage into
// RFC822 compliant messages which can then be sent to the IMAP server.
type BuildStage struct {
	input       BuildStageInput
	output      BuildStageOutput
	maxBuildMem uint64

	panicHandler async.PanicHandler
	log          *logrus.Entry
	reporter     reporter.Reporter

	// Observability
	observabilitySender observability.Sender

	// Feature Flags
	featureFlagProvider unleash.FeatureFlagValueProvider
}

func NewBuildStage(
	input BuildStageInput,
	output BuildStageOutput,
	maxBuildMem uint64,
	panicHandler async.PanicHandler,
	observabilitySender observability.Sender,
	reporter reporter.Reporter,
	featureFlagProvider unleash.FeatureFlagValueProvider,
) *BuildStage {
	return &BuildStage{
		input:               input,
		output:              output,
		maxBuildMem:         maxBuildMem,
		log:                 logrus.WithField("sync-stage", "build"),
		panicHandler:        panicHandler,
		observabilitySender: observabilitySender,
		reporter:            reporter,
		featureFlagProvider: featureFlagProvider,
	}
}

func (b *BuildStage) Run(group *async.Group) {
	group.Once(func(ctx context.Context) {
		logging.DoAnnotated(
			ctx,
			func(ctx context.Context) {
				b.run(ctx)
			},
			logging.Labels{"sync-stage": "build"},
		)
	})
}

func (b *BuildStage) run(ctx context.Context) {
	maxMessagesInParallel := runtime.NumCPU()

	defer b.output.Close()
	for {
		req, err := b.input.Consume(ctx)
		if err != nil {
			if !errors.Is(err, ErrNoMoreInput) && !errors.Is(err, context.Canceled) {
				b.log.WithError(err).Error("Exiting state with error")
			}

			return
		}

		if req.checkCancelled() {
			continue
		}

		if len(req.batch) == 0 {
			// it is possible that if one does a mass delete on another client an entire download batch fails,
			// and we reach this point without any messages to build.
			req.onStageCompleted(ctx)

			if err := b.output.Produce(ctx, ApplyRequest{
				childJob: req.childJob,
				messages: nil,
			}); err != nil {
				err = fmt.Errorf("failed to produce output for next stage: %w", err)
				logrus.Error(err.Error())
				req.job.onError(err)
			}

			continue
		}

		err = req.job.messageBuilder.WithKeys(func(_ *crypto.KeyRing, addrKRs map[string]*crypto.KeyRing) error {
			chunks := chunkSyncBuilderBatch(req.batch, b.maxBuildMem)

			// This stage will split our existing job into many smaller bits. We need to update the Parent Job so
			// that it correctly tracks the lifetime of extra jobs. Additionally, we also need to make sure
			// that only the last chunk contains the metadata to clear the cache.
			chunkedJobs := req.chunkDivide(chunks)

			for idx, chunk := range chunks {
				if chunkedJobs[idx].checkCancelled() {
					// Cancel all other chunks.
					for i := idx + 1; i < len(chunkedJobs); i++ {
						chunkedJobs[i].checkCancelled()
					}

					return nil
				}

				result, err := parallel.MapContext(ctx, maxMessagesInParallel, chunk, func(_ context.Context, msg proton.FullMessage) (BuildResult, error) {
					defer async.HandlePanic(b.panicHandler)

					kr, ok := addrKRs[msg.AddressID]
					if !ok {
						req.job.log.Errorf("Address '%v' on message '%v' does not have an unlocked kerying", msg.AddressID, msg.ID)

						if err := req.job.state.AddFailedMessageID(req.getContext(), msg.ID); err != nil {
							req.job.log.WithError(err).Error("Failed to add failed message ID")
						}

						b.observabilitySender.AddDistinctMetrics(observability.SyncError, obsMetrics.GenerateNoUnlockedKeyringMetric())
						return BuildResult{}, nil
					}

					buf := buildBufferPool.Get().(*bytes.Buffer) //nolint:forcetypeassert
					buf.Reset()

					// Note the buffer grows inside the BuildMessage function
					res, err := req.job.messageBuilder.BuildMessage(req.job.labels, msg, kr, buf)

					buildBufferPool.Put(buf)

					if err != nil {
						if errors.Is(err, message.ErrRNGUnavailable) && !b.featureFlagProvider.GetFlagValue(unleash.RNGServiceNotAvailableSentryCallDisabled) {
							if sentryErr := b.reporter.ReportMessageWithContext("OS RNG unavailable during sync: MIME boundary generation failed",
								reporter.Context{"err": err.Error(), "msgID": msg.ID}); sentryErr != nil {
								b.log.WithError(sentryErr).Error("Failed to report RNG unavailable error to Sentry")
							}
						}

						req.job.log.WithError(err).WithField("msgID", msg.ID).Error("Failed to build message (sync)")

						if err := req.job.state.AddFailedMessageID(req.getContext(), msg.ID); err != nil {
							req.job.log.WithError(err).Error("Failed to add failed message ID")
						}

						b.observabilitySender.AddDistinctMetrics(observability.SyncError, obsMetrics.GenerateFailedToBuildMetric())
						return BuildResult{}, nil
					}

					b.observabilitySender.AddMetrics(obsMetrics.GenerateMessageBuiltSuccessMetric())
					return res, nil
				})
				if err != nil {
					return err
				}

				success := utils.Filter(result, func(t BuildResult) bool {
					return t.Update != nil
				})

				if len(success) > 0 {
					successIDs := xslices.Map(success, func(t BuildResult) string {
						return t.MessageID
					})

					if err := req.job.state.RemFailedMessageID(req.getContext(), successIDs...); err != nil {
						req.job.log.WithError(err).Error("Failed to remove failed message ID")
					}
				}

				outJob := chunkedJobs[idx]

				outJob.onStageCompleted(ctx)

				if err := b.output.Produce(ctx, ApplyRequest{
					childJob: outJob,
					messages: success,
				}); err != nil {
					return fmt.Errorf("failed to produce output for next stage: %w", err)
				}
			}

			return nil
		})
		if err != nil {
			req.job.onError(err)
		}
	}
}

func chunkSyncBuilderBatch(batch []proton.FullMessage, maxMemory uint64) [][]proton.FullMessage {
	var expectedMemUsage uint64
	chunks := make([][]proton.FullMessage, 0, len(batch)/2)
	lastIndex := 0

	// The chunking uses consecutive half-open intervals [a0, b0), [b0, b1),..., [bk, n)
	// If the next message is too large to fit in the memory, we start a new chunk.

	for index, v := range batch {
		var dataSize uint64
		for _, a := range v.Attachments {
			dataSize += uint64(a.Size) //nolint:gosec // disable G115
		}

		// 2x increase for attachment due to extra memory needed for decrypting and writing
		// in memory buffer.
		dataSize *= 2
		dataSize += uint64(len(v.Body))

		nextMemSize := expectedMemUsage + dataSize
		if nextMemSize >= maxMemory {
			if index > lastIndex {
				chunks = append(chunks, copySlice(batch[lastIndex:index]))
				lastIndex = index
			}
			expectedMemUsage = dataSize
		} else {
			expectedMemUsage = nextMemSize
		}
	}

	if lastIndex < len(batch) {
		chunks = append(chunks, copySlice(batch[lastIndex:]))
	}

	return chunks
}

func copySlice[T any](src []T) []T {
	dst := make([]T, len(src))
	copy(dst, src)

	return dst
}
