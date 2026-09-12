import { act, fireEvent, render, screen, within } from '@testing-library/react'
import { containersAtFrame } from './ContainerGallery'
import { SimulationGallery } from './SimulationGallery'
import { multiContainerFixture, simulationFixture } from '../test/fixtures'

function statValue(region: HTMLElement, label: string): string | null {
  const term = within(region).getByText(label)
  return term.parentElement?.querySelector('dd')?.textContent ?? null
}

describe('SimulationGallery', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('uses one playhead for every run and holds shorter runs on their final frame', () => {
    const shorter = simulationFixture('shorter')
    shorter.frames = shorter.frames.slice(0, 2)

    render(<SimulationGallery simulations={[simulationFixture('longer'), shorter]} />)
    const longer = screen.getByRole('region', { name: 'Simulation longer' })
    const short = screen.getByRole('region', { name: 'Simulation shorter' })

    expect(statValue(longer, 'Placed')).toBe('0')
    expect(statValue(short, 'Placed')).toBe('0')

    act(() => vi.advanceTimersByTime(250))
    expect(statValue(longer, 'Placed')).toBe('1')
    expect(statValue(short, 'Placed')).toBe('1')

    act(() => vi.advanceTimersByTime(250))
    expect(statValue(longer, 'Placed')).toBe('2')
    expect(statValue(short, 'Placed')).toBe('1')
  })

  it('supports shared pause, stepping, scrubbing, speed, and restart controls', () => {
    render(<SimulationGallery simulations={[simulationFixture()]} />)
    const view = screen.getByRole('region', { name: 'Simulation simulation-1' })

    fireEvent.click(screen.getByRole('button', { name: 'Pause animation' }))
    act(() => vi.advanceTimersByTime(500))
    expect(statValue(view, 'Placed')).toBe('0')

    fireEvent.click(screen.getByRole('button', { name: 'Next frame' }))
    expect(statValue(view, 'Placed')).toBe('1')
    fireEvent.click(screen.getByRole('button', { name: 'Previous frame' }))
    expect(statValue(view, 'Placed')).toBe('0')

    fireEvent.change(screen.getByRole('slider', { name: 'Animation timeline' }), {
      target: { value: '2' },
    })
    expect(statValue(view, 'Placed')).toBe('2')
    fireEvent.change(screen.getByLabelText('Speed'), { target: { value: '2' } })
    expect(screen.getByLabelText('Speed')).toHaveValue('2')

    fireEvent.click(screen.getByRole('button', { name: 'Restart animation' }))
    expect(statValue(view, 'Placed')).toBe('0')
    act(() => vi.advanceTimersByTime(125))
    expect(statValue(view, 'Placed')).toBe('1')
  })

  it('reveals containers and their immutable placements at their recorded frames', () => {
    const recording = multiContainerFixture()
    recording.containers.reverse()
    expect(containersAtFrame(recording, 2).map(({ container }) => container.id)).toEqual([1, 2])

    render(<SimulationGallery simulations={[recording]} />)
    const view = screen.getByRole('region', { name: 'Simulation multi' })
    fireEvent.click(screen.getByRole('button', { name: 'Pause animation' }))

    expect(within(view).getByText('Container 1')).toBeInTheDocument()
    expect(within(view).queryByText('Container 2')).not.toBeInTheDocument()
    expect(statValue(view, 'Used bins')).toBe('0')

    fireEvent.click(screen.getByRole('button', { name: 'Next frame' }))
    expect(within(view).queryByText('Container 2')).not.toBeInTheDocument()
    expect(statValue(view, 'Used bins')).toBe('1')

    fireEvent.click(screen.getByRole('button', { name: 'Next frame' }))
    expect(within(view).getByText('Container 2')).toBeInTheDocument()
    expect(statValue(view, 'Used bins')).toBe('2')
  })

  it('defaults to the full grid and paginates the partial grid', () => {
    const simulations = Array.from({ length: 7 }, (_, index) => simulationFixture(`run-${index + 1}`))
    render(<SimulationGallery simulations={simulations} />)

    expect(screen.getByRole('button', { name: 'Full' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getAllByRole('region', { name: /Simulation run-/ })).toHaveLength(7)

    fireEvent.click(screen.getByRole('button', { name: 'Partial' }))
    expect(screen.getAllByRole('region', { name: /Simulation run-/ })).toHaveLength(6)
    expect(screen.getByText('Page 1 of 2')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Next page' }))
    expect(screen.getByRole('region', { name: 'Simulation run-7' })).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Simulation run-1' })).not.toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Runs per page'), { target: { value: '4' } })
    expect(screen.getAllByRole('region', { name: /Simulation run-/ })).toHaveLength(4)
    expect(screen.getByText('Page 1 of 2')).toBeInTheDocument()
  })

  it('opens cards in single view, navigates runs, and restores the prior overview', () => {
    const simulations = [simulationFixture('first'), simulationFixture('second'), simulationFixture('third')]
    render(<SimulationGallery simulations={simulations} />)

    fireEvent.click(screen.getByRole('region', { name: 'Simulation second' }))
    expect(screen.getByRole('button', { name: 'Single' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('region', { name: 'Simulation second' })).toBeInTheDocument()
    expect(screen.getByText('Placement policy')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Next simulation' }))
    expect(screen.getByRole('region', { name: 'Simulation third' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Previous simulation' }))
    expect(screen.getByRole('region', { name: 'Simulation second' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Back to full view' }))
    expect(screen.getByRole('button', { name: 'Full' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getAllByRole('region', { name: /Simulation (first|second|third)/ })).toHaveLength(3)
  })

  it('shows experiment identity and progress metadata on compact cards', () => {
    render(<SimulationGallery simulations={[simulationFixture()]} />)
    const view = screen.getByRole('region', { name: 'Simulation simulation-1' })

    expect(within(view).getByRole('heading', { name: 'standard' })).toBeInTheDocument()
    expect(statValue(view, 'Placement policy')).toBe('bottom-left')
    expect(statValue(view, 'Selector')).toBe('first-fit')
    expect(statValue(view, 'Seed')).toBe('42')
    expect(statValue(view, 'Timestamp')).toBe('Initial')
    expect(statValue(view, 'Rejected')).toBe('0')
  })
})
