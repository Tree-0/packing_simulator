export type EvaluationFormat = 'percent' | 'decimal'

export interface SimulationStats {
  iterations: number
  generated: number
  placed: number
  rotated: number
  rejected: number
  batches: number
  stoppedEarly: boolean
}

export interface PlacedBox {
  id: number
  x: number
  y: number
  width: number
  height: number
}

export interface EvaluationValue {
  name: string
  value: number
  format: EvaluationFormat
}

export interface SimulationFrame {
  timestamp: number | null
  queueCount: number
  stats: SimulationStats
  boxes: PlacedBox[]
  evaluations: EvaluationValue[]
}

export interface SimulationRecording {
  id: string
  policy: string
  seed: number
  width: number
  height: number
  queueLimit: number
  frameDelayMs: number
  frames: SimulationFrame[]
}

export interface SimulationsResponse {
  simulations: SimulationRecording[]
}
