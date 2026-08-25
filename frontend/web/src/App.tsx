import { useCallback, useEffect, useState } from 'react'
import { loadSimulations } from './api'
import { SimulationGallery } from './components/SimulationGallery'
import type { SimulationRecording } from './types'

type LoadState =
  | { status: 'loading' }
  | { status: 'loaded'; simulations: SimulationRecording[] }
  | { status: 'error'; message: string }

export default function App() {
  const [loadKey, setLoadKey] = useState(0)
  const [state, setState] = useState<LoadState>({ status: 'loading' })
  const retry = useCallback(() => setLoadKey((key) => key + 1), [])

  useEffect(() => {
    const controller = new AbortController()
    setState({ status: 'loading' })
    loadSimulations(controller.signal)
      .then((simulations) => setState({ status: 'loaded', simulations }))
      .catch((error: unknown) => {
        if (controller.signal.aborted) return
        const message = error instanceof Error ? error.message : 'Unable to load the simulation.'
        setState({ status: 'error', message })
      })
    return () => controller.abort()
  }, [loadKey])

  return (
    <div className="app-shell">
      <header className="app-header">
        <div>
          <h1>Packing Simulator</h1>
        </div>
        <p className="app-intro">Bin-packing with rolling-horizon heuristics</p>
      </header>

      {state.status === 'loading' && (
        <div className="loading-state" role="status">
          <span className="loading-mark" aria-hidden="true" />
          Preparing simulation frames…
        </div>
      )}
      {state.status === 'error' && (
        <div className="error-state" role="alert">
          <strong>Could not load the visualizer.</strong>
          <span>{state.message}</span>
          <button type="button" onClick={retry}>Retry</button>
        </div>
      )}
      {state.status === 'loaded' && <SimulationGallery simulations={state.simulations} />}
    </div>
  )
}
