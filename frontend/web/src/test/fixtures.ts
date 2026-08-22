import type { SimulationFrame, SimulationRecording } from '../types'

const evaluations = [
  { name: 'Container utilization', value: 0, format: 'percent' as const },
  { name: 'Container fragmentation', value: 0, format: 'decimal' as const },
  { name: 'Area-weighted fragmentation', value: 0, format: 'decimal' as const },
  { name: 'Compactness', value: 1, format: 'decimal' as const },
  { name: 'Future fit probability', value: 1, format: 'percent' as const },
]

function frame(timestamp: number | null, generated: number): SimulationFrame {
  return {
    timestamp,
    queueCount: generated % 2,
    stats: {
      iterations: generated,
      generated,
      placed: generated,
      rotated: 0,
      rejected: 0,
      batches: Math.floor(generated / 2),
      stoppedEarly: false,
    },
    boxes: Array.from({ length: generated }, (_, index) => ({
      id: index + 1,
      x: index,
      y: 1,
      width: 1,
      height: 1,
    })),
    evaluations: evaluations.map((evaluation, index) => ({
      ...evaluation,
      value: index === 0 ? generated / 8 : evaluation.value,
    })),
  }
}

export function simulationFixture(id = 'simulation-1'): SimulationRecording {
  return {
    id,
    policy: 'bottom-left',
    seed: 42,
    width: 4,
    height: 2,
    queueLimit: 2,
    frameDelayMs: 250,
    frames: [frame(null, 0), frame(0, 1), frame(1, 2)],
  }
}
