import { PLAYBACK_SPEEDS, type PlaybackController } from '../hooks/usePlayback'

interface PlaybackControlsProps {
  playback: PlaybackController
  frameCount: number
}

export function PlaybackControls({ playback, frameCount }: PlaybackControlsProps) {
  const lastFrame = Math.max(0, frameCount - 1)
  return (
    <div className="playback-controls" aria-label="Playback controls">
      <div className="transport-controls">
        <button type="button" onClick={playback.restart} disabled={frameCount <= 1} aria-label="Restart animation">
          Restart
        </button>
        <button type="button" onClick={playback.previous} disabled={playback.frameIndex === 0} aria-label="Previous frame">
          <span aria-hidden="true">←</span>
        </button>
        <button
          type="button"
          className="play-toggle"
          onClick={playback.isPlaying ? playback.pause : playback.play}
          disabled={frameCount <= 1}
          aria-label={playback.isPlaying ? 'Pause animation' : 'Play animation'}
        >
          {playback.isPlaying ? 'Pause' : 'Play'}
        </button>
        <button type="button" onClick={playback.next} disabled={playback.frameIndex === lastFrame} aria-label="Next frame">
          <span aria-hidden="true">→</span>
        </button>
      </div>

      <div className="timeline-row">
        <input
          className="timeline"
          type="range"
          min="0"
          max={lastFrame}
          value={playback.frameIndex}
          onChange={(event) => playback.seek(Number(event.target.value))}
          aria-label="Animation timeline"
        />
        <output className="frame-progress" aria-live="off">
          {playback.frameIndex + 1} / {frameCount}
        </output>
        <label className="speed-control">
          <span>Speed</span>
          <select value={playback.speed} onChange={(event) => playback.setSpeed(Number(event.target.value))}>
            {PLAYBACK_SPEEDS.map((speed) => (
              <option key={speed} value={speed}>
                {speed}×
              </option>
            ))}
          </select>
        </label>
      </div>
    </div>
  )
}
