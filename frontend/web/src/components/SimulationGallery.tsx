import type { SimulationRecording } from '../types'
import { SimulationView } from './SimulationView'

interface SimulationGalleryProps {
  simulations: SimulationRecording[]
}

export function SimulationGallery({ simulations }: SimulationGalleryProps) {
  if (simulations.length === 0) {
    return <div className="empty-state">No simulation recordings are available.</div>
  }

  return (
    <main className="simulation-gallery">
      {simulations.map((simulation, index) => (
        <SimulationView key={simulation.id} recording={simulation} position={index} />
      ))}
    </main>
  )
}
