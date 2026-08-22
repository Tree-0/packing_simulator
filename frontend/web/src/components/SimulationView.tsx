import { usePlayback } from '../hooks/usePlayback'
import type { SimulationRecording } from '../types'
import { PackingCanvas } from './PackingCanvas'
import { PlaybackControls } from './PlaybackControls'
import { SimulationDashboard } from './SimulationDashboard'

interface SimulationViewProps {
  recording: SimulationRecording
  position: number
}

export function SimulationView({ recording, position }: SimulationViewProps) {
  const playback = usePlayback(recording.frames.length, recording.frameDelayMs)
  const frame = recording.frames[playback.frameIndex]

  if (!frame) {
    return (
      <section className="simulation-card empty-recording" aria-label={`Simulation ${recording.id}`}>
        This simulation has no recorded frames.
      </section>
    )
  }

  return (
    <section className="simulation-card" aria-label={`Simulation ${recording.id}`}>
      <header className="simulation-heading">
        <div>
          <p className="eyebrow">Simulation {String(position + 1).padStart(2, '0')}</p>
          <h2>Packing run</h2>
        </div>
        <span className={`playback-status ${playback.isPlaying ? 'is-playing' : ''}`}>
          <span aria-hidden="true" />
          {playback.isPlaying ? 'Playing' : playback.frameIndex === recording.frames.length - 1 ? 'Complete' : 'Paused'}
        </span>
      </header>

      <div className="simulation-layout">
        <div className="visualization-panel">
          <PackingCanvas width={recording.width} height={recording.height} boxes={frame.boxes} />
          <PlaybackControls playback={playback} frameCount={recording.frames.length} />
        </div>
        <SimulationDashboard recording={recording} frame={frame} />
      </div>
    </section>
  )
}
