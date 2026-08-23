import type { EvaluationValue, SimulationFrame, SimulationRecording } from '../types'

interface SimulationDashboardProps {
  recording: SimulationRecording
  frame: SimulationFrame
}

function formatEvaluation(evaluation: EvaluationValue): string {
  if (evaluation.format === 'percent') {
    return `${(evaluation.value * 100).toFixed(1)}%`
  }
  return evaluation.value.toFixed(4)
}

export function SimulationDashboard({ recording, frame }: SimulationDashboardProps) {
  const timestamp = frame.timestamp === null ? 'Initial' : frame.timestamp
  const stats = frame.stats
  const primaryStats = [
    ['Timestamp', timestamp],
    ['Queue', `${frame.queueCount} / ${recording.queueLimit}`],
    ['Iterations', stats.iterations],
    ['Generated', stats.generated],
    ['Placed', stats.placed],
    ['Rotated', stats.rotated],
    ['Rejected', stats.rejected],
    ['Batches', stats.batches],
  ]

  return (
    <aside className="simulation-dashboard" aria-label="Simulation dashboard">
      <div className="run-meta">
        <div>
          <span className="meta-label">Policy</span>
          <strong>{recording.policy}</strong>
        </div>
        <div>
          <span className="meta-label">Seed</span>
          <strong>{recording.seed}</strong>
        </div>
      </div>

      {stats.stoppedEarly && (
        <div className="stopped-notice" role="status">
          Stopped early: no box in the batch could be placed.
        </div>
      )}

      <dl className="stats-grid">
        {primaryStats.map(([label, value]) => (
          <div className="stat" key={label}>
            <dt>{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>

      <div className="evaluation-panel">
        <h3>Evaluations</h3>
        <dl>
          {frame.evaluations.map((evaluation) => (
            <div key={evaluation.name}>
              <dt>{evaluation.name}</dt>
              <dd>{formatEvaluation(evaluation)}</dd>
            </div>
          ))}
        </dl>
      </div>
    </aside>
  )
}
