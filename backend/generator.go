/*
Random or seeded generator for boxes arriving into the queue at time stamps
*/

package backend

import (
	"errors"
	"fmt"
	"math/rand"
)

type Workload struct {
	ID       int
	Arrivals []QueuedBox
}

type BoxGenerator interface {
	Next(arrivedAt int) QueuedBox
	NextN(arrivedAt int, n int) (*Workload, error)
	BoxDistribution() UniformBoxDistribution
}

type UniformBoxDistribution struct {
	MinWidth, MaxWidth   int
	MinHeight, MaxHeight int
}

type RandomBoxGenerator struct {
	rng              *rand.Rand
	boxDistribution  UniformBoxDistribution
	nextID           int
	nextWorkloadID   int
	allowBoxRotation bool
}

func NewRandomBoxGenerator(
	seed int64,
	minWidth, maxWidth int,
	minHeight, maxHeight int,
	allowBoxRotation bool,
) (*RandomBoxGenerator, error) {
	if minWidth <= 0 || minHeight <= 0 {
		return nil, errors.New("minimum box dimensions must be positive")
	}
	if maxWidth < minWidth || maxHeight < minHeight {
		return nil, errors.New("maximum box dimensions must be at least the minimum dimensions")
	}

	return &RandomBoxGenerator{
		rng: rand.New(rand.NewSource(seed)),
		boxDistribution: UniformBoxDistribution{
			MinWidth:  minWidth,
			MaxWidth:  maxWidth,
			MinHeight: minHeight,
			MaxHeight: maxHeight,
		},
		nextID:           1,
		nextWorkloadID:   1,
		allowBoxRotation: allowBoxRotation,
	}, nil
}

func (g *RandomBoxGenerator) Next(t int) QueuedBox {
	dist := g.boxDistribution
	width := dist.MinWidth + g.rng.Intn(dist.MaxWidth-dist.MinWidth+1)
	height := dist.MinHeight + g.rng.Intn(dist.MaxHeight-dist.MinHeight+1)

	box := Box{
		ID:        g.nextID,
		Width:     width,
		Height:    height,
		CanRotate: g.allowBoxRotation,
	}
	g.nextID++

	return QueuedBox{Box: box, ArrivedAt: t}
}

func (g *RandomBoxGenerator) BoxDistribution() UniformBoxDistribution {
	return g.boxDistribution
}

// Generates the specified number of boxes from the generator
func (g *RandomBoxGenerator) NextN(t int, numBoxes int) (*Workload, error) {
	if numBoxes < 0 {
		return nil, fmt.Errorf("numBoxes must be nonnegative, got %d", numBoxes)
	}

	arrivals := make([]QueuedBox, numBoxes)
	for i := 0; i < numBoxes; i++ {
		// for now, all boxes arrive at the same time when creating a workload
		arrivals[i] = g.Next(t)
	}

	workload := Workload{
		ID:       g.nextWorkloadID,
		Arrivals: arrivals,
	}

	g.nextWorkloadID += 1

	return &workload, nil

}
