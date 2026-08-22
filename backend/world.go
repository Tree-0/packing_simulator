/*
World state model; Boxes, Container (grid), box queue
*/

package backend

import "errors"

const EmptyCell = 0

type Box struct {
	ID        int // >= 1
	Height    int
	Width     int
	CanRotate bool
}

func (b Box) Rotate() Box {
	return Box{
		ID:        b.ID,
		Height:    b.Width,
		Width:     b.Height,
		CanRotate: b.CanRotate,
	}
}

func (b Box) TryRotate() (Box, error) {
	if !b.CanRotate {
		return b, errors.New("box is not rotatable")
	}
	return b.Rotate(), nil
}

type BoxPlacement struct {
	BoxID int // >= 1
	// (X,Y) is the top left corner of the Box
	X int
	Y int
	// Rotated records whether the box was placed with its width and height swapped.
	Rotated bool
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

func (c *Container) Place(box Box, x, y int, rotated bool) error {
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
		BoxID:   box.ID,
		X:       x,
		Y:       y,
		Rotated: rotated,
	}

	return nil
}

func (c *Container) CanFitDimensions(width, height int) bool {
	if c == nil || width <= 0 || height <= 0 {
		return false
	}

	return newOccupancyIndex(c).canFitDimensions(width, height)
}

// occupancyIndex answers "is this rectangular subsection of the container empty?"
// efficiently for one immutable container state using prefix sums.
// It must be rebuilt after the container changes.
type occupancyIndex struct {
	width  int
	height int
	prefix [][]int
}

func newOccupancyIndex(c *Container) occupancyIndex {
	index := occupancyIndex{
		width:  c.width,
		height: c.height,
		prefix: make([][]int, c.height+1),
	}
	for y := range index.prefix {
		index.prefix[y] = make([]int, c.width+1)
	}

	// prefix[y][x] stores the number of occupied cells in the rectangle from
	// (0, 0) up to, but not including, (x, y).
	for y := 1; y <= c.height; y++ {
		for x := 1; x <= c.width; x++ {
			occupied := 0
			if c.cells[y-1][x-1] != EmptyCell {
				occupied = 1
			}

			index.prefix[y][x] = occupied +
				index.prefix[y-1][x] +
				index.prefix[y][x-1] -
				index.prefix[y-1][x-1]
		}
	}

	return index
}

func (index occupancyIndex) canFitDimensions(width, height int) bool {
	if width <= 0 || height <= 0 || height > index.height || width > index.width {
		return false
	}

	for y := 0; y <= index.height-height; y++ {
		for x := 0; x <= index.width-width; x++ {
			bottom := y + height
			right := x + width
			occupied := index.prefix[bottom][right] -
				index.prefix[y][right] -
				index.prefix[bottom][x] +
				index.prefix[y][x]

			if occupied == 0 {
				return true
			}
		}
	}

	return false
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
