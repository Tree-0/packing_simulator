import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from './App'
import { loadSimulations } from './api'
import { simulationFixture } from './test/fixtures'

vi.mock('./api', () => ({ loadSimulations: vi.fn() }))

const mockedLoadSimulations = vi.mocked(loadSimulations)

describe('App', () => {
  it('shows a loading state while recordings are requested', () => {
    mockedLoadSimulations.mockReturnValue(new Promise(() => undefined))
    render(<App />)
    expect(screen.getByRole('status')).toHaveTextContent('Preparing simulation frames')
  })

  it('renders recordings returned by the API', async () => {
    mockedLoadSimulations.mockResolvedValue([simulationFixture()])
    render(<App />)
    expect(await screen.findByRole('region', { name: 'Simulation simulation-1' })).toBeInTheDocument()
  })

  it('shows API errors and retries', async () => {
    const user = userEvent.setup()
    mockedLoadSimulations
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce([simulationFixture()])

    render(<App />)
    expect(await screen.findByRole('alert')).toHaveTextContent('network unavailable')
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByRole('region', { name: 'Simulation simulation-1' })).toBeInTheDocument()
  })
})
