import type { ContainerRecording, RecordedBox, SimulationRecording } from '../types'
import { PackingCanvas } from './PackingCanvas'

export interface VisibleContainer {
  container: ContainerRecording
  boxes: RecordedBox[]
}

export function containersAtFrame(
  recording: SimulationRecording,
  frameIndex: number,
): VisibleContainer[] {
  return recording.containers
    .filter((container) => container.createdFrame <= frameIndex)
    .sort((left, right) => left.id - right.id)
    .map((container) => ({
      container,
      boxes: container.boxes.filter((box) => box.frameIndex <= frameIndex),
    }))
}

export function usedBinCount(recording: SimulationRecording, frameIndex: number): number {
  return containersAtFrame(recording, frameIndex).filter(({ boxes }) => boxes.length > 0).length
}

interface ContainerGalleryProps {
  recording: SimulationRecording
  frameIndex: number
  density?: 'detail' | 'compact'
}

export function ContainerGallery({
  recording,
  frameIndex,
  density = 'detail',
}: ContainerGalleryProps) {
  const visibleContainers = containersAtFrame(recording, frameIndex)

  return (
    <div
      className={`container-gallery is-${density}`}
      aria-label={`Containers for simulation ${recording.id}`}
    >
      {visibleContainers.map(({ container, boxes }) => (
        <figure className="container-recording" key={container.id}>
          <figcaption>
            <span>Container {container.id}</span>
            <span>{boxes.length} {boxes.length === 1 ? 'box' : 'boxes'}</span>
          </figcaption>
          <PackingCanvas
            width={container.width}
            height={container.height}
            boxes={boxes}
            density={density}
          />
        </figure>
      ))}
    </div>
  )
}
