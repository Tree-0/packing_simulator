/*
World state model; Boxes, Container (grid), box queue
*/

package backend

import "errors"

const EmptyCell = 0

type Box struct {
	ID     int // >= 1
	Height int
	Width  int
	// Boxes do not rotate for now, may do so in the future
}

type BoxPlacement struct {
	BoxID int // >= 1
	// (X,Y) is the top left corner of the Box
	X int
	Y int
}

type QueuedBox struct {
	Box       Box
	ArrivedAt int
}

// The grid of placed boxes
type Container struct {
	// slots 0 -> n-1
	height     int
	width      int
	cells      [][]int // Cells[y][x] contains a box ID or EmptyCell
	placements map[int]BoxPlacement
}

func (c *Container) Height() int {
	return c.height
}

func (c *Container) Width() int {
	return c.width
}

func (c *Container) Cell(x, y int) (int, error) {
	if x < 0 || y < 0 || x >= c.width || y >= c.height {
		return EmptyCell, errors.New("cell coordinates out of bounds")
	}

	return c.cells[y][x], nil
}

func NewContainer(height, width int) (*Container, error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New("dimensions must be positive")
	}

	cells := make([][]int, height)
	for y := range cells {
		cells[y] = make([]int, width)
	}

	return &Container{
		height:     height,
		width:      width,
		cells:      cells,
		placements: make(map[int]BoxPlacement),
	}, nil
}

func (c *Container) CanPlace(box Box, x, y int) bool {
	if c == nil || box.ID < 1 || box.Height <= 0 || box.Width <= 0 {
		return false
	}

	// Check the top-left coordinate and ensure the box does not extend
	// beyond the right or bottom edges of the container.
	if x < 0 || y < 0 || x > c.width-box.Width || y > c.height-box.Height {
		return false
	}

	// A box ID represents one placement in the container.
	if _, exists := c.placements[box.ID]; exists {
		return false
	}

	for cellY := y; cellY < y+box.Height; cellY++ {
		for cellX := x; cellX < x+box.Width; cellX++ {
			if c.cells[cellY][cellX] != EmptyCell {
				return false
			}
		}
	}

	return true
}

func (c *Container) Place(box Box, x, y int) error {
	if !c.CanPlace(box, x, y) {
		return errors.New("invalid placement")
	}

	// Fill the cells corresponding to the placement
	for cellY := y; cellY < y+box.Height; cellY++ {
		for cellX := x; cellX < x+box.Width; cellX++ {
			c.cells[cellY][cellX] = box.ID
		}
	}

	c.placements[box.ID] = BoxPlacement{
		BoxID: box.ID,
		X:     x,
		Y:     y,
	}

	return nil
}

// The queue of boxes to be placed into a Container
type BoxQueue struct {
	Items []QueuedBox
	Limit int
}

func (q *BoxQueue) Full() bool {
	return len(q.Items) >= q.Limit
}

func (q *BoxQueue) Enqueue(box QueuedBox) bool {
	if q.Full() {
		return false
	}

	q.Items = append(q.Items, box)
	return true
}

func (q *BoxQueue) Drain() []QueuedBox {
	batch := q.Items
	q.Items = make([]QueuedBox, 0, q.Limit)
	return batch
}

// The full world state model
type World struct {
	Container Container
	Queue     BoxQueue
}

func NewWorld(height, width int, queueSize int) (*World, error) {
	if queueSize <= 0 {
		return nil, errors.New("queue size must be positive")
	}

	container, err := NewContainer(height, width)
	if err != nil {
		return nil, err
	}

	return &World{
		Container: *container,
		Queue: BoxQueue{
			Items: make([]QueuedBox, 0, queueSize),
			Limit: queueSize,
		},
	}, nil
}
