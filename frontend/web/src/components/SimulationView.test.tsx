import { act, fireEvent, render, screen, within } from '@testing-library/react'
import { SimulationGallery } from './SimulationGallery'
import { SimulationView } from './SimulationView'
import { simulationFixture } from '../test/fixtures'

function statValue(region: HTMLElement, label: string): string | null {
  const term = within(region).getByText(label)
  return term.parentElement?.querySelector('dd')?.textContent ?? null
}

describe('SimulationView', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('autoplays and keeps the dashboard synchronized with the frame', () => {
    render(<SimulationView recording={simulationFixture()} position={0} />)
    const view = screen.getByRole('region', { name: 'Simulation simulation-1' })
    expect(statValue(view, 'Timestamp')).toBe('Initial')
    expect(statValue(view, 'Generated')).toBe('0')

    act(() => vi.advanceTimersByTime(250))
    expect(statValue(view, 'Timestamp')).toBe('0')
    expect(statValue(view, 'Generated')).toBe('1')

    act(() => vi.advanceTimersByTime(250))
    expect(statValue(view, 'Generated')).toBe('2')
    expect(within(view).getByText('Complete')).toBeInTheDocument()
  })

  it('supports pause, stepping, scrubbing, speed, and restart', () => {
    render(<SimulationView recording={simulationFixture()} position={0} />)
    const view = screen.getByRole('region', { name: 'Simulation simulation-1' })

    fireEvent.click(within(view).getByRole('button', { name: 'Pause animation' }))
    act(() => vi.advanceTimersByTime(500))
    expect(statValue(view, 'Generated')).toBe('0')

    fireEvent.click(within(view).getByRole('button', { name: 'Next frame' }))
    expect(statValue(view, 'Generated')).toBe('1')
    fireEvent.click(within(view).getByRole('button', { name: 'Previous frame' }))
    expect(statValue(view, 'Generated')).toBe('0')

    fireEvent.change(within(view).getByRole('slider', { name: 'Animation timeline' }), { target: { value: '2' } })
    expect(statValue(view, 'Generated')).toBe('2')
    fireEvent.change(within(view).getByRole('combobox'), { target: { value: '2' } })
    expect(within(view).getByRole('combobox')).toHaveValue('2')

    fireEvent.click(within(view).getByRole('button', { name: 'Restart animation' }))
    expect(statValue(view, 'Generated')).toBe('0')
    act(() => vi.advanceTimersByTime(125))
    expect(statValue(view, 'Generated')).toBe('1')
  })

  it('gives multiple simulation cards independent playback state', () => {
    render(<SimulationGallery simulations={[simulationFixture('first'), simulationFixture('second')]} />)
    const first = screen.getByRole('region', { name: 'Simulation first' })
    const second = screen.getByRole('region', { name: 'Simulation second' })

    fireEvent.click(within(first).getByRole('button', { name: 'Pause animation' }))
    act(() => vi.advanceTimersByTime(250))

    expect(statValue(first, 'Generated')).toBe('0')
    expect(statValue(second, 'Generated')).toBe('1')
  })
})
