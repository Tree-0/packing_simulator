import type { SimulationRecording } from '../types'
import { ContainerGallery, usedBinCount } from './ContainerGallery'

interface SimulationSummaryCardProps {
  recording: SimulationRecording
  position: number
  frameIndex: number
  isPlaying: boolean
  density: 'full' | 'partial'
  onOpen: () => void
}

export function SimulationSummaryCard({
  recording,
  position,
  frameIndex,
  isPlaying,
  density,
  onOpen,
}: SimulationSummaryCardProps) {
  const recordingFrameIndex = Math.min(frameIndex, Math.max(0, recording.frames.length - 1))
  const frame = recording.frames[recordingFrameIndex]

  if (!frame) {
    return (
      <section className="simulation-summary-card empty-recording" aria-label={`Simulation ${recording.id}`}>
        This simulation has no recorded frames.
      </section>
    )
  }

  const timestamp = frame.timestamp === null ? 'Initial' : frame.timestamp
  const playbackLabel = recordingFrameIndex === recording.frames.length - 1
    ? 'Complete'
    : isPlaying
      ? 'Playing'
      : 'Paused'

  return (
    <section
      className={`simulation-summary-card is-${density}`}
      aria-label={`Simulation ${recording.id}`}
      onClick={onOpen}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          onOpen()
        }
      }}
      tabIndex={0}
    >
      <header className="summary-heading">
        <div>
          <p className="eyebrow">Simulation {String(position + 1).padStart(2, '0')}</p>
          <h2>{recording.workload || 'Packing run'}</h2>
        </div>
        <span className={`playback-status ${playbackLabel === 'Playing' ? 'is-playing' : ''}`}>
          <span aria-hidden="true" />
          {playbackLabel}
        </span>
      </header>

      <dl className="summary-identity">
        <div><dt>Placement policy</dt><dd>{recording.policy}</dd></div>
        <div><dt>Selector</dt><dd>{recording.containerSelector}</dd></div>
        <div><dt>Seed</dt><dd>{recording.seed}</dd></div>
      </dl>

      <div className="summary-containers">
        <ContainerGallery recording={recording} frameIndex={recordingFrameIndex} density="compact" />
      </div>

      <dl className="summary-progress">
        <div><dt>Timestamp</dt><dd>{timestamp}</dd></div>
        <div><dt>Placed</dt><dd>{frame.stats.placed}</dd></div>
        <div><dt>Rejected</dt><dd>{frame.stats.rejected}</dd></div>
        <div><dt>Used bins</dt><dd>{usedBinCount(recording, recordingFrameIndex)}</dd></div>
      </dl>
      <span className="open-details">Open details</span>
    </section>
  )
}
