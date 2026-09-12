import type { SimulationRecording } from '../types'
import { ContainerGallery } from './ContainerGallery'
import { SimulationDashboard } from './SimulationDashboard'

interface SimulationViewProps {
  recording: SimulationRecording
  position: number
  frameIndex: number
  isPlaying: boolean
}

export function SimulationView({ recording, position, frameIndex, isPlaying }: SimulationViewProps) {
  const recordingFrameIndex = Math.min(frameIndex, Math.max(0, recording.frames.length - 1))
  const frame = recording.frames[recordingFrameIndex]

  if (!frame) {
    return (
      <section className="simulation-card empty-recording" aria-label={`Simulation ${recording.id}`}>
        This simulation has no recorded frames.
      </section>
    )
  }

  const playbackLabel = recordingFrameIndex === recording.frames.length - 1
    ? 'Complete'
    : isPlaying
      ? 'Playing'
      : 'Paused'

  return (
    <section className="simulation-card" aria-label={`Simulation ${recording.id}`}>
      <header className="simulation-heading">
        <div>
          <p className="eyebrow">Simulation {String(position + 1).padStart(2, '0')}</p>
          <h2>Packing run</h2>
        </div>
        <span className={`playback-status ${playbackLabel === 'Playing' ? 'is-playing' : ''}`}>
          <span aria-hidden="true" />
          {playbackLabel}
        </span>
      </header>

      <div className="simulation-layout">
        <div className="visualization-panel">
          <ContainerGallery recording={recording} frameIndex={recordingFrameIndex} />
        </div>
        <SimulationDashboard recording={recording} frame={frame} frameIndex={recordingFrameIndex} />
      </div>
    </section>
  )
}
