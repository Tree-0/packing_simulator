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

export interface RecordedBox {
  frameIndex: number
  id: number
  x: number
  y: number
  width: number
  height: number
}

export interface ContainerRecording {
  id: number
  width: number
  height: number
  createdFrame: number
  boxes: RecordedBox[]
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
  evaluations: EvaluationValue[]
}

export interface SimulationRecording {
  id: string
  workload: string
  policy: string
  containerSelector: string
  seed: number
  queueLimit: number
  frameDelayMs: number
  containers: ContainerRecording[]
  frames: SimulationFrame[]
}

export interface SimulationsResponse {
  simulations: SimulationRecording[]
}
