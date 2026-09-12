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
    evaluations: evaluations.map((evaluation, index) => ({
      ...evaluation,
      value: index === 0 ? generated / 8 : evaluation.value,
    })),
  }
}

export function simulationFixture(id = 'simulation-1'): SimulationRecording {
  return {
    id,
    workload: 'standard',
    policy: 'bottom-left',
    containerSelector: 'first-fit',
    seed: 42,
    queueLimit: 2,
    frameDelayMs: 250,
    containers: [
      {
        id: 1,
        width: 4,
        height: 2,
        createdFrame: 0,
        boxes: [
          { frameIndex: 1, id: 1, x: 0, y: 1, width: 1, height: 1 },
          { frameIndex: 2, id: 2, x: 1, y: 1, width: 1, height: 1 },
        ],
      },
    ],
    frames: [frame(null, 0), frame(0, 1), frame(1, 2)],
  }
}

export function multiContainerFixture(id = 'multi'): SimulationRecording {
  const recording = simulationFixture(id)
  return {
    ...recording,
    containerSelector: 'next-fit',
    containers: [
      {
        id: 1,
        width: 2,
        height: 2,
        createdFrame: 0,
        boxes: [{ frameIndex: 1, id: 1, x: 0, y: 0, width: 2, height: 2 }],
      },
      {
        id: 2,
        width: 2,
        height: 2,
        createdFrame: 2,
        boxes: [{ frameIndex: 2, id: 2, x: 0, y: 0, width: 1, height: 1 }],
      },
    ],
  }
}
