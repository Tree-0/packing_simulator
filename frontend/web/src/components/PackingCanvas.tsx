import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import type { PlacedBox } from '../types'

interface PackingCanvasProps {
  width: number
  height: number
  boxes: PlacedBox[]
}

const MIN_CELL_SIZE = 2
const MAX_SCROLLABLE_DIMENSION = 8192
const MAX_FITTED_HEIGHT = 704
const MIN_FITTED_HEIGHT = 240
const CONTROLS_HEIGHT_RESERVE = 150

export interface CanvasSize {
  width: number
  height: number
  viewportHeight: number
  overflowX: boolean
  overflowY: boolean
}

export function calculateCanvasSize(
  gridWidth: number,
  gridHeight: number,
  availableWidth: number,
  availableHeight: number,
): CanvasSize {
  if (gridWidth <= 0 || gridHeight <= 0 || availableWidth <= 0 || availableHeight <= 0) {
    return { width: 0, height: 0, viewportHeight: 0, overflowX: false, overflowY: false }
  }

  const fittedCellSize = Math.min(availableWidth / gridWidth, availableHeight / gridHeight)
  const largestGridDimension = Math.max(gridWidth, gridHeight)
  const minimumReadableCellSize = Math.min(MIN_CELL_SIZE, MAX_SCROLLABLE_DIMENSION / largestGridDimension)
  const cellSize = Math.max(fittedCellSize, minimumReadableCellSize)
  const width = gridWidth * cellSize
  const height = gridHeight * cellSize

  return {
    width,
    height,
    viewportHeight: Math.min(height, availableHeight),
    overflowX: width > availableWidth + 0.5,
    overflowY: height > availableHeight + 0.5,
  }
}

export function colorForBox(id: number): string {
  const hue = Math.round((id * 137.508) % 360)
  return `hsl(${hue} 62% 61%)`
}

export function drawPacking(
  context: CanvasRenderingContext2D,
  gridWidth: number,
  gridHeight: number,
  boxes: PlacedBox[],
  canvasWidth: number,
  canvasHeight: number,
): void {
  const cellSize = Math.min(canvasWidth / gridWidth, canvasHeight / gridHeight)
  const drawingWidth = cellSize * gridWidth
  const drawingHeight = cellSize * gridHeight
  const offsetX = (canvasWidth - drawingWidth) / 2
  const offsetY = (canvasHeight - drawingHeight) / 2

  context.clearRect(0, 0, canvasWidth, canvasHeight)
  context.fillStyle = '#f8fafc'
  context.fillRect(offsetX, offsetY, drawingWidth, drawingHeight)

  context.beginPath()
  context.strokeStyle = '#d9e1ec'
  context.lineWidth = 1
  for (let x = 0; x <= gridWidth; x += 1) {
    const lineX = offsetX + x * cellSize
    context.moveTo(lineX, offsetY)
    context.lineTo(lineX, offsetY + drawingHeight)
  }
  for (let y = 0; y <= gridHeight; y += 1) {
    const lineY = offsetY + y * cellSize
    context.moveTo(offsetX, lineY)
    context.lineTo(offsetX + drawingWidth, lineY)
  }
  context.stroke()

  boxes.forEach((box) => {
    const x = offsetX + box.x * cellSize
    const y = offsetY + box.y * cellSize
    const width = box.width * cellSize
    const height = box.height * cellSize

    context.fillStyle = colorForBox(box.id)
    context.fillRect(x, y, width, height)
    context.strokeStyle = '#344054'
    context.lineWidth = Math.max(1, Math.min(2, cellSize * 0.06))
    context.strokeRect(x, y, width, height)

    if (width >= 26 && height >= 20) {
      context.fillStyle = '#172033'
      context.font = `600 ${Math.max(11, Math.min(16, cellSize * 0.42))}px ui-monospace, monospace`
      context.textAlign = 'center'
      context.textBaseline = 'middle'
      context.fillText(String(box.id), x + width / 2, y + height / 2)
    }
  })
}

export function PackingCanvas({ width, height, boxes }: PackingCanvasProps) {
  const viewportRef = useRef<HTMLDivElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [canvasSize, setCanvasSize] = useState(() => calculateCanvasSize(width, height, 640, 480))

  useLayoutEffect(() => {
    const viewport = viewportRef.current
    if (!viewport) return undefined

    const measure = () => {
      const bounds = viewport.getBoundingClientRect()
      const availableWidth = viewport.clientWidth || bounds.width
      if (availableWidth <= 0) return

      const remainingViewportHeight = window.innerHeight - Math.max(0, bounds.top) - CONTROLS_HEIGHT_RESERVE
      const availableHeight = Math.min(
        MAX_FITTED_HEIGHT,
        Math.max(MIN_FITTED_HEIGHT, remainingViewportHeight),
      )
      const nextSize = calculateCanvasSize(width, height, availableWidth, availableHeight)
      setCanvasSize((current) => {
        if (
          current.width === nextSize.width &&
          current.height === nextSize.height &&
          current.viewportHeight === nextSize.viewportHeight &&
          current.overflowX === nextSize.overflowX &&
          current.overflowY === nextSize.overflowY
        ) {
          return current
        }
        return nextSize
      })
    }

    measure()
    window.addEventListener('resize', measure)
    if (typeof ResizeObserver === 'undefined') {
      return () => window.removeEventListener('resize', measure)
    }
    const observer = new ResizeObserver(measure)
    observer.observe(viewport)
    return () => {
      observer.disconnect()
      window.removeEventListener('resize', measure)
    }
  }, [height, width])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return undefined

    const redraw = () => {
      const bounds = canvas.getBoundingClientRect()
      if (bounds.width === 0 || bounds.height === 0) return

      const pixelRatio = window.devicePixelRatio || 1
      canvas.width = Math.round(bounds.width * pixelRatio)
      canvas.height = Math.round(bounds.height * pixelRatio)
      const context = canvas.getContext('2d')
      if (!context) return
      context.setTransform(pixelRatio, 0, 0, pixelRatio, 0, 0)
      drawPacking(context, width, height, boxes, bounds.width, bounds.height)
    }

    redraw()
    if (typeof ResizeObserver === 'undefined') return undefined
    const observer = new ResizeObserver(redraw)
    observer.observe(canvas)
    return () => observer.disconnect()
  }, [boxes, canvasSize, height, width])

  const scrollable = canvasSize.overflowX || canvasSize.overflowY

  return (
    <div
      ref={viewportRef}
      className="packing-canvas-viewport"
      style={{ height: canvasSize.viewportHeight }}
      tabIndex={scrollable ? 0 : undefined}
      aria-label={scrollable ? 'Scrollable packing grid' : undefined}
    >
      <div
        className="packing-canvas-shell"
        style={{ width: canvasSize.width, height: canvasSize.height }}
      >
        <canvas
          ref={canvasRef}
          className="packing-canvas"
          role="img"
          aria-label={`${width} by ${height} packing grid with ${boxes.length} placed ${boxes.length === 1 ? 'box' : 'boxes'}`}
        />
      </div>
    </div>
  )
}
