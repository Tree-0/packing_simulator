import { useEffect, useMemo, useState } from 'react'
import { usePlayback } from '../hooks/usePlayback'
import type { SimulationRecording } from '../types'
import { PlaybackControls } from './PlaybackControls'
import { SimulationSummaryCard } from './SimulationSummaryCard'
import { SimulationView } from './SimulationView'

type ViewMode = 'full' | 'partial' | 'single'
const PAGE_SIZES = [4, 6, 9, 12] as const

interface SimulationGalleryProps {
  simulations: SimulationRecording[]
}

export function SimulationGallery({ simulations }: SimulationGalleryProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('full')
  const [overviewMode, setOverviewMode] = useState<Exclude<ViewMode, 'single'>>('full')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [pageSize, setPageSize] = useState<number>(6)
  const [pageIndex, setPageIndex] = useState(0)
  const frameCount = useMemo(
    () => simulations.reduce((largest, simulation) => Math.max(largest, simulation.frames.length), 0),
    [simulations],
  )
  const frameDelay = simulations[0]?.frameDelayMs ?? 250
  const playback = usePlayback(frameCount, frameDelay)
  const pageCount = Math.max(1, Math.ceil(simulations.length / pageSize))

  useEffect(() => {
    setSelectedIndex((current) => Math.min(current, Math.max(0, simulations.length - 1)))
  }, [simulations.length])

  useEffect(() => {
    setPageIndex((current) => Math.min(current, pageCount - 1))
  }, [pageCount])

  if (simulations.length === 0) {
    return <div className="empty-state">No simulation recordings are available.</div>
  }

  const setMode = (mode: ViewMode) => {
    if (mode !== 'single') setOverviewMode(mode)
    setViewMode(mode)
  }
  const openSingle = (index: number) => {
    if (viewMode !== 'single') setOverviewMode(viewMode)
    setSelectedIndex(index)
    setViewMode('single')
  }
  const visibleEntries = viewMode === 'partial'
    ? simulations
        .slice(pageIndex * pageSize, (pageIndex + 1) * pageSize)
        .map((recording, offset) => ({ recording, index: pageIndex * pageSize + offset }))
    : simulations.map((recording, index) => ({ recording, index }))
  const selected = simulations[selectedIndex]

  return (
    <main className="simulation-gallery">
      <section className="gallery-toolbar" aria-label="Experiment playback controls">
        <div className="view-controls" role="group" aria-label="View mode">
          {(['full', 'partial', 'single'] as const).map((mode) => (
            <button
              type="button"
              key={mode}
              className={viewMode === mode ? 'is-active' : ''}
              aria-pressed={viewMode === mode}
              onClick={() => setMode(mode)}
            >
              {mode[0].toUpperCase() + mode.slice(1)}
            </button>
          ))}
        </div>

        <PlaybackControls playback={playback} frameCount={frameCount} />

        {viewMode === 'partial' && (
          <div className="page-controls" aria-label="Partial grid pagination">
            <label>
              Runs per page
              <select value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPageIndex(0) }}>
                {PAGE_SIZES.map((size) => <option key={size} value={size}>{size}</option>)}
              </select>
            </label>
            <button type="button" onClick={() => setPageIndex((page) => Math.max(0, page - 1))} disabled={pageIndex === 0}>
              Previous page
            </button>
            <span>Page {pageIndex + 1} of {pageCount}</span>
            <button type="button" onClick={() => setPageIndex((page) => Math.min(pageCount - 1, page + 1))} disabled={pageIndex === pageCount - 1}>
              Next page
            </button>
          </div>
        )}

        {viewMode === 'single' && (
          <div className="single-run-controls" aria-label="Single simulation navigation">
            <button type="button" onClick={() => setMode(overviewMode)}>Back to {overviewMode} view</button>
            <button type="button" aria-label="Previous simulation" onClick={() => setSelectedIndex((index) => Math.max(0, index - 1))} disabled={selectedIndex === 0}>Previous</button>
            <label>
              Simulation
              <select value={selectedIndex} onChange={(event) => setSelectedIndex(Number(event.target.value))}>
                {simulations.map((simulation, index) => (
                  <option key={simulation.id} value={index}>
                    {index + 1}. {simulation.workload || 'Single run'} / {simulation.policy} / {simulation.containerSelector} / seed {simulation.seed}
                  </option>
                ))}
              </select>
            </label>
            <button type="button" aria-label="Next simulation" onClick={() => setSelectedIndex((index) => Math.min(simulations.length - 1, index + 1))} disabled={selectedIndex === simulations.length - 1}>Next</button>
          </div>
        )}
      </section>

      {viewMode === 'single' && selected ? (
        <SimulationView
          recording={selected}
          position={selectedIndex}
          frameIndex={playback.frameIndex}
          isPlaying={playback.isPlaying}
        />
      ) : (
        <div className={`simulation-grid is-${viewMode}`}>
          {visibleEntries.map(({ recording, index }) => (
            <SimulationSummaryCard
              key={recording.id}
              recording={recording}
              position={index}
              frameIndex={playback.frameIndex}
              isPlaying={playback.isPlaying}
              density={viewMode === 'partial' ? 'partial' : 'full'}
              onOpen={() => openSingle(index)}
            />
          ))}
        </div>
      )}
    </main>
  )
}
