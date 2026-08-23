import { calculateCanvasSize, colorForBox, drawPacking } from './PackingCanvas'

function drawingContext() {
  return {
    clearRect: vi.fn(),
    fillRect: vi.fn(),
    beginPath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    stroke: vi.fn(),
    strokeRect: vi.fn(),
    fillText: vi.fn(),
    fillStyle: '',
    strokeStyle: '',
    lineWidth: 0,
    font: '',
    textAlign: 'start',
    textBaseline: 'alphabetic',
  }
}

describe('PackingCanvas drawing helpers', () => {
  it('assigns stable, distinct colors to box IDs', () => {
    expect(colorForBox(4)).toBe(colorForBox(4))
    expect(colorForBox(4)).not.toBe(colorForBox(5))
  })

  it('draws a box as one contiguous rectangle and labels it once', () => {
    const context = drawingContext()
    drawPacking(
      context as unknown as CanvasRenderingContext2D,
      4,
      2,
      [{ id: 7, x: 1, y: 0, width: 2, height: 1 }],
      400,
      200,
    )

    expect(context.fillRect).toHaveBeenCalledWith(100, 0, 200, 100)
    expect(context.strokeRect).toHaveBeenCalledTimes(1)
    expect(context.fillText).toHaveBeenCalledWith('7', 200, 50)
  })

  it('fits ordinary wide and tall containers within both available dimensions', () => {
    expect(calculateCanvasSize(20, 10, 800, 500)).toEqual({
      width: 800,
      height: 400,
      viewportHeight: 400,
      overflowX: false,
      overflowY: false,
    })
    expect(calculateCanvasSize(10, 100, 800, 500)).toEqual({
      width: 50,
      height: 500,
      viewportHeight: 500,
      overflowX: false,
      overflowY: false,
    })
    expect(calculateCanvasSize(100, 10, 800, 500)).toEqual({
      width: 800,
      height: 80,
      viewportHeight: 80,
      overflowX: false,
      overflowY: false,
    })
  })

  it('uses an internal scrollbar instead of making extreme cells illegible', () => {
    expect(calculateCanvasSize(10, 1000, 800, 600)).toEqual({
      width: 20,
      height: 2000,
      viewportHeight: 600,
      overflowX: false,
      overflowY: true,
    })
    expect(calculateCanvasSize(1000, 10, 800, 600)).toEqual({
      width: 2000,
      height: 20,
      viewportHeight: 20,
      overflowX: true,
      overflowY: false,
    })
  })
})
