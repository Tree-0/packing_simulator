import type { SimulationRecording, SimulationsResponse } from './types'

export async function loadSimulations(signal?: AbortSignal): Promise<SimulationRecording[]> {
  const response = await fetch('/api/simulations', {
    headers: { Accept: 'application/json' },
    signal,
  })
  if (!response.ok) {
    throw new Error(`The simulation endpoint returned ${response.status}.`)
  }

  const payload = (await response.json()) as SimulationsResponse
  if (!payload || !Array.isArray(payload.simulations)) {
    throw new Error('The simulation endpoint returned an invalid response.')
  }
  return payload.simulations
}
