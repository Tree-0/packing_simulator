/*
World state model; Boxes, Container (grid), box queue
*/

package backend

import (
	"errors"
	"fmt"
)

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
	id int
	// slots 0 -> n-1
	height       int
	width        int
	cells        [][]int // Cells[y][x] contains a box ID or EmptyCell
	placements   map[int]BoxPlacement
	occupiedArea int
}

// Read-only wrapper of a Container
// Passed to policies as part of a PolicyContext for
// use in making placement decisions.
type ContainerSnapshot struct {
	id           int // same ID as the container it comes from
	occupiedArea int
	index        occupancyIndex
}

func (c *Container) Id() int {
	return c.id
}

func (s *ContainerSnapshot) Id() int {
	return s.id
}

func (c *Container) Height() int {
	return c.height
}

func (s *ContainerSnapshot) Height() int {
	return s.index.height
}

func (c *Container) Width() int {
	return c.width
}

func (s *ContainerSnapshot) Width() int {
	return s.index.width
}

func (c *Container) Cell(x, y int) (int, error) {
	if x < 0 || y < 0 || x >= c.width || y >= c.height {
		return EmptyCell, errors.New("cell coordinates out of bounds")
	}

	return c.cells[y][x], nil
}

func (c *Container) OccupiedArea() int {
	return c.occupiedArea
}

func (s *ContainerSnapshot) OccupiedArea() int {
	return s.occupiedArea
}

func NewContainer(height, width int, id int) (*Container, error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New("dimensions must be positive")
	}

	cells := make([][]int, height)
	for y := range cells {
		cells[y] = make([]int, width)
	}

	return &Container{
		id:         id,
		height:     height,
		width:      width,
		cells:      cells,
		placements: make(map[int]BoxPlacement),
	}, nil
}

func (c *Container) ContainerSnapshot() ContainerSnapshot {
	if c == nil {
		return ContainerSnapshot{}
	}

	return ContainerSnapshot{
		id:           c.id,
		occupiedArea: c.occupiedArea,
		index:        newOccupancyIndex(c), //value
	}
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

func (s *ContainerSnapshot) CanPlace(box Box, x, y int) bool {
	if s == nil || box.ID < 1 || box.Width <= 0 || box.Height <= 0 {
		return false
	}

	index := s.index

	// Check the top-left coordinate and ensure the box does not extend
	// beyond the right or bottom edges of the container snapshot.
	if x < 0 || y < 0 ||
		x > index.width-box.Width ||
		y > index.height-box.Height {
		return false
	}

	bottom := y + box.Height
	right := x + box.Width
	occupied := index.prefix[bottom][right] -
		index.prefix[y][right] -
		index.prefix[bottom][x] +
		index.prefix[y][x]

	return occupied == 0
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
	c.occupiedArea += box.Height * box.Width

	return nil
}

func (c *Container) CanFitDimensions(width, height int) bool {
	if c == nil || width <= 0 || height <= 0 {
		return false
	}

	return newOccupancyIndex(c).canFitDimensions(width, height)
}

// Like Container.CanFitDimensions, but re-uses existing occupancy index
// for the snapshot, rather than building a new one.
func (s *ContainerSnapshot) CanFitDimensions(width, height int) bool {
	if s == nil || width <= 0 || height <= 0 {
		return false
	}

	return s.index.canFitDimensions(width, height)
}

// occupancyIndex answers "is this rectangular subsection of the container empty?"
// efficiently for one immutable container state using prefix sums.
// It must be rebuilt after the container changes.
type occupancyIndex struct {
	width  int
	height int
	prefix [][]int
}

// publicly exposed wrapper of the index that can be used by other packages,
// such as `evaluator`.
// Snapshots become stale after Container.Place().
type OccupancySnapshot struct {
	index occupancyIndex
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

// Checks whether we can fit a box anywhere in the container.
// The index is built off of a particular container with Container.newOccupancyIndex().
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

func (c *Container) OccupancySnapshot() OccupancySnapshot {
	if c == nil {
		return OccupancySnapshot{}
	}

	return OccupancySnapshot{
		newOccupancyIndex(c),
	}
}

func (s OccupancySnapshot) CanFitDimensions(width, height int) bool {
	return s.index.canFitDimensions(width, height)
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
	nextContainerId int // 1-indexed
	Containers      []*Container
	Queue           BoxQueue
}

func NewWorld(height, width int, queueSize int) (*World, error) {
	if queueSize <= 0 {
		return nil, errors.New("queue size must be positive")
	}

	world := World{
		nextContainerId: 1,
		Containers:      []*Container{},
		Queue: BoxQueue{
			Items: make([]QueuedBox, 0, queueSize),
			Limit: queueSize,
		},
	}

	_, err := world.NewContainer(height, width)
	if err != nil {
		return nil, errors.New("failed to create/append container when initializing a World")
	}

	return &world, nil
}

// Given a container ID, retrieve reference to it. IDs are 1-indexed.
func (w *World) ContainerById(id int) (*Container, error) {
	// currently, container have 1-indexed IDs, are stored sequentially in a slice,
	// and are never removed or deleted during the lifetime of the simulation. Because of this,
	// we can access them directly by indexing with container.id - 1.
	if id < 1 || id > len(w.Containers) {
		return nil, fmt.Errorf(
			"container %d does not exist (out of bounds); must be between 1 and %d",
			id, len(w.Containers),
		)
	}

	return w.Containers[id-1], nil
}

// Adds a new container to the world's list of containers and returns a reference to it.
// Use this function when creating containers that will be added to the world, as it
// manages auto-incrementing containerID state.
func (w *World) NewContainer(height, width int) (*Container, error) {

	container, err := NewContainer(height, width, w.nextContainerId)
	if err != nil {
		return nil, err
	}

	w.Containers = append(w.Containers, container)
	w.nextContainerId += 1

	return container, nil
}

// return a list of snapshots for every container in the world.
func (w *World) ContainerSnapshots() []ContainerSnapshot {
	snapshots := make([]ContainerSnapshot, 0, len(w.Containers))
	for _, container := range w.Containers {
		snapshots = append(snapshots, container.ContainerSnapshot())
	}

	return snapshots
}
